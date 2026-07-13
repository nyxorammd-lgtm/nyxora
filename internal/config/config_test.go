package config

import (
	"os"
	"testing"
)

func TestWatcherStartStop(t *testing.T) {
	tmp := t.TempDir() + "/test-config.json"
	os.WriteFile(tmp, []byte(`{"mode":"lite"}`), 0644)

	called := false
	w := NewWatcher(tmp, func(cfg *Config) {
		called = true
	}, 1)

	if err := w.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	w.Stop()

	if !called {
		t.Log("callback not called (file not changed, expected)")
	}
}

func TestConfigLoadAndSave(t *testing.T) {
	tmp := t.TempDir() + "/test-config.json"
	cfg := &DefaultConfig
	cfg.Mode = ModeLite
	if err := cfg.Save(tmp); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Mode != ModeLite {
		t.Errorf("expected mode lite, got %s", loaded.Mode)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := DefaultConfig
	cfg.Mode = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid mode")
	}

	cfg.Mode = ModeFull
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDetectMode(t *testing.T) {
	mode := DetectMode()
	if mode != ModeFull && mode != ModeLite && mode != ModeMinimal {
		t.Errorf("unexpected mode: %s", mode)
	}
}

func TestGetTransportsForMode(t *testing.T) {
	full := GetTransportsForMode(ModeFull)
	lite := GetTransportsForMode(ModeLite)
	min := GetTransportsForMode(ModeMinimal)

	if len(full) != 12 {
		t.Errorf("full mode should have 12 transports, got %d", len(full))
	}
	if len(lite) != 6 {
		t.Errorf("lite mode should have 6 transports, got %d", len(lite))
	}
	if len(min) != 2 {
		t.Errorf("minimal mode should have 2 transports, got %d", len(min))
	}
}

func TestValidateTransports(t *testing.T) {
	err := ValidateTransports([]string{"wireguard", "ssh"})
	if err != nil {
		t.Errorf("valid transports should pass: %v", err)
	}

	err = ValidateTransports([]string{"invalid_transport"})
	if err == nil {
		t.Error("invalid transport should fail")
	}
}

func TestValidatePortOverrides(t *testing.T) {
	err := ValidatePortOverrides(map[string]int{"wireguard": 51820})
	if err != nil {
		t.Errorf("valid port override should pass: %v", err)
	}

	err = ValidatePortOverrides(map[string]int{"wireguard": 99999})
	if err == nil {
		t.Error("invalid port should fail")
	}
}

func TestLoadSecrets(t *testing.T) {
	s := LoadSecrets()
	if s.SSPassword == "" {
		t.Error("SS password should be generated")
	}
	if s.SSMethod == "" {
		t.Error("SS method should have default")
	}
}
