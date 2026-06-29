package transport

import (
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

type QUIC struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
}

func NewQUIC() *QUIC {
	return &QUIC{
		name:   "quic",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   9923,
	}
}

func (q *QUIC) Name() string { return q.name }
func (q *QUIC) Type() string { return "quic" }

func (q *QUIC) Init(cfg map[string]string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &q.port)
	}
	log.Printf("[quic] initialized (port: %d)", q.port)
	return nil
}

func (q *QUIC) Connect(remoteAddr string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.remoteAddr = remoteAddr
	q.status = StatusTesting

	q.updateMetrics()

	if q.metrics.PacketLoss > 80 {
		q.status = StatusFailed
		return fmt.Errorf("high packet loss (%.1f%%)", q.metrics.PacketLoss)
	}

	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", remoteAddr, q.port), 3*time.Second)
	if err != nil {
		log.Printf("[quic] port %d unreachable, using ping-only mode", q.port)
		q.status = StatusActive
		return nil
	}
	conn.Close()

	q.status = StatusActive
	log.Printf("[quic] connected to %s:%d", remoteAddr, q.port)
	return nil
}

func (q *QUIC) Disconnect() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.status = StatusInactive
	log.Printf("[quic] disconnected")
	return nil
}

func (q *QUIC) Status() Status {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.status
}

func (q *QUIC) Metrics() *Metrics {
	q.mu.RLock()
	defer q.mu.RUnlock()
	m := *q.metrics
	return &m
}

func (q *QUIC) Health() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.status == StatusActive
}

func (q *QUIC) Score() float64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.updateMetrics()

	m := q.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 10
	}

	latencyScore := math.Max(0, 100-m.LatencyMs/2)
	lossScore := math.Max(0, 100-m.PacketLoss*2)
	jitterScore := math.Max(0, 100-m.JitterMs*3)
	stabilityScore := m.Stability * 100

	return latencyScore*0.35 + lossScore*0.30 + jitterScore*0.15 + stabilityScore*0.20
}

func (q *QUIC) updateMetrics() {
	latency, loss, jitter := measureLatency(q.remoteAddr, 4)
	q.metrics.LatencyMs = latency
	q.metrics.PacketLoss = loss
	q.metrics.JitterMs = jitter

	if loss < 5 && latency < 100 {
		q.metrics.Stability = math.Min(1, q.metrics.Stability+0.1)
	} else {
		q.metrics.Stability = math.Max(0, q.metrics.Stability-0.1)
	}
}
