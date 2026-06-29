package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type Hysteria struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	cmd         *exec.Cmd
}

func NewHysteria() *Hysteria {
	return &Hysteria{
		name:   "hysteria",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   8443,
	}
}

func (h *Hysteria) Name() string { return h.name }
func (h *Hysteria) Type() string { return "hysteria" }

func (h *Hysteria) Init(cfg map[string]string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &h.port)
	}
	log.Printf("[hysteria] initialized (port: %d)", h.port)
	return nil
}

func (h *Hysteria) Connect(remoteAddr string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.remoteAddr = remoteAddr
	h.status = StatusTesting
	h.updateMetrics()

	if h.metrics.PacketLoss > 80 {
		h.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", h.metrics.PacketLoss)
	}

	if !commandExists("hysteria") {
		log.Printf("[hysteria] binary not found, ping score only")
		h.status = StatusActive
		return nil
	}

	config := fmt.Sprintf(`server: %s:%d
auth: nyxora-hy2-auth
transport:
  type: udp
  udp:
    hopInterval: 5s
bandwidth:
  up: 100 mbps
  down: 500 mbps
socks5:
  listen: 127.0.0.1:1082
`, remoteAddr, h.port)

	tmpPath := fmt.Sprintf("/tmp/nyxora-hy2-%s.yaml", remoteAddr)
	if err := writeConfig(tmpPath, config); err != nil {
		log.Printf("[hysteria] config error: %v", err)
		h.status = StatusActive
		return nil
	}

	go func() {
		cmd := exec.Command("hysteria", "client", "-c", tmpPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[hysteria] start error: %v", err)
			return
		}
		h.mu.Lock()
		h.cmd = cmd
		h.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[hysteria] connecting to %s:%d", remoteAddr, h.port)
	h.status = StatusActive
	return nil
}

func (h *Hysteria) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cmd != nil && h.cmd.Process != nil {
		h.cmd.Process.Kill()
		h.cmd = nil
	}
	h.status = StatusInactive
	log.Printf("[hysteria] disconnected")
	return nil
}

func (h *Hysteria) Status() Status {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func (h *Hysteria) Metrics() *Metrics {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := *h.metrics
	return &m
}

func (h *Hysteria) Health() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status == StatusActive
}

func (h *Hysteria) Score() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateMetrics()
	m := h.metrics
	if m.PacketLoss > 60 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/4)*0.20 + math.Max(0, 100-m.PacketLoss*1.5)*0.35 +
		math.Max(0, 100-m.JitterMs*2)*0.15 + (m.Stability*100)*0.30
}

func (h *Hysteria) updateMetrics() {
	lat, loss, jitter := measureLatency(h.remoteAddr, 3)
	h.metrics.LatencyMs = lat
	h.metrics.PacketLoss = loss
	h.metrics.JitterMs = jitter
	if loss < 20 || lat < 400 {
		h.metrics.Stability = math.Min(1, h.metrics.Stability+0.07)
	} else {
		h.metrics.Stability = math.Max(0, h.metrics.Stability-0.10)
	}
	h.metrics.Bandwidth = 300
}
