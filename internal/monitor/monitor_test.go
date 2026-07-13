package monitor

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m := NewMonitor(10)
	if m == nil {
		t.Fatal("NewMonitor returned nil")
	}
}

func TestMonitorPing(t *testing.T) {
	m := NewMonitor(10)
	result := m.Ping("127.0.0.1", 1)
	if result.LatencyMs <= 0 && result.PacketLoss == 0 {
		t.Error("ping to localhost should succeed")
	}
	if result.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestMonitorLastResult(t *testing.T) {
	m := NewMonitor(10)
	_, ok := m.LastResult("nonexistent")
	if ok {
		t.Error("nonexistent target should return false")
	}
}

func TestMonitorHistory(t *testing.T) {
	m := NewMonitor(10)
	history := m.History("nonexistent")
	if history != nil {
		t.Error("nonexistent target should return nil")
	}
}

func TestMonitorAverageLatency(t *testing.T) {
	m := NewMonitor(10)
	_, ok := m.AverageLatency("nonexistent", 5)
	if ok {
		t.Error("nonexistent target should return false")
	}
}

func TestMonitorStartStop(t *testing.T) {
	m := NewMonitor(1)
	go m.Start([]string{"127.0.0.1"})
	time.Sleep(100 * time.Millisecond)
	m.Stop()
}
