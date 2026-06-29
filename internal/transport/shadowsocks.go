package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
	"time"
)

type ShadowSOCKS struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	password    string
	method      string
	cmd         *exec.Cmd
}

func NewShadowSOCKS() *ShadowSOCKS {
	return &ShadowSOCKS{
		name:   "shadowsocks",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   8388,
		method: "aes-256-gcm",
		password: fmt.Sprintf("nyxora-ss-%d", time.Now().Unix()),
	}
}

func (s *ShadowSOCKS) Name() string { return s.name }
func (s *ShadowSOCKS) Type() string { return "shadowsocks" }

func (s *ShadowSOCKS) Init(cfg map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &s.port)
	}
	if pw, ok := cfg["password"]; ok {
		s.password = pw
	}
	log.Printf("[shadowsocks] initialized (port: %d)", s.port)
	return nil
}

func (s *ShadowSOCKS) Connect(remoteAddr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.remoteAddr = remoteAddr
	s.status = StatusTesting
	s.updateMetrics()

	if s.metrics.PacketLoss > 80 {
		s.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", s.metrics.PacketLoss)
	}

	if !commandExists("ss-local") && !commandExists("ss-redir") {
		log.Printf("[shadowsocks] binary not found, ping score only")
		s.status = StatusActive
		return nil
	}

	config := fmt.Sprintf(`{
    "server": "%s",
    "server_port": %d,
    "local_port": 1081,
    "password": "%s",
    "method": "%s",
    "timeout": 60
}`, remoteAddr, s.port, s.password, s.method)

	tmpPath := fmt.Sprintf("/tmp/nyxora-ss-%s.json", remoteAddr)
	if err := writeConfig(tmpPath, config); err != nil {
		log.Printf("[shadowsocks] config error: %v", err)
		s.status = StatusActive
		return nil
	}

	binary := "ss-local"
	if !commandExists(binary) {
		binary = "ss-redir"
	}

	go func() {
		cmd := exec.Command(binary, "-c", tmpPath)
		if err := cmd.Start(); err != nil {
			log.Printf("[shadowsocks] start error: %v", err)
			return
		}
		s.mu.Lock()
		s.cmd = cmd
		s.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[shadowsocks] connecting to %s:%d", remoteAddr, s.port)
	s.status = StatusActive
	return nil
}

func (s *ShadowSOCKS) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd = nil
	}
	s.status = StatusInactive
	log.Printf("[shadowsocks] disconnected")
	return nil
}

func (s *ShadowSOCKS) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *ShadowSOCKS) Metrics() *Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := *s.metrics
	return &m
}

func (s *ShadowSOCKS) Health() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status == StatusActive
}

func (s *ShadowSOCKS) Score() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateMetrics()
	m := s.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/3)*0.20 + math.Max(0, 100-m.PacketLoss*2)*0.30 +
		math.Max(0, 100-m.JitterMs*3)*0.20 + (m.Stability*100)*0.30
}

func (s *ShadowSOCKS) updateMetrics() {
	lat, loss, jitter := measureLatency(s.remoteAddr, 3)
	s.metrics.LatencyMs = lat
	s.metrics.PacketLoss = loss
	s.metrics.JitterMs = jitter
	if loss < 15 && lat < 250 {
		s.metrics.Stability = math.Min(1, s.metrics.Stability+0.04)
	} else {
		s.metrics.Stability = math.Max(0, s.metrics.Stability-0.08)
	}
	s.metrics.Bandwidth = 30
}


