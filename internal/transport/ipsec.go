package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"sync"
)

type IPsec struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
}

func NewIPsec() *IPsec {
	return &IPsec{
		name:   "ipsec",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   500,
	}
}

func (i *IPsec) Name() string { return i.name }
func (i *IPsec) Type() string { return "ipsec" }

func (i *IPsec) Init(cfg map[string]string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &i.port)
	}
	log.Printf("[ipsec] initialized (port: %d)", i.port)
	return nil
}

func (i *IPsec) Connect(remoteAddr string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.remoteAddr = remoteAddr
	i.status = StatusTesting
	i.updateMetrics()

	if i.metrics.PacketLoss > 80 {
		i.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", i.metrics.PacketLoss)
	}

	if !commandExists("ipsec") {
		log.Printf("[ipsec] strongswan not installed, ping score only")
		i.status = StatusActive
		return nil
	}

	secret := fmt.Sprintf("%s : PSK \"nyxora-ipsec-psk\"\n", remoteAddr)
	secretPath := "/etc/ipsec.secrets"
	writeConfig(secretPath, secret)

	config := fmt.Sprintf(`conn nyxora
    left=%%any
    leftsubnet=0.0.0.0/0
    right=%s
    rightsubnet=0.0.0.0/0
    authby=secret
    ike=aes256-sha256-modp2048
    esp=aes256-sha256
    auto=start
`, remoteAddr)

	confPath := "/etc/ipsec.conf"
	writeConfig(confPath, config)

	go func() {
		cmd := exec.Command("ipsec", "restart")
		if err := cmd.Run(); err != nil {
			log.Printf("[ipsec] restart error: %v", err)
			i.mu.Lock()
			i.status = StatusFailed
			i.mu.Unlock()
			return
		}
		exec.Command("ipsec", "up", "nyxora").Run()
	}()

	log.Printf("[ipsec] connecting to %s", remoteAddr)
	i.status = StatusActive
	return nil
}

func (i *IPsec) Disconnect() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	exec.Command("ipsec", "down", "nyxora").Run()
	i.status = StatusInactive
	log.Printf("[ipsec] disconnected")
	return nil
}

func (i *IPsec) Status() Status {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.status
}

func (i *IPsec) Metrics() *Metrics {
	i.mu.RLock()
	defer i.mu.RUnlock()
	m := *i.metrics
	return &m
}

func (i *IPsec) Health() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.status == StatusActive
}

func (i *IPsec) Score() float64 {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.updateMetrics()
	m := i.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}
	return math.Max(0, 100-m.LatencyMs/2)*0.25 + math.Max(0, 100-m.PacketLoss*2)*0.25 +
		math.Max(0, 100-m.JitterMs*3)*0.20 + (m.Stability*100)*0.30
}

func (i *IPsec) updateMetrics() {
	lat, loss, jitter := measureLatency(i.remoteAddr, 3)
	i.metrics.LatencyMs = lat
	i.metrics.PacketLoss = loss
	i.metrics.JitterMs = jitter
	if loss < 5 && lat < 100 {
		i.metrics.Stability = math.Min(1, i.metrics.Stability+0.08)
	} else {
		i.metrics.Stability = math.Max(0, i.metrics.Stability-0.15)
	}
	i.metrics.Bandwidth = 500
}
