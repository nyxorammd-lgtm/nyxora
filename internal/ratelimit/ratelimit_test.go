package ratelimit

import (
	"testing"
)

func TestTokenBucketAllow(t *testing.T) {
	tb := NewTokenBucket(10, 10)
	if !tb.Allow(5) {
		t.Error("should allow within capacity")
	}
	if !tb.Allow(5) {
		t.Error("should allow second batch")
	}
	if tb.Allow(1) {
		t.Error("should deny when empty")
	}
}

func TestTokenBucketAvailable(t *testing.T) {
	tb := NewTokenBucket(100, 0)
	avail := tb.Available()
	if avail != 100 {
		t.Errorf("expected 100, got %f", avail)
	}
	tb.Allow(30)
	avail = tb.Available()
	if avail != 70 {
		t.Errorf("expected 70, got %f", avail)
	}
}

func TestLimiterAllow(t *testing.T) {
	l := NewLimiter(Config{Enabled: true, MaxTokens: 100, RefillRate: 100})
	if !l.Allow("transport1", 50) {
		t.Error("should allow first request")
	}
	if !l.Allow("transport1", 50) {
		t.Error("should allow second request")
	}
	if l.Allow("transport1", 1) {
		t.Error("should deny when depleted")
	}
}

func TestLimiterDisabled(t *testing.T) {
	l := NewLimiter(Config{Enabled: false})
	for i := 0; i < 1000; i++ {
		if !l.Allow("test", 100) {
			t.Error("disabled limiter should always allow")
		}
	}
}

func TestLimiterRemove(t *testing.T) {
	l := NewLimiter(Config{Enabled: true, MaxTokens: 10, RefillRate: 0})
	l.Allow("key1", 10)
	l.Remove("key1")
	if !l.Allow("key1", 5) {
		t.Error("should allow after remove")
	}
}
