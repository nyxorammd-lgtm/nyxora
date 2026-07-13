package dns

import (
	"testing"
)

func TestNewResolver(t *testing.T) {
	r := NewResolver(nil, 3, 300)
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	if len(r.nameservers) == 0 {
		t.Error("should have default nameservers")
	}
}

func TestResolverLookup(t *testing.T) {
	r := NewResolver([]string{"1.1.1.1:53"}, 3, 300)
	ip, err := r.Lookup("localhost")
	if err != nil {
		t.Logf("lookup localhost: %v (may fail in CI)", err)
		return
	}
	if ip == nil {
		t.Error("IP should not be nil")
	}
}

func TestResolverClearCache(t *testing.T) {
	r := NewResolver(nil, 3, 300)
	r.ClearCache()
	if r.CacheSize() != 0 {
		t.Error("cache should be empty after clear")
	}
}

func TestResolverStats(t *testing.T) {
	r := NewResolver(nil, 3, 300)
	stats := r.Stats()
	if stats["cache_size"] != 0 {
		t.Error("initial cache size should be 0")
	}
}

func TestMakeDNSQuery(t *testing.T) {
	msg := makeDNSQuery("example.com")
	if len(msg) == 0 {
		t.Error("DNS query should not be empty")
	}
}

func TestSplitDomain(t *testing.T) {
	labels := splitDomain("www.example.com")
	if len(labels) != 3 {
		t.Errorf("expected 3 labels, got %d", len(labels))
	}
	if labels[0] != "www" || labels[1] != "example" || labels[2] != "com" {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestMonitorStartStop(t *testing.T) {
	r := NewResolver(nil, 3, 300)
	m := NewMonitor(r, 1)
	m.Start([]string{"localhost"})
	m.Stop()
}
