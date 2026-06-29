package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type SSH struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	user        string
	password    string
	cmd         *exec.Cmd
	localPort   int
}

func NewSSH() *SSH {
	return &SSH{
		name:      "ssh",
		status:    StatusInactive,
		metrics:   &Metrics{},
		port:      22,
		user:      "root",
		localPort: 1080,
	}
}

func (s *SSH) Name() string { return s.name }
func (s *SSH) Type() string { return "ssh" }

func (s *SSH) Init(cfg map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &s.port)
	}
	if u, ok := cfg["user"]; ok {
		s.user = u
	}
	if pw, ok := cfg["password"]; ok {
		s.password = pw
	}
	log.Printf("[ssh] initialized (port: %d, user: %s)", s.port, s.user)
	return nil
}

func (s *SSH) Connect(remoteAddr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.remoteAddr = remoteAddr
	s.status = StatusTesting

	s.updateMetrics()

	if s.metrics.PacketLoss > 80 {
		s.status = StatusFailed
		return fmt.Errorf("high packet loss (%.1f%%)", s.metrics.PacketLoss)
	}

	pingOnly := false

	checkCmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "PasswordAuthentication=no",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-p", fmt.Sprintf("%d", s.port),
		fmt.Sprintf("%s@%s", s.user, remoteAddr),
		"echo connected",
	)

	if err := checkCmd.Run(); err != nil {
		log.Printf("[ssh] key auth failed for %s@%s (trying password mode)", s.user, remoteAddr)
		pingOnly = true
	}

	if pingOnly {
		log.Printf("[ssh] using ping-only mode for %s", remoteAddr)
		s.status = StatusActive
		return nil
	}

	go func() {
		tunnelCmd := exec.Command("ssh",
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-o", "ServerAliveInterval=10",
			"-o", "ServerAliveCountMax=3",
			"-o", "ExitOnForwardFailure=yes",
			"-N",
			"-D", fmt.Sprintf("127.0.0.1:%d", s.localPort),
			"-p", fmt.Sprintf("%d", s.port),
			fmt.Sprintf("%s@%s", s.user, remoteAddr),
		)

		s.mu.Lock()
		s.cmd = tunnelCmd
		s.mu.Unlock()

		log.Printf("[ssh] starting tunnel %s@%s:%d -> 127.0.0.1:%d",
			s.user, remoteAddr, s.port, s.localPort)

		if err := tunnelCmd.Start(); err != nil {
			log.Printf("[ssh] tunnel start error: %v", err)
			s.mu.Lock()
			s.status = StatusFailed
			s.mu.Unlock()
			return
		}

		s.mu.Lock()
		s.status = StatusActive
		s.mu.Unlock()

		if err := tunnelCmd.Wait(); err != nil {
			log.Printf("[ssh] tunnel exited: %v", err)
			s.mu.Lock()
			s.status = StatusFailed
			s.mu.Unlock()
		}
	}()

	return nil
}

func (s *SSH) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
		s.cmd = nil
	}
	s.status = StatusInactive
	log.Printf("[ssh] disconnected")
	return nil
}

func (s *SSH) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *SSH) Metrics() *Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := *s.metrics
	return &m
}

func (s *SSH) Health() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status == StatusActive
}

func (s *SSH) Score() float64 {
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

	latencyScore := math.Max(0, 100-m.LatencyMs/3)
	lossScore := math.Max(0, 100-m.PacketLoss*2)
	jitterScore := math.Max(0, 100-m.JitterMs*2)
	stabilityScore := m.Stability * 100

	return latencyScore*0.30 + lossScore*0.30 + jitterScore*0.15 + stabilityScore*0.25
}

func (s *SSH) updateMetrics() {
	latency, loss, jitter := measureLatency(s.remoteAddr, 3)
	s.metrics.LatencyMs = latency
	s.metrics.PacketLoss = loss
	s.metrics.JitterMs = jitter

	if loss < 10 && latency < 200 {
		s.metrics.Stability = math.Min(1, s.metrics.Stability+0.05)
	} else {
		s.metrics.Stability = math.Max(0, s.metrics.Stability-0.15)
	}
}
