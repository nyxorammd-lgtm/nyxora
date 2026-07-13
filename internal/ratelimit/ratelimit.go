package ratelimit

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}
	return false
}

func (tb *TokenBucket) Wait(tokens float64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if tb.Allow(tokens) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now
}

func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

type Limiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	config   Config
}

type Config struct {
	Enabled    bool    `json:"enabled"`
	MaxTokens  float64 `json:"max_tokens"`
	RefillRate float64 `json:"refill_rate"`
}

var DefaultConfig = Config{
	Enabled:    true,
	MaxTokens:  1000,
	RefillRate: 100,
}

func NewLimiter(cfg Config) *Limiter {
	return &Limiter{
		buckets: make(map[string]*TokenBucket),
		config:  cfg,
	}
}

func (l *Limiter) GetBucket(key string) *TokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.buckets[key]; !ok {
		l.buckets[key] = NewTokenBucket(l.config.MaxTokens, l.config.RefillRate)
	}
	return l.buckets[key]
}

func (l *Limiter) Allow(key string, tokens float64) bool {
	if !l.config.Enabled {
		return true
	}
	return l.GetBucket(key).Allow(tokens)
}

func (l *Limiter) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func (l *Limiter) Stats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return map[string]interface{}{
		"enabled":     l.config.Enabled,
		"max_tokens":  l.config.MaxTokens,
		"refill_rate": l.config.RefillRate,
		"buckets":     len(l.buckets),
	}
}

type TransportLimiter struct {
	limiter *Limiter
}

func NewTransportLimiter(cfg Config) *TransportLimiter {
	return &TransportLimiter{limiter: NewLimiter(cfg)}
}

func (tl *TransportLimiter) AllowTransport(name string, bytes int64) bool {
	return tl.limiter.Allow(name, float64(bytes))
}

func (tl *TransportLimiter) Stats() map[string]interface{} {
	return tl.limiter.Stats()
}
