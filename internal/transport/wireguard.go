package transport

import (
	"fmt"
	"log"
	"math"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type WireGuard struct {
	mu             sync.RWMutex
	name           string
	status         Status
	metrics        *Metrics
	remoteAddr     string
	iface          string
	privateKey     string
	remotePubKey   string
	localAddr      string
	configPath     string
}

func NewWireGuard() *WireGuard {
	return &WireGuard{
		name:   "wireguard",
		status: StatusInactive,
		metrics: &Metrics{},
		iface:  "nyxora0",
	}
}

func (w *WireGuard) Name() string { return w.name }
func (w *WireGuard) Type() string { return "wireguard" }

func (w *WireGuard) Init(cfg map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if key, ok := cfg["private_key"]; ok {
		w.privateKey = key
	}
	if pub, ok := cfg["remote_pub"]; ok {
		w.remotePubKey = pub
	}
	if addr, ok := cfg["local_addr"]; ok {
		w.localAddr = addr
	}
	if iface, ok := cfg["interface"]; ok {
		w.iface = iface
	}
	log.Printf("[wireguard] initialized (iface: %s)", w.iface)
	return nil
}

func (w *WireGuard) Connect(remoteAddr string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.remoteAddr = remoteAddr
	w.status = StatusTesting

	w.updateMetrics()

	if w.metrics.PacketLoss > 80 {
		w.status = StatusFailed
		return fmt.Errorf("high packet loss (%.1f%%), skipping wireguard", w.metrics.PacketLoss)
	}

	if !commandExists("wg") {
		log.Printf("[wireguard] wg not installed, skipping wg setup")
		w.status = StatusActive
		return nil
	}

	wgPort := 51820
	endpoint := formatEndpoint(remoteAddr, wgPort)
	conn, err := net.DialTimeout("udp", endpoint, 3*time.Second)
	if err != nil {
		log.Printf("[wireguard] remote WG port unreachable via %s: %v", endpoint, err)
		w.status = StatusActive
		return nil
	}
	conn.Close()

	cfg := w.generateConfig(remoteAddr)
	cfgPath := fmt.Sprintf("/etc/wireguard/%s.conf", w.iface)
	if err := writeConfig(cfgPath, cfg); err != nil {
		log.Printf("[wireguard] config write error: %v (continuing with ping-only)", err)
		w.status = StatusActive
		return nil
	}

	if commandExists("wg-quick") {
		down := exec.Command("wg-quick", "down", w.iface)
		down.Run()
		up := exec.Command("wg-quick", "up", w.iface)
		if out, err := up.CombinedOutput(); err != nil {
			log.Printf("[wireguard] wg-quick up failed: %v\n%s", err, string(out))
		} else {
			log.Printf("[wireguard] wg-quick up %s success", w.iface)
		}
	}

	w.configPath = cfgPath
	w.status = StatusActive
	log.Printf("[wireguard] connected to %s via %s", remoteAddr, w.iface)
	return nil
}

func (w *WireGuard) Disconnect() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if commandExists("wg-quick") {
		exec.Command("wg-quick", "down", w.iface).Run()
	}
	w.status = StatusInactive
	log.Printf("[wireguard] disconnected")
	return nil
}

func (w *WireGuard) Status() Status {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

func (w *WireGuard) Metrics() *Metrics {
	w.mu.RLock()
	defer w.mu.RUnlock()
	m := *w.metrics
	return &m
}

func (w *WireGuard) Health() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status == StatusActive
}

func (w *WireGuard) Score() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.updateMetrics()

	m := w.metrics
	if m.PacketLoss > 50 {
		return 0
	}
	if m.LatencyMs <= 0 {
		return 10
	}

	latencyScore := math.Max(0, 100-m.LatencyMs/2)
	lossScore := math.Max(0, 100-m.PacketLoss*2)
	jitterScore := math.Max(0, 100-m.JitterMs*3)
	stabilityScore := m.Stability * 100

	return latencyScore*0.35 + lossScore*0.30 + jitterScore*0.15 + stabilityScore*0.20
}

func (w *WireGuard) updateMetrics() {
	latency, loss, jitter := measureLatency(w.remoteAddr, 4)
	w.metrics.LatencyMs = latency
	w.metrics.PacketLoss = loss
	w.metrics.JitterMs = jitter

	if loss < 5 && latency < 100 {
		w.metrics.Stability = math.Min(1, w.metrics.Stability+0.1)
	} else {
		w.metrics.Stability = math.Max(0, w.metrics.Stability-0.1)
	}
}

func (w *WireGuard) generateConfig(remoteAddr string) string {
	privateKey := w.privateKey
	if privateKey == "" {
		out, _ := exec.Command("wg", "genkey").Output()
		if len(out) > 0 {
			privateKey = strings.TrimSpace(string(out))
		} else {
			privateKey = "nyxora-auto-key"
		}
	}

	pubKey := w.remotePubKey
	if pubKey == "" {
		pubKey = remoteAddr + "-pub"
	}
	localAddr := w.localAddr
	if localAddr == "" {
		localAddr = "10.100.0.2/24"
	}
	subnet := extractSubnet(localAddr)

	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1
MTU = 1420

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 10.100.%d.0/24
PersistentKeepalive = 25
`, privateKey, localAddr, pubKey, formatEndpoint(remoteAddr, 51820), subnet)
}

func extractSubnet(addr string) int {
	parts := strings.Split(addr, ".")
	if len(parts) >= 3 {
		var n int
		if _, err := fmt.Sscanf(parts[2], "%d", &n); err == nil && n >= 0 && n <= 255 {
			return n
		}
	}
	return 0
}

func formatEndpoint(addr string, port int) string {
	if strings.Contains(addr, ":") {
		return fmt.Sprintf("[%s]:%d", addr, port)
	}
	return fmt.Sprintf("%s:%d", addr, port)
}

var (
	commandCache = make(map[string]bool)
	commandMu    sync.Mutex
)

func commandExists(name string) bool {
	commandMu.Lock()
	defer commandMu.Unlock()
	if cached, ok := commandCache[name]; ok {
		return cached
	}
	_, err := exec.LookPath(name)
	commandCache[name] = err == nil
	return err == nil
}

func writeConfig(path, content string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("mkdir -p $(dirname %s) && cat > %s", path, path))
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func measureLatency(addr string, count int) (latency, packetLoss, jitter float64) {
	if addr == "" {
		return 999, 100, 999
	}
	var rtts []float64
	var lossCount float64
	for i := 0; i < count; i++ {
		start := time.Now()
		cmd := exec.Command("ping", "-c", "1", "-W", "2", addr)
		if err := cmd.Run(); err == nil {
			rtt := time.Since(start).Seconds() * 1000
			rtts = append(rtts, rtt)
		} else {
			lossCount++
		}
	}
	packetLoss = (lossCount / float64(count)) * 100
	if len(rtts) == 0 {
		return 999, 100, 999
	}
	var sum float64
	for _, r := range rtts {
		sum += r
	}
	latency = sum / float64(len(rtts))
	if len(rtts) > 1 {
		var jSum float64
		for i := 1; i < len(rtts); i++ {
			diff := rtts[i] - rtts[i-1]
			if diff < 0 {
				diff = -diff
			}
			jSum += diff
		}
		jitter = jSum / float64(len(rtts)-1)
	}
	return
}
