package transport

import (
	"testing"
)

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name     string
		metrics  *Metrics
		expected float64
	}{
		{
			name:     "perfect conditions",
			metrics:  &Metrics{LatencyMs: 10, PacketLoss: 0, JitterMs: 1, Stability: 1.0},
			expected: 98.05,
		},
		{
			name:     "high loss",
			metrics:  &Metrics{LatencyMs: 50, PacketLoss: 60, JitterMs: 5, Stability: 0.8},
			expected: 0,
		},
		{
			name:     "zero latency",
			metrics:  &Metrics{LatencyMs: 0, PacketLoss: 0, JitterMs: 0, Stability: 1.0},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ComputeScore(tt.metrics, DefaultScoringWeights)
			if score != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, score)
			}
		})
	}
}

func TestCommandExists(t *testing.T) {
	if !CommandExists("ls") {
		t.Error("ls should exist")
	}
	if CommandExists("nonexistent_binary_xyz") {
		t.Error("nonexistent binary should not exist")
	}
}

func TestFormatEndpoint(t *testing.T) {
	tests := []struct {
		addr     string
		port     int
		expected string
	}{
		{"192.168.1.1", 51820, "192.168.1.1:51820"},
		{"::1", 80, "[::1]:80"},
		{"localhost", 22, "localhost:22"},
	}
	for _, tt := range tests {
		result := FormatEndpoint(tt.addr, tt.port)
		if result != tt.expected {
			t.Errorf("FormatEndpoint(%s, %d) = %s, want %s", tt.addr, tt.port, result, tt.expected)
		}
	}
}

func TestExtractSubnet(t *testing.T) {
	tests := []struct {
		addr     string
		expected int
	}{
		{"192.168.1.0", 1},
		{"10.0.0.0", 0},
		{"172.16.5.0", 5},
		{"invalid", 0},
	}
	for _, tt := range tests {
		result := ExtractSubnet(tt.addr)
		if result != tt.expected {
			t.Errorf("ExtractSubnet(%s) = %d, want %d", tt.addr, result, tt.expected)
		}
	}
}

func TestUpdateStability(t *testing.T) {
	m := &Metrics{Stability: 0.5, PacketLoss: 1, LatencyMs: 50}
	UpdateStability(m, 10, 200, 0.05, 0.10)
	if m.Stability <= 0.5 {
		t.Error("stability should increase for good conditions")
	}

	m = &Metrics{Stability: 0.5, PacketLoss: 50, LatencyMs: 300}
	UpdateStability(m, 10, 200, 0.05, 0.10)
	if m.Stability >= 0.5 {
		t.Error("stability should decrease for bad conditions")
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager(false)
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager(false)
	m.Register(NewTCP())
	if len(m.List()) != 1 {
		t.Errorf("expected 1 transport, got %d", len(m.List()))
	}
}

func TestManagerWeightNormalize(t *testing.T) {
	m := NewManager(false)
	m.Register(NewTCP())
	m.Register(NewSSH())
	w := m.GetWeights()
	if len(w) != 2 {
		t.Errorf("expected 2 weights, got %d", len(w))
	}
}

func TestTunnelRegistryLookup(t *testing.T) {
	meta, err := LookupTunnel("wireguard")
	if err != nil {
		t.Fatalf("LookupTunnel(wireguard): %v", err)
	}
	if meta.Port != 51820 {
		t.Errorf("expected port 51820, got %d", meta.Port)
	}

	_, err = LookupTunnel("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tunnel")
	}
}

func TestCategoryList(t *testing.T) {
	vpn := CategoryList(CatVPN)
	if len(vpn) == 0 {
		t.Error("should have VPN tunnels")
	}
	for _, t2 := range vpn {
		if t2.Category != CatVPN {
			t.Errorf("expected category vpn, got %s", t2.Category)
		}
	}
}

func TestInstallScript(t *testing.T) {
	script := InstallScript("wireguard")
	if script == "" {
		t.Error("wireguard install script should not be empty")
	}

	script = InstallScript("ssh")
	if script != "" {
		t.Error("ssh install script should be empty (pre-installed)")
	}

	script = InstallScript("nonexistent")
	if script != "" {
		t.Error("nonexistent install script should be empty")
	}
}
