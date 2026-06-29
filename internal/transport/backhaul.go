package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
	"time"
)

type Backhaul struct {
	mu         sync.RWMutex
	name       string
	status     Status
	metrics    *Metrics
	remoteAddr string
	port       int
	token      string
	transport  string
	cmd        *exec.Cmd
	remoteCmd  *exec.Cmd
}

func NewBackhaul() *Backhaul {
	return &Backhaul{
		name:      "backhaul",
		status:    StatusInactive,
		metrics:   &Metrics{},
		port:      3080,
		token:     fmt.Sprintf("nyxora-bh-%d", time.Now().Unix()),
		transport: "tcp",
	}
}

func (b *Backhaul) Name() string { return b.name }
func (b *Backhaul) Type() string { return "backhaul" }

func (b *Backhaul) Init(cfg map[string]string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &b.port)
	}
	if t, ok := cfg["token"]; ok {
		b.token = t
	}
	if tr, ok := cfg["backhaul_transport"]; ok {
		b.transport = tr
	}
	log.Printf("[backhaul] initialized (port: %d, transport: %s)", b.port, b.transport)
	return nil
}

func (b *Backhaul) Connect(remoteAddr string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.remoteAddr = remoteAddr
	b.status = StatusTesting
	b.updateMetrics()

	if b.metrics.PacketLoss > 80 {
		b.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", b.metrics.PacketLoss)
	}

	if !commandExists("backhaul") {
		log.Printf("[backhaul] binary not found locally, ping score only")
		b.status = StatusActive
		return nil
	}

	serverCfg := fmt.Sprintf(`[server]
bind_addr = "0.0.0.0:%d"
transport = "%s"
token = "%s"
keepalive_period = 75
nodelay = true
heartbeat = 40
channel_size = 2048
log_level = "error"
ports = []
`, b.port, b.transport, b.token)

	serverCfgPath := "/etc/nyxora/backhaul-server.toml"
	if err := writeConfig(serverCfgPath, serverCfg); err != nil {
		log.Printf("[backhaul] server config error: %v", err)
		b.status = StatusFailed
		return err
	}

	go func() {
		cmd := exec.Command("backhaul", "-c", serverCfgPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[backhaul] server start error: %v", err)
			return
		}
		b.mu.Lock()
		b.remoteCmd = cmd
		b.mu.Unlock()
		cmd.Wait()
	}()

	clientCfg := fmt.Sprintf(`[client]
remote_addr = "%s:%d"
transport = "%s"
token = "%s"
connection_pool = 4
keepalive_period = 75
dial_timeout = 10
nodelay = true
retry_interval = 3
log_level = "error"
`, remoteAddr, b.port, b.transport, b.token)

	clientCfgPath := "/etc/nyxora/backhaul-client.toml"
	if err := writeConfig(clientCfgPath, clientCfg); err != nil {
		log.Printf("[backhaul] client config error: %v", err)
		b.status = StatusFailed
		return err
	}

	go func() {
		cmd := exec.Command("backhaul", "-c", clientCfgPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[backhaul] client start error: %v", err)
			return
		}
		b.mu.Lock()
		b.cmd = cmd
		b.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[backhaul] connecting %s to %s:%d (%s)", b.transport, remoteAddr, b.port, b.transport)
	b.status = StatusActive
	return nil
}

func (b *Backhaul) Disconnect() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd != nil && b.cmd.Process != nil {
		b.cmd.Process.Kill()
		b.cmd = nil
	}
	if b.remoteCmd != nil && b.remoteCmd.Process != nil {
		b.remoteCmd.Process.Kill()
		b.remoteCmd = nil
	}
	exec.Command("pkill", "-f", "backhaul.*nyxora").Run()
	b.status = StatusInactive
	log.Printf("[backhaul] disconnected")
	return nil
}

func (b *Backhaul) Status() Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

func (b *Backhaul) Metrics() *Metrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	m := *b.metrics
	return &m
}

func (b *Backhaul) Health() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status == StatusActive
}

func (b *Backhaul) Score() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.updateMetrics()
	m := b.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/2)*0.30 + math.Max(0, 100-m.PacketLoss*2)*0.30 +
		math.Max(0, 100-m.JitterMs*3)*0.15 + (m.Stability*100)*0.25
}

func (b *Backhaul) updateMetrics() {
	lat, loss, jitter := measureLatency(b.remoteAddr, 3)
	b.metrics.LatencyMs = lat
	b.metrics.PacketLoss = loss
	b.metrics.JitterMs = jitter
	if loss < 10 && lat < 200 {
		b.metrics.Stability = math.Min(1, b.metrics.Stability+0.05)
	} else {
		b.metrics.Stability = math.Max(0, b.metrics.Stability-0.08)
	}
	b.metrics.Bandwidth = 150
}
