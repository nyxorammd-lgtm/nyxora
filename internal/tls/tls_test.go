package tls

import (
	"crypto/tls"
	"testing"
)

func TestCertManagerGenerate(t *testing.T) {
	cm := NewCertManager("", "")
	err := cm.GenerateSelfSigned("localhost")
	if err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}
}

func TestCertManagerGetTLSConfig(t *testing.T) {
	cm := NewCertManager("", "")
	cm.GenerateSelfSigned("localhost")
	cfg := cm.GetTLSConfig("localhost")
	if cfg == nil {
		t.Fatal("GetTLSConfig returned nil")
	}
	if cfg.ServerName != "localhost" {
		t.Errorf("expected server name localhost, got %s", cfg.ServerName)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Error("should require TLS 1.2+")
	}
}

func TestDefaultConfig(t *testing.T) {
	if !DefaultConfig.Enabled {
		t.Error("default TLS config should be enabled")
	}
	if !DefaultConfig.AutoGen {
		t.Error("default should auto-gen certs")
	}
}
