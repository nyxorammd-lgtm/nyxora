package routing

import (
	"log"
	"sync"
)

type Engine struct {
	mu      sync.RWMutex
	scorer  *Scorer
	paths   map[string]*PathScore
	current string
}

func NewEngine() *Engine {
	return &Engine{
		scorer: NewScorer(),
		paths:  make(map[string]*PathScore),
	}
}

func (e *Engine) Update(name, transportType string, latency, jitter, loss, stability float64, bandwidth int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	score := e.scorer.Score(latency, jitter, loss, stability, bandwidth)
	e.paths[name] = &PathScore{
		Name:      name,
		Type:      transportType,
		Score:     score,
		Latency:   latency,
		Jitter:    jitter,
		Loss:      loss,
		Stability: stability,
		Bandwidth: bandwidth,
	}
	log.Printf("[routing] %s score: %.1f (lat=%.1f, loss=%.1f%%, jit=%.1f)", name, score, latency, loss, jitter)
}

func (e *Engine) Current() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

func (e *Engine) SetCurrent(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.current = name
}

func (e *Engine) BestPath() *PathScore {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var best *PathScore
	for _, p := range e.paths {
		if best == nil || p.Score > best.Score {
			best = p
		}
	}
	return best
}

func (e *Engine) AllPaths() []PathScore {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []PathScore
	for _, p := range e.paths {
		list = append(list, *p)
	}
	if len(list) > 0 {
		e.scorer.Rank(list)
	}
	return list
}

func (e *Engine) NeedsFailover(threshold float64) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	current, ok := e.paths[e.current]
	if !ok {
		return true
	}

	best := e.BestPath()
	if best == nil {
		return true
	}

	return best.Score-current.Score > threshold
}
