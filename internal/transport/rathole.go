package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type Rathole struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	cmd         *exec.Cmd
}

func NewRathole() *Rathole {
	return &Rathole{
		name:   "rathole",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   2333,
	}
}

func (r *Rathole) Name() string { return r.name }
func (r *Rathole) Type() string { return "rathole" }

func (r *Rathole) Init(cfg map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &r.port)
	}
	log.Printf("[rathole] initialized (port: %d)", r.port)
	return nil
}

func (r *Rathole) Connect(remoteAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.remoteAddr = remoteAddr
	r.status = StatusTesting
	r.updateMetrics()

	if r.metrics.PacketLoss > 80 {
		r.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", r.metrics.PacketLoss)
	}

	if !commandExists("rathole") {
		log.Printf("[rathole] binary not found, scoring from ping")
		r.status = StatusActive
		return nil
	}

	cfg := fmt.Sprintf(`[client]
remote_addr = "%s:%d"
[client.services.nyxora]
type = "tcp"
local_addr = "127.0.0.1:22"
`, remoteAddr, r.port)

	tmpPath := fmt.Sprintf("/tmp/nyxora-rathole-%s.toml", remoteAddr)
	if err := writeConfig(tmpPath, cfg); err != nil {
		log.Printf("[rathole] config error: %v", err)
		r.status = StatusActive
		return nil
	}

	go func() {
		cmd := exec.Command("rathole", "--client", tmpPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[rathole] start error: %v", err)
			return
		}
		r.mu.Lock()
		r.cmd = cmd
		r.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[rathole] connecting to %s:%d", remoteAddr, r.port)
	r.status = StatusActive
	return nil
}

func (r *Rathole) Disconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		r.cmd.Process.Kill()
		r.cmd = nil
	}
	r.status = StatusInactive
	log.Printf("[rathole] disconnected")
	return nil
}

func (r *Rathole) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status
}

func (r *Rathole) Metrics() *Metrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := *r.metrics
	return &m
}

func (r *Rathole) Health() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status == StatusActive
}

func (r *Rathole) Score() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateMetrics()
	m := r.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/2)*0.30 + math.Max(0, 100-m.PacketLoss*2)*0.35 +
		math.Max(0, 100-m.JitterMs*3)*0.10 + (m.Stability*100)*0.25
}

func (r *Rathole) updateMetrics() {
	lat, loss, jitter := measureLatency(r.remoteAddr, 3)
	r.metrics.LatencyMs = lat
	r.metrics.PacketLoss = loss
	r.metrics.JitterMs = jitter
	if loss < 8 && lat < 150 {
		r.metrics.Stability = math.Min(1, r.metrics.Stability+0.06)
	} else {
		r.metrics.Stability = math.Max(0, r.metrics.Stability-0.12)
	}
	r.metrics.Bandwidth = 200
}
