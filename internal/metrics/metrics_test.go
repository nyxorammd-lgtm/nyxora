package metrics

import (
	"testing"
)

func TestCounter(t *testing.T) {
	c := NewCounter(nil)
	if c.Value() != 0 {
		t.Error("initial counter should be 0")
	}
	c.Inc()
	c.Inc()
	c.Add(3)
	if c.Value() != 5 {
		t.Errorf("expected 5, got %f", c.Value())
	}
	c.Reset()
	if c.Value() != 0 {
		t.Error("reset counter should be 0")
	}
}

func TestGauge(t *testing.T) {
	g := NewGauge(nil)
	g.Set(42)
	if g.Value() != 42 {
		t.Errorf("expected 42, got %f", g.Value())
	}
	g.Inc()
	if g.Value() != 43 {
		t.Errorf("expected 43, got %f", g.Value())
	}
	g.Dec()
	if g.Value() != 42 {
		t.Errorf("expected 42, got %f", g.Value())
	}
}

func TestHistogram(t *testing.T) {
	h := NewHistogram([]float64{10, 50, 100})
	h.Observe(5)
	h.Observe(25)
	h.Observe(75)
	h.Observe(200)
	if h.Count() != 4 {
		t.Errorf("expected 4, got %d", h.Count())
	}
	if h.Sum() != 305 {
		t.Errorf("expected 305, got %f", h.Sum())
	}
	if h.Mean() != 76.25 {
		t.Errorf("expected 76.25, got %f", h.Mean())
	}
}

func TestCollector(t *testing.T) {
	c := NewCollector()
	c.Counter("requests", nil).Inc()
	c.Gauge("connections", nil).Set(42)
	c.Histogram("latency", nil).Observe(10)

	snap := c.Snapshot()
	if _, ok := snap["counters"]; !ok {
		t.Error("snapshot should have counters")
	}
	if _, ok := snap["gauges"]; !ok {
		t.Error("snapshot should have gauges")
	}
	if _, ok := snap["histograms"]; !ok {
		t.Error("snapshot should have histograms")
	}
}

func TestPromethize(t *testing.T) {
	c := NewCollector()
	c.Counter("requests", nil).Inc()
	output := c.Promethize()
	if len(output) == 0 {
		t.Error("promethize should produce output")
	}
}
