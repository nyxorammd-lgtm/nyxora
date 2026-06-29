package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type Cloudflare struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	tunnelToken string
	cmd         *exec.Cmd
}

func NewCloudflare() *Cloudflare {
	return &Cloudflare{
		name:   "cloudflare",
		status: StatusInactive,
		metrics: &Metrics{},
	}
}

func (c *Cloudflare) Name() string { return c.name }
func (c *Cloudflare) Type() string { return "cloudflare" }

func (c *Cloudflare) Init(cfg map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := cfg["tunnel_token"]; ok {
		c.tunnelToken = t
	}
	log.Printf("[cloudflare] initialized")
	return nil
}

func (c *Cloudflare) Connect(remoteAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.remoteAddr = remoteAddr
	c.status = StatusTesting
	c.updateMetrics()

	if c.metrics.PacketLoss > 80 {
		c.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", c.metrics.PacketLoss)
	}

	if !commandExists("cloudflared") {
		log.Printf("[cloudflare] cloudflared not found, ping score only")
		c.status = StatusActive
		return nil
	}

	go func() {
		args := []string{"tunnel", "--no-autoupdate"}
		if c.tunnelToken != "" {
			args = append(args, "--token", c.tunnelToken)
		} else {
			args = append(args, "--url", "http://localhost:22")
		}

		cmd := exec.Command("cloudflared", args...)
		if err := cmd.Start(); err != nil {
			log.Printf("[cloudflare] start error: %v", err)
			return
		}
		c.mu.Lock()
		c.cmd = cmd
		c.mu.Unlock()
		cmd.Wait()
	}()

	log.Printf("[cloudflare] connecting to %s", remoteAddr)
	c.status = StatusActive
	return nil
}

func (c *Cloudflare) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd = nil
	}
	c.status = StatusInactive
	log.Printf("[cloudflare] disconnected")
	return nil
}

func (c *Cloudflare) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Cloudflare) Metrics() *Metrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := *c.metrics
	return &m
}

func (c *Cloudflare) Health() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status == StatusActive
}

func (c *Cloudflare) Score() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updateMetrics()
	m := c.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/3)*0.25 + math.Max(0, 100-m.PacketLoss*2)*0.25 +
		math.Max(0, 100-m.JitterMs*3)*0.15 + (m.Stability*100)*0.35
}

func (c *Cloudflare) updateMetrics() {
	lat, loss, jitter := measureLatency(c.remoteAddr, 3)
	c.metrics.LatencyMs = lat
	c.metrics.PacketLoss = loss
	c.metrics.JitterMs = jitter
	if loss < 15 && lat < 300 {
		c.metrics.Stability = math.Min(1, c.metrics.Stability+0.04)
	} else {
		c.metrics.Stability = math.Max(0, c.metrics.Stability-0.08)
	}
	c.metrics.Bandwidth = 80
}
