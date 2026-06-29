package transport

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type OpenVPN struct {
	mu          sync.RWMutex
	name        string
	status      Status
	metrics     *Metrics
	remoteAddr  string
	port        int
	configPath  string
	cmd         *exec.Cmd
}

func NewOpenVPN() *OpenVPN {
	return &OpenVPN{
		name:   "openvpn",
		status: StatusInactive,
		metrics: &Metrics{},
		port:   1194,
	}
}

func (o *OpenVPN) Name() string { return o.name }
func (o *OpenVPN) Type() string { return "openvpn" }

func (o *OpenVPN) Init(cfg map[string]string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if p, ok := cfg["port"]; ok {
		fmt.Sscanf(p, "%d", &o.port)
	}
	log.Printf("[openvpn] initialized (port: %d)", o.port)
	return nil
}

func (o *OpenVPN) Connect(remoteAddr string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.remoteAddr = remoteAddr
	o.status = StatusTesting
	o.updateMetrics()

	if o.metrics.PacketLoss > 80 {
		o.status = StatusFailed
		return fmt.Errorf("high loss (%.1f%%)", o.metrics.PacketLoss)
	}

	if !commandExists("openvpn") {
		log.Printf("[openvpn] binary not found, scoring from ping only")
		o.status = StatusActive
		return nil
	}

	config := fmt.Sprintf(`client
dev tun
proto udp
remote %s %d
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
auth SHA256
cipher AES-256-CBC
verb 3
`, remoteAddr, o.port)

	tmpPath := fmt.Sprintf("/tmp/nyxora-openvpn-%d.conf", time.Now().UnixNano())
	if err := writeConfig(tmpPath, config); err != nil {
		log.Printf("[openvpn] config write error: %v", err)
		o.status = StatusFailed
		return err
	}

	go func() {
		cmd := exec.Command("openvpn", "--config", tmpPath, "--daemon")
		if err := cmd.Run(); err != nil {
			log.Printf("[openvpn] start error: %v", err)
			o.mu.Lock()
			o.status = StatusFailed
			o.mu.Unlock()
			return
		}
		o.mu.Lock()
		o.cmd = cmd
		o.status = StatusActive
		o.mu.Unlock()
		log.Printf("[openvpn] connected to %s:%d", remoteAddr, o.port)
	}()

	o.status = StatusActive
	return nil
}

func (o *OpenVPN) Disconnect() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	exec.Command("pkill", "-f", "openvpn.*nyxora").Run()
	o.status = StatusInactive
	log.Printf("[openvpn] disconnected")
	return nil
}

func (o *OpenVPN) Status() Status {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

func (o *OpenVPN) Metrics() *Metrics {
	o.mu.RLock()
	defer o.mu.RUnlock()
	m := *o.metrics
	return &m
}

func (o *OpenVPN) Health() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status == StatusActive
}

func (o *OpenVPN) Score() float64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.updateMetrics()

	m := o.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 5
	}

	latScore := math.Max(0, 100-m.LatencyMs/2)
	lossScore := math.Max(0, 100-m.PacketLoss*2)
	jitScore := math.Max(0, 100-m.JitterMs*3)
	staScore := m.Stability * 100

	return latScore*0.30 + lossScore*0.30 + jitScore*0.15 + staScore*0.25
}

func (o *OpenVPN) updateMetrics() {
	lat, loss, jitter := measureLatency(o.remoteAddr, 3)
	o.metrics.LatencyMs = lat
	o.metrics.PacketLoss = loss
	o.metrics.JitterMs = jitter

	if loss < 10 && lat < 200 {
		o.metrics.Stability = math.Min(1, o.metrics.Stability+0.05)
	} else {
		o.metrics.Stability = math.Max(0, o.metrics.Stability-0.1)
	}
	o.metrics.Bandwidth = 50
}

func init() {
	var _ = strings.TrimSpace
}
