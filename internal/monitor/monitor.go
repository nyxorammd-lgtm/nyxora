package monitor

import (
	"log"
	"math"
	"os/exec"
	"sync"
	"time"
)

type Result struct {
	LatencyMs  float64 `json:"latency_ms"`
	JitterMs   float64 `json:"jitter_ms"`
	PacketLoss float64 `json:"packet_loss"`
	Jitter     float64 `json:"jitter"`
	Timestamp  time.Time `json:"timestamp"`
}

type Monitor struct {
	mu       sync.RWMutex
	history  map[string][]Result
	interval time.Duration
	running  bool
	stopCh   chan struct{}
}

func NewMonitor(intervalSec int) *Monitor {
	return &Monitor{
		history:  make(map[string][]Result),
		interval: time.Duration(intervalSec) * time.Second,
		stopCh:   make(chan struct{}),
	}
}

func (m *Monitor) Start(targets []string) {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	log.Printf("[monitor] started monitoring %d targets every %s", len(targets), m.interval)

	for {
		select {
		case <-m.stopCh:
			return
		case <-time.After(m.interval):
			for _, target := range targets {
				result := m.Ping(target, 4)
				m.mu.Lock()
				m.history[target] = append(m.history[target], result)
				if len(m.history[target]) > 100 {
					m.history[target] = m.history[target][len(m.history[target])-100:]
				}
				m.mu.Unlock()
			}
		}
	}
}

func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		close(m.stopCh)
		m.running = false
		log.Printf("[monitor] stopped")
	}
}

func (m *Monitor) Ping(target string, count int) Result {
	var rtts []float64
	var lossCount float64

	for i := 0; i < count; i++ {
		start := time.Now()
		cmd := exec.Command("ping", "-c", "1", "-W", "2", target)
		if err := cmd.Run(); err == nil {
			rtt := time.Since(start).Seconds() * 1000
			rtts = append(rtts, rtt)
		} else {
			lossCount++
		}
	}

	packetLoss := (lossCount / float64(count)) * 100

	if len(rtts) == 0 {
		return Result{LatencyMs: 999, PacketLoss: 100, Jitter: 999, Timestamp: time.Now()}
	}

	var sum float64
	for _, r := range rtts {
		sum += r
	}
	latency := sum / float64(len(rtts))

	var jitter float64
	if len(rtts) > 1 {
		var jSum float64
		for i := 1; i < len(rtts); i++ {
			diff := rtts[i] - rtts[i-1]
			if diff < 0 {
				diff = -diff
			}
			jSum += diff
		}
		jitter = jSum / float64(len(rtts)-1)
	}

	return Result{
		LatencyMs:  math.Round(latency*100) / 100,
		JitterMs:   math.Round(jitter*100) / 100,
		PacketLoss: math.Round(packetLoss*100) / 100,
		Jitter:     math.Round(jitter*100) / 100,
		Timestamp:  time.Now(),
	}
}

func (m *Monitor) LastResult(target string) (Result, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history, ok := m.history[target]
	if !ok || len(history) == 0 {
		return Result{}, false
	}
	return history[len(history)-1], true
}

func (m *Monitor) History(target string) []Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history, ok := m.history[target]
	if !ok {
		return nil
	}
	result := make([]Result, len(history))
	copy(result, history)
	return result
}

func (m *Monitor) AverageLatency(target string, n int) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history, ok := m.history[target]
	if !ok || len(history) == 0 {
		return 0, false
	}
	if n > len(history) {
		n = len(history)
	}
	var sum float64
	for i := len(history) - n; i < len(history); i++ {
		sum += history[i].LatencyMs
	}
	return sum / float64(n), true
}
