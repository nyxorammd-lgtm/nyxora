package transport

import (
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

type TCP struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	localPort   int
	listener    net.Listener
	connections []net.Conn
}

func NewTCP() *TCP {
	return &TCP{
		name:      "tcp",
		status:    StatusInactive,
		metrics:   &Metrics{},
		port:      9924,
		localPort: 9925,
	}
}

func (t *TCP) Name() string { return t.name }

func (t *TCP) Type() string { return "tcp" }

func (t *TCP) Init(cfg map[string]string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &t.port)
	}
	if lp, ok := cfg["local_port"]; ok {
		fmt.Sscanf(lp, "%d", &t.localPort)
	}
	log.Printf("[tcp] initialized (port: %d, local: %d)", t.port, t.localPort)
	return nil
}

func (t *TCP) Connect(remoteAddr string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.remoteAddr = remoteAddr
	t.status = StatusTesting

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", remoteAddr, t.port), 5*time.Second)
	if err != nil {
		t.status = StatusFailed
		return fmt.Errorf("tcp connect failed: %w", err)
	}
	t.connections = append(t.connections, conn)

	t.status = StatusActive
	log.Printf("[tcp] connected to %s:%d", remoteAddr, t.port)

	go t.handleConnection(conn)
	return nil
}

func (t *TCP) handleConnection(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[tcp] connection read error: %v", err)
			}
			t.mu.Lock()
			t.status = StatusFailed
			t.mu.Unlock()
			return
		}
	}
}

func (t *TCP) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, conn := range t.connections {
		conn.Close()
	}
	t.connections = nil
	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}
	t.status = StatusInactive
	log.Printf("[tcp] disconnected")
	return nil
}

func (t *TCP) Status() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

func (t *TCP) Metrics() *Metrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	m := *t.metrics
	return &m
}

func (t *TCP) Health() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status == StatusActive
}

func (t *TCP) Score() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.updateMetrics()

	m := t.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}

	latencyScore := math.Max(0, 100-m.LatencyMs/2)
	lossScore := math.Max(0, 100-m.PacketLoss*2)
	jitterScore := math.Max(0, 100-m.JitterMs*3)
	stabilityScore := m.Stability * 100

	score := latencyScore*0.25 + lossScore*0.35 + jitterScore*0.15 + stabilityScore*0.25
	return score
}

func (t *TCP) updateMetrics() {
	latency, loss, jitter := measureLatency(t.remoteAddr, 3)
	t.metrics.LatencyMs = latency
	t.metrics.PacketLoss = loss
	t.metrics.JitterMs = jitter

	if loss < 10 && latency < 200 {
		t.metrics.Stability = math.Min(1, t.metrics.Stability+0.05)
	} else {
		t.metrics.Stability = math.Max(0, t.metrics.Stability-0.1)
	}
}
