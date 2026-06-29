package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type FRP struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	cmd         *exec.Cmd
}

func NewFRP() *FRP {
	return &FRP{
		name:   "frp",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   7000,
	}
}

func (f *FRP) Name() string { return f.name }
func (f *FRP) Type() string { return "frp" }

func (f *FRP) Init(cfg map[string]string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &f.port)
	}
	log.Printf("[frp] initialized (port: %d)", f.port)
	return nil
}

func (f *FRP) Connect(remoteAddr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.remoteAddr = remoteAddr
	f.status = StatusTesting
	f.updateMetrics()

	if f.metrics.PacketLoss > 80 {
		f.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", f.metrics.PacketLoss)
	}

	if !commandExists("frpc") {
		log.Printf("[frp] frpc not found, using ping score")
		f.status = StatusActive
		return nil
	}

	cfg := fmt.Sprintf(`[common]
server_addr = %s
server_port = %d

[nyxora-tunnel]
type = tcp
local_ip = 127.0.0.1
local_port = 22
remote_port = 6000
`, remoteAddr, f.port)

	tmpPath := fmt.Sprintf("/tmp/nyxora-frpc-%s.ini", remoteAddr)
	if err := writeConfig(tmpPath, cfg); err != nil {
		log.Printf("[frp] config error: %v", err)
		f.status = StatusActive
		return nil
	}

	go func() {
		cmd := exec.Command("frpc", "-c", tmpPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[frp] start error: %v", err)
			return
		}
		f.mu.Lock()
		f.cmd = cmd
		f.status = StatusActive
		f.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[frp] connecting to %s:%d", remoteAddr, f.port)
	f.status = StatusActive
	return nil
}

func (f *FRP) Disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cmd != nil && f.cmd.Process != nil {
		f.cmd.Process.Kill()
		f.cmd = nil
	}
	f.status = StatusInactive
	log.Printf("[frp] disconnected")
	return nil
}

func (f *FRP) Status() Status {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.status
}

func (f *FRP) Metrics() *Metrics {
	f.mu.RLock()
	defer f.mu.RUnlock()
	m := *f.metrics
	return &m
}

func (f *FRP) Health() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.status == StatusActive
}

func (f *FRP) Score() float64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateMetrics()
	m := f.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/2)*0.35 + math.Max(0, 100-m.PacketLoss*2)*0.30 +
		math.Max(0, 100-m.JitterMs*3)*0.15 + (m.Stability*100)*0.20
}

func (f *FRP) updateMetrics() {
	lat, loss, jitter := measureLatency(f.remoteAddr, 3)
	f.metrics.LatencyMs = lat
	f.metrics.PacketLoss = loss
	f.metrics.JitterMs = jitter
	if loss < 10 && lat < 200 {
		f.metrics.Stability = math.Min(1, f.metrics.Stability+0.05)
	} else {
		f.metrics.Stability = math.Max(0, f.metrics.Stability-0.1)
	}
	f.metrics.Bandwidth = 100
}
