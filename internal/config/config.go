package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	AgentID    string `json:"agent_id"`
	ListenAddr string `json:"listen_addr"`
	ServerMode bool   `json:"server_mode"`
	RemoteAddr string `json:"remote_addr"`

	MonitorInterval  int `json:"monitor_interval"`
	FailoverInterval int `json:"failover_interval"`
	StabilityWindow  int `json:"stability_window"`

	AllTunnelsActive bool `json:"all_tunnels_active"`
	MaxBandwidth     int  `json:"max_bandwidth"`
	DataDir          string `json:"data_dir"`
	LogLevel         string `json:"log_level"`
}

var DefaultConfig = Config{
	ListenAddr:       "0.0.0.0:9922",
	MonitorInterval:  30,
	FailoverInterval: 15,
	StabilityWindow:  5,
	AllTunnelsActive: false,
	MaxBandwidth:     1000,
	DataDir:          "/etc/nyxora",
	LogLevel:         "info",
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig
	if path == "" {
		path = "/etc/nyxora/config.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = filepath.Join(c.DataDir, "config.json")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
