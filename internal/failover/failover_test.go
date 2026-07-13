package failover

import (
	"testing"
	"time"
)

func TestNewFailover(t *testing.T) {
	f := NewFailover(10)
	if f == nil {
		t.Fatal("NewFailover returned nil")
	}
}

func TestFailoverUpdate(t *testing.T) {
	f := NewFailover(10)
	f.Update("wireguard", 50, 0)
	if !f.IsHealthy("wireguard") {
		t.Error("wireguard should be healthy")
	}

	f.Update("wireguard", 300, 50)
	for i := 0; i < 3; i++ {
		f.Update("wireguard", 300, 50)
	}
	if f.IsHealthy("wireguard") {
		t.Error("wireguard should not be healthy after bad updates")
	}
}

func TestFailoverStatus(t *testing.T) {
	f := NewFailover(10)
	status := f.Status("nonexistent")
	if status != StatusDown {
		t.Error("nonexistent transport should be down")
	}
}

func TestFailoverAllStatus(t *testing.T) {
	f := NewFailover(10)
	f.Update("wg", 10, 0)
	all := f.AllStatus()
	if len(all) != 1 {
		t.Errorf("expected 1 status, got %d", len(all))
	}
}

func TestFailoverStartStop(t *testing.T) {
	f := NewFailover(1)
	f.Start()
	time.Sleep(50 * time.Millisecond)
	f.Stop()
}

func TestFailoverCallbacks(t *testing.T) {
	f := NewFailover(10)
	failoverCalled := false
	recoverCalled := false

	f.OnFailover(func(from, to string) {
		failoverCalled = true
	})
	f.OnRecover(func(name string) {
		recoverCalled = true
	})

	cb := f.GetOnFailover()
	if cb == nil {
		t.Error("OnFailover callback should be set")
	}
	_ = failoverCalled
	_ = recoverCalled
}
