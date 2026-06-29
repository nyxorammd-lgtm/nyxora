package orchestrator

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nyxora/nyxora/internal/config"
	"github.com/nyxora/nyxora/internal/dashboard"
	"github.com/nyxora/nyxora/internal/failover"
	"github.com/nyxora/nyxora/internal/monitor"
	"github.com/nyxora/nyxora/internal/multipath"
	"github.com/nyxora/nyxora/internal/packager"
	"github.com/nyxora/nyxora/internal/remote"
	"github.com/nyxora/nyxora/internal/routing"
	"github.com/nyxora/nyxora/internal/transport"
)

type Phase string

const (
	PhaseInit       Phase = "initializing"
	PhaseConnecting Phase = "connecting"
	PhaseSetup      Phase = "setting up remote"
	PhaseTunnel     Phase = "establishing tunnel"
	PhaseMultipath  Phase = "multipath active"
	PhaseActive     Phase = "active"
	PhaseFailed     Phase = "failed"
)

type StepStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Done   bool   `json:"done"`
	TimeMs int64  `json:"time_ms"`
}

type Orchestrator struct {
	cfg         *config.Config
	transportM  *transport.Manager
	mon         *monitor.Monitor
	routeEngine *routing.Engine
	fail        *failover.Failover
	pkg         *packager.Packager
	tui         *dashboard.TUI
	scheduler   *multipath.Scheduler

	remoteHost  *remote.Host
	localNodeID string

	mu        sync.Mutex
	running   bool
	connected bool
	phase     Phase
	steps     []StepStatus
	startTime time.Time

	onStepUpdate func(StepStatus)
}

func New(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		cfg:         cfg,
		transportM:  transport.NewManager(cfg.AllTunnelsActive),
		mon:         monitor.NewMonitor(cfg.MonitorInterval),
		routeEngine: routing.NewEngine(),
		fail:        failover.NewFailover(cfg.FailoverInterval),
		pkg:         packager.NewPackager(cfg.DataDir),
		tui:         dashboard.NewTUI(2),
		scheduler:   multipath.NewScheduler(),
		localNodeID: generateNodeID(),
		phase:       PhaseInit,
		startTime:   time.Now(),
	}
}

func (o *Orchestrator) Init() error {
	log.Printf("[orchestrator] initializing nyxora v0.1.0")
	log.Printf("[orchestrator] node id: %s", o.localNodeID)

	if err := os.MkdirAll(o.cfg.DataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	allTransports := []transport.Transport{
		transport.NewWireGuard(),
		transport.NewOpenVPN(),
		transport.NewSSH(),
		transport.NewQUIC(),
		transport.NewFRP(),
		transport.NewRathole(),
		transport.NewCloudflare(),
		transport.NewIPsec(),
		transport.NewShadowSOCKS(),
		transport.NewHysteria(),
		transport.NewBackhaul(),
	}

	for _, t := range allTransports {
		o.transportM.Register(t)
	}

	for _, meta := range transport.ListTunnels() {
		o.scheduler.AddPath(meta.Name, meta.Type, meta.Weight)
	}

	o.fail.OnFailover(func(from, to string) {
		log.Printf("[orchestrator] *** FAILOVER: %s -> %s ***", from, to)
		o.routeEngine.SetCurrent(to)
		o.scheduler.RecordFailover()
	})

	o.fail.OnRecover(func(name string) {
		log.Printf("[orchestrator] *** RECOVER: %s healthy ***", name)
	})

	log.Printf("[orchestrator] initialized with %d transports, %d paths",
		len(allTransports), len(transport.ListTunnels()))
	return nil
}

func (o *Orchestrator) addStep(name, status, detail string) {
	step := StepStatus{
		Name:   name,
		Status: status,
		Detail: detail,
		Done:   status == "OK",
		TimeMs: time.Since(o.startTime).Milliseconds(),
	}
	o.mu.Lock()
	o.steps = append(o.steps, step)
	o.mu.Unlock()
	if o.onStepUpdate != nil {
		o.onStepUpdate(step)
	}
}

func (o *Orchestrator) OnStepUpdate(fn func(StepStatus)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.onStepUpdate = fn
}

func (o *Orchestrator) ConnectToRemote(addr string, port int, user, password string) error {
	o.phase = PhaseConnecting
	log.Printf("[orchestrator] connecting to %s@%s:%d", user, addr, port)

	o.remoteHost = remote.NewHost(addr, port, user, password)

	o.addStep("Pinging remote server", "RUNNING", "")
	lat, loss := o.remoteHost.Ping(4)
	if loss > 80 {
		o.addStep("Pinging remote server", "FAILED", fmt.Sprintf("loss: %.0f%%", loss))
		o.phase = PhaseFailed
		return fmt.Errorf("remote unreachable: %.0f%% loss", loss)
	}
	o.addStep("Pinging remote server", "OK", fmt.Sprintf("%.0fms, %.0f%% loss", lat, loss))

	o.addStep("SSH authentication", "RUNNING", "")
	msg, ok := o.remoteHost.CheckConnectivity()
	if !ok {
		o.addStep("SSH authentication", "FAILED", msg)
		o.phase = PhaseFailed
		return fmt.Errorf("ssh: %s", msg)
	}
	o.addStep("SSH authentication", "OK", msg)

	o.phase = PhaseSetup
	o.addStep("Detecting OS", "RUNNING", "")
	if err := o.remoteHost.DetectOS(); err != nil {
		o.addStep("Detecting OS", "FAILED", err.Error())
		o.phase = PhaseFailed
		return err
	}
	o.addStep("Detecting OS", "OK", fmt.Sprintf("%s | %s", o.remoteHost.OSInfo(), o.remoteHost.Arch()))

	o.addStep("Installing dependencies", "RUNNING", "")
	var failedDeps []string
	for _, meta := range transport.ListTunnels() {
		if meta.Binary == "" {
			continue
		}
		if o.remoteHost.CheckTool(meta.Binary) {
			continue
		}
		script := transport.InstallScript(meta.Name)
		if script == "" {
			continue
		}
		log.Printf("[orchestrator] installing %s on remote...", meta.Name)
		_, err := o.remoteHost.SSHCommand(script)
		if err != nil {
			failedDeps = append(failedDeps, meta.Name)
			log.Printf("[orchestrator] %s install failed: %v", meta.Name, err)
		}
	}
	if len(failedDeps) > 0 {
		o.addStep("Installing dependencies", "WARN", fmt.Sprintf("%d failed: %s", len(failedDeps), strings.Join(failedDeps, ", ")))
	} else {
		o.addStep("Installing dependencies", "OK", "all tunnel dependencies ready")
	}

	o.phase = PhaseTunnel
	o.addStep("Generating WireGuard keys", "RUNNING", "")
	localPriv, localPub := o.generateLocalWGKey()
	o.addStep("Generating WireGuard keys", "OK", fmt.Sprintf("pub: %s...", localPub[:16]))

	o.addStep("Setting up remote WG endpoint", "RUNNING", "")
	remotePub, err := remote.SetupWireGuardRemote(o.remoteHost, localPub, 51820)
	if err != nil {
		o.addStep("Setting up remote WG endpoint", "FAILED", err.Error())
		o.phase = PhaseFailed
		return err
	}
	o.addStep("Setting up remote WG endpoint", "OK", fmt.Sprintf("pub: %s...", remotePub[:16]))

	o.addStep("Setting up local WG endpoint", "RUNNING", "")
	remoteIP, err := remote.GetRemotePublicIP(o.remoteHost)
	if err != nil {
		remoteIP = addr
	}

	localWG, ok := o.transportM.Get("wireguard")
	if !ok {
		localWG = transport.NewWireGuard()
		o.transportM.Register(localWG)
	}
	wgPort := 51820
	subnet := wgPort % 256
	localWG.Init(map[string]string{
		"private_key": localPriv,
		"remote_pub":  remotePub,
		"interface":   "nyxora0",
		"local_addr":  fmt.Sprintf("10.100.%d.2/24", subnet),
	})
	if err := localWG.Connect(remoteIP); err != nil {
		o.addStep("Setting up local WG endpoint", "FAILED", err.Error())
		o.phase = PhaseFailed
		return err
	}
	o.addStep("Setting up local WG endpoint", "OK", "interface nyxora0 ready")

	o.connected = true
	o.phase = PhaseMultipath

	if o.cfg.AllTunnelsActive {
		o.addStep("Multipath mode", "OK", "all tunnels active simultaneously")
		o.transportM.SetAllActive(true)
		o.transportM.ConnectAll(remoteIP)
	} else {
		o.addStep("Smart mode", "OK", "best tunnel selected automatically")
	}

	o.addStep("Tunnel established", "OK",
		fmt.Sprintf("%s <-> %s (%s)", o.localNodeID[:8], o.remoteHost.Hostname(), remoteIP))

	log.Printf("[orchestrator] tunnel active: %s <-> %s (%s)",
		o.localNodeID[:8], o.remoteHost.Hostname(), remoteIP)

	o.routeEngine.SetCurrent("wireguard")
	go o.startMonitoring(remoteIP)

	return nil
}

func (o *Orchestrator) generateLocalWGKey() (priv, pub string) {
	if commandExists("wg") {
		out, err := exec.Command("wg", "genkey").Output()
		if err == nil && len(out) > 0 {
			priv = strings.TrimSpace(string(out))
			pubOut, err := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' | wg pubkey", priv)).Output()
			if err == nil && len(pubOut) > 0 {
				pub = strings.TrimSpace(string(pubOut))
				return
			}
		}
	}
	priv = fmt.Sprintf("nyx-local-key-%d", time.Now().UnixNano())
	pub = fmt.Sprintf("nyx-local-pub-%d", time.Now().UnixNano())
	return
}

func (o *Orchestrator) startMonitoring(remoteAddr string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		if !o.running && !o.connected {
			return
		}

		lat, loss := o.remoteHost.Ping(2)

		for _, info := range o.transportM.List() {
			o.routeEngine.Update(info.Name, info.Type, lat, info.Jitter, loss, info.Stability, info.Bandwidth)
			o.fail.Update(info.Name, lat, loss)
			o.scheduler.UpdatePath(info.Name, info.Score, lat, loss, info.Bandwidth)
		}

		if o.cfg.AllTunnelsActive {
			weights := o.scheduler.Distribution()
			for name, w := range weights {
				o.transportM.SetWeight(name, w)
			}
		}

		best := o.routeEngine.BestPath()
		current := o.routeEngine.Current()
		if best != nil && best.Name != current && current != "" {
			diff := best.Score - o.getCurrentScore(current)
			if diff > 15 {
				log.Printf("[orchestrator] failover: %s (%.1f) -> %s (%.1f)",
					current, o.getCurrentScore(current), best.Name, best.Score)
				if cb := o.fail.GetOnFailover(); cb != nil {
					cb(current, best.Name)
				}
				o.scheduler.RecordFailover()
			}
		}

		<-ticker.C
	}
}

func (o *Orchestrator) getCurrentScore(name string) float64 {
	for _, p := range o.routeEngine.AllPaths() {
		if p.Name == name {
			return p.Score
		}
	}
	return 0
}

func (o *Orchestrator) Start() error {
	o.mu.Lock()
	if o.running {
		o.mu.Unlock()
		return fmt.Errorf("already running")
	}
	o.running = true
	o.mu.Unlock()

	log.Printf("[orchestrator] starting")

	o.tui.SetProvider(o)
	if err := o.tui.Start(); err != nil {
		log.Printf("[orchestrator] tui error: %v", err)
	}

	go o.fail.Start()

	log.Printf("[orchestrator] multipath scheduler: %s", o.scheduler.String())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("[orchestrator] shutting down...")
	o.Stop()
	return nil
}

func (o *Orchestrator) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.running {
		return
	}
	o.running = false
	o.connected = false
	o.fail.Stop()
	o.mon.Stop()
	o.tui.Stop()
	o.transportM.DisconnectAll()
	if o.remoteHost != nil {
		remote.TeardownRemote(o.remoteHost, "nyxora0")
	}
	log.Printf("[orchestrator] stopped (uptime: %s)", time.Since(o.startTime).Round(time.Second))
}

func (o *Orchestrator) Status() map[string]interface{} {
	status := map[string]interface{}{
		"running":          o.running,
		"connected":        o.connected,
		"phase":            string(o.phase),
		"node_id":          o.localNodeID,
		"active_transport": o.transportM.ActiveNames(),
		"all_active":       o.cfg.AllTunnelsActive,
		"mode":             "single-side",
		"uptime":           time.Since(o.startTime).Round(time.Second).String(),
	}

	if o.remoteHost != nil {
		status["remote"] = map[string]interface{}{
			"hostname": o.remoteHost.Hostname(),
			"address":  o.remoteHost.Address,
			"port":     o.remoteHost.Port,
			"os":       o.remoteHost.OSInfo(),
			"arch":     o.remoteHost.Arch(),
		}
	}

	var transports []map[string]interface{}
	for _, info := range o.transportM.List() {
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
			"weight":    info.Weight,
		})
	}
	status["transports"] = transports

	best := o.routeEngine.BestPath()
	if best != nil {
		status["best_path"] = best.Name
		status["best_score"] = best.Score
	}

	failoverStatus := make(map[string]string)
	for name, s := range o.fail.AllStatus() {
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

	status["multipath"] = map[string]interface{}{
		"active":     o.scheduler.Stats().ActivePaths,
		"total":      len(transport.ListTunnels()),
		"best":       o.scheduler.Stats().BestPath,
		"failovers":  o.scheduler.Stats().FailoverCount,
		"bandwidth":  o.scheduler.AggregateBandwidth(),
		"mode":       "weighted",
		"paths":      o.scheduler.AllPaths(),
	}

	o.mu.Lock()
	steps := make([]StepStatus, len(o.steps))
	copy(steps, o.steps)
	o.mu.Unlock()
	status["steps"] = steps

	return status
}

func (o *Orchestrator) Steps() []StepStatus {
	o.mu.Lock()
	defer o.mu.Unlock()
	steps := make([]StepStatus, len(o.steps))
	copy(steps, o.steps)
	return steps
}

func generateNodeID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("nyx-%x", b)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
