package agent

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nyxora/nyxora/internal/config"
	"github.com/nyxora/nyxora/internal/dashboard"
	"github.com/nyxora/nyxora/internal/failover"
	"github.com/nyxora/nyxora/internal/monitor"
	"github.com/nyxora/nyxora/internal/packager"
	"github.com/nyxora/nyxora/internal/routing"
	"github.com/nyxora/nyxora/internal/transport"
)

type Agent struct {
	cfg         *config.Config
	transportM  *transport.Manager
	mon         *monitor.Monitor
	routeEngine *routing.Engine
	fail        *failover.Failover
	pkg         *packager.Packager
	tui         *dashboard.TUI
	mu          sync.Mutex
	running     bool
	remoteAddr  string
	connected   bool
}

func New(cfg *config.Config) *Agent {
	return &Agent{
		cfg:         cfg,
		transportM:  transport.NewManager(cfg.AllTunnelsActive),
		mon:         monitor.NewMonitor(cfg.MonitorInterval),
		routeEngine: routing.NewEngine(),
		fail:        failover.NewFailover(cfg.FailoverInterval),
		pkg:         packager.NewPackager(cfg.DataDir),
		tui:         dashboard.NewTUI(2),
	}
}

func (a *Agent) Init() error {
	log.Printf("[agent] initializing nyxora agent v0.1.0")
	log.Printf("[agent] data dir: %s", a.cfg.DataDir)

	if err := os.MkdirAll(a.cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	transports := []transport.Transport{
		transport.NewWireGuard(),
		transport.NewQUIC(),
		transport.NewSSH(),
		transport.NewTCP(),
	}

	for _, t := range transports {
		a.transportM.Register(t)
	}

	a.fail.OnFailover(func(from, to string) {
		log.Printf("[agent] *** FAILOVER: %s -> %s ***", from, to)
		a.routeEngine.SetCurrent(to)
		a.transportM.DisconnectAll()
		cfg := map[string]string{}
		if t, ok := a.transportM.Get(to); ok {
			t.Init(cfg)
			if err := t.Connect(a.remoteAddr); err != nil {
				log.Printf("[agent] failover connect error: %v", err)
			}
		}
	})

	a.fail.OnRecover(func(name string) {
		log.Printf("[agent] *** RECOVER: %s healthy again ***", name)
	})

	log.Printf("[agent] initialized with %d transports", len(transports))
	return nil
}

func (a *Agent) Start() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("agent already running")
	}
	a.running = true
	a.mu.Unlock()

	log.Printf("[agent] starting nyxora agent")

	a.tui.SetProvider(a)
	if err := a.tui.Start(); err != nil {
		log.Printf("[agent] tui start error: %v", err)
	}

	go a.fail.Start()
	go a.monitorLoop()

	if a.connected {
		go a.qualityLoop()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("[agent] shutting down...")
	a.Stop()
	return nil
}

func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.running {
		return
	}
	a.running = false
	a.connected = false
	a.fail.Stop()
	a.mon.Stop()
	a.tui.Stop()
	a.transportM.DisconnectAll()
	log.Printf("[agent] stopped")
}

func (a *Agent) Connect(remoteAddr string) error {
	a.remoteAddr = remoteAddr
	log.Printf("[agent] connecting to %s", remoteAddr)

	if a.cfg.ServerMode {
		log.Printf("[agent] server mode enabled, accepting connections")
		return a.startServer()
	}

	if err := a.transportM.ConnectAll(remoteAddr); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	t := a.transportM.ActiveTransport()
	if t != nil {
		a.routeEngine.SetCurrent(t.Name())
		metrics := t.Metrics()
		a.routeEngine.Update(t.Name(), t.Type(), metrics.LatencyMs, metrics.JitterMs, metrics.PacketLoss, metrics.Stability, metrics.Bandwidth)
		a.fail.Update(t.Name(), metrics.LatencyMs, metrics.PacketLoss)
	}

	a.connected = true
	return nil
}

func (a *Agent) Disconnect() error {
	log.Printf("[agent] disconnecting")
	a.connected = false
	a.transportM.DisconnectAll()
	return nil
}

func (a *Agent) startServer() error {
	log.Printf("[agent] server listening on %s", a.cfg.ListenAddr)
	return nil
}

func (a *Agent) qualityLoop() {
	for {
		if !a.running || !a.connected {
			return
		}

		for _, info := range a.transportM.List() {
			a.routeEngine.Update(info.Name, info.Type, info.Latency, info.Jitter, info.Loss, info.Stability, info.Bandwidth)
			a.fail.Update(info.Name, info.Latency, info.Loss)
		}

		best := a.routeEngine.BestPath()
		current := a.routeEngine.Current()

		if best != nil && best.Name != current && current != "" {
			diff := best.Score - a.getCurrentScore(current)
			if diff > 15 {
				log.Printf("[agent] triggering failover: %s -> %s", current, best.Name)
				if cb := a.fail.GetOnFailover(); cb != nil {
					cb(current, best.Name)
				}
			}
		}

		time.Sleep(time.Duration(a.cfg.MonitorInterval) * time.Second)
	}
}

func (a *Agent) getCurrentScore(name string) float64 {
	for _, p := range a.routeEngine.AllPaths() {
		if p.Name == name {
			return p.Score
		}
	}
	return 0
}

func (a *Agent) monitorLoop() {
	for {
		if !a.running {
			return
		}
		for _, info := range a.transportM.List() {
			log.Printf("[agent] %s: score=%.1f latency=%.1fms loss=%.1f%%",
				info.Name, info.Score, info.Latency, info.Loss)
		}
		time.Sleep(10 * time.Second)
	}
}

func (a *Agent) Status() map[string]interface{} {
	status := map[string]interface{}{
		"running":          a.running,
		"connected":        a.connected,
		"server_mode":      a.cfg.ServerMode,
		"all_active":       a.cfg.AllTunnelsActive,
		"active_transport": a.transportM.Active(),
		"remote_addr":      a.remoteAddr,
	}

	var transports []map[string]interface{}
	for _, info := range a.transportM.List() {
		transports = append(transports, map[string]interface{}{
			"name":      info.Name,
			"type":      info.Type,
			"status":    info.Status,
			"score":     info.Score,
			"latency":   info.Latency,
			"jitter":    info.Jitter,
			"loss":      info.Loss,
			"stable":    info.Stability,
			"bandwidth": info.Bandwidth,
		})
	}
	status["transports"] = transports

	best := a.routeEngine.BestPath()
	if best != nil {
		status["best_path"] = best.Name
		status["best_score"] = best.Score
	}

	failoverStatus := make(map[string]string)
	for name, s := range a.fail.AllStatus() {
		switch s {
		case failover.StatusHealthy:
			failoverStatus[name] = "healthy"
		case failover.StatusDegraded:
			failoverStatus[name] = "degraded"
		case failover.StatusDown:
			failoverStatus[name] = "down"
		}
	}
	status["failover"] = failoverStatus

	return status
}

func (a *Agent) RunDashboard() error {
	a.tui.SetProvider(a)
	return a.tui.Start()
}
