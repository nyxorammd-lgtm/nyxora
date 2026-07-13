package multipath

import (
	"testing"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
}

func TestSchedulerAddPath(t *testing.T) {
	s := NewScheduler()
	s.AddPath("wireguard", "wireguard", 30)
	s.AddPath("ssh", "ssh", 5)

	stats := s.Stats()
	if stats.FailoverCount != 0 {
		t.Errorf("expected 0 failovers, got %d", stats.FailoverCount)
	}
}

func TestSchedulerRecordFailover(t *testing.T) {
	s := NewScheduler()
	s.RecordFailover()
	s.RecordFailover()
	stats := s.Stats()
	if stats.FailoverCount != 2 {
		t.Errorf("expected 2 failovers, got %d", stats.FailoverCount)
	}
}

func TestSchedulerSelectPath(t *testing.T) {
	s := NewScheduler()
	s.AddPath("wireguard", "wireguard", 30)
	s.AddPath("ssh", "ssh", 5)
	s.SetActive("wireguard", true)
	s.SetActive("ssh", true)
	path := s.SelectPath()
	if path == nil {
		t.Error("SelectPath should return a path")
	}
}

func TestSchedulerStats(t *testing.T) {
	s := NewScheduler()
	stats := s.Stats()
	if stats.ActivePaths != 0 {
		t.Errorf("expected 0 active paths initially, got %d", stats.ActivePaths)
	}
}
