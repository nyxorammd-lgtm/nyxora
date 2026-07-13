package remote

import (
	"testing"
)

func TestHostNew(t *testing.T) {
	h := NewHost("192.168.1.1", 22, "root", "password")
	if h == nil {
		t.Fatal("NewHost returned nil")
	}
	if h.Address != "192.168.1.1" {
		t.Errorf("expected addr 192.168.1.1, got %s", h.Address)
	}
	if h.Port != 22 {
		t.Errorf("expected port 22, got %d", h.Port)
	}
	if h.User != "root" {
		t.Errorf("expected user root, got %s", h.User)
	}
}
