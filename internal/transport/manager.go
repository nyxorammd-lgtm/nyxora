package transport

import (
	"fmt"
	"log"
	"sort"
	"sync"
)

type Manager struct {
	mu         sync.RWMutex
	transports map[string]Transport
	active     string
	allActive  bool
}

func NewManager(allActive bool) *Manager {
	return &Manager{
		transports: make(map[string]Transport),
		allActive:  allActive,
	}
}

func (m *Manager) Register(t Transport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transports[t.Name()] = t
	log.Printf("[transport] registered: %s (%s)", t.Name(), t.Type())
}

func (m *Manager) Get(name string) (Transport, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.transports[name]
	return t, ok
}

func (m *Manager) List() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []Info
	for _, t := range m.transports {
		metrics := t.Metrics()
		list = append(list, Info{
			Name:      t.Name(),
			Type:      t.Type(),
			Status:    t.Status(),
			Score:     t.Score(),
			Latency:   metrics.LatencyMs,
			Jitter:    metrics.JitterMs,
			Loss:      metrics.PacketLoss,
			Stability: metrics.Stability,
			Bandwidth: metrics.Bandwidth,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Score > list[j].Score
	})
	return list
}

func (m *Manager) Best() (Transport, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bestLocked()
}

func (m *Manager) ConnectAll(remoteAddr string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[manager] testing all transports to %s", remoteAddr)

	type candidate struct {
		t     Transport
		score float64
	}

	var candidates []candidate
	for name, t := range m.transports {
		log.Printf("[manager] trying %s...", name)
		err := t.Connect(remoteAddr)
		if err != nil {
			log.Printf("[manager] %s failed: %v", name, err)
			t.Disconnect()
			continue
		}
		score := t.Score()
		log.Printf("[manager] %s score: %.1f", name, score)
		candidates = append(candidates, candidate{t, score})
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no transport could connect to %s", remoteAddr)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	best := candidates[0]
	m.active = best.t.Name()

	if !m.allActive {
		for i, c := range candidates {
			if i > 0 {
				log.Printf("[manager] disconnecting %s (not selected)", c.t.Name())
				c.t.Disconnect()
			}
		}
	}

	log.Printf("[manager] active transport: %s (score: %.1f)", best.t.Name(), best.score)
	return nil
}

func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, t := range m.transports {
		if err := t.Disconnect(); err != nil {
			log.Printf("[transport] %s disconnect error: %v", name, err)
		}
	}
	m.active = ""
}

func (m *Manager) Active() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

func (m *Manager) ActiveTransport() Transport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.active == "" {
		return nil
	}
	return m.transports[m.active]
}

func (m *Manager) bestLocked() (Transport, float64) {
	var best Transport
	var bestScore float64
	for _, t := range m.transports {
		s := t.Score()
		if s > bestScore {
			bestScore = s
			best = t
		}
	}
	return best, bestScore
}

func (m *Manager) RefreshScores() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.transports {
		t.Score()
	}
}
