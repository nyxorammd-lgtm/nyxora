package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type CacheEntry struct {
	IP        net.IP    `json:"ip"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Resolver struct {
	mu          sync.RWMutex
	cache       map[string]*CacheEntry
	nameservers []string
	timeout     time.Duration
	cacheTTL    time.Duration
}

func NewResolver(nameservers []string, timeoutSec, cacheTTL int) *Resolver {
	if len(nameservers) == 0 {
		nameservers = []string{
			"1.1.1.1:53",
			"8.8.8.8:53",
			"9.9.9.9:53",
		}
	}
	if timeoutSec <= 0 {
		timeoutSec = 3
	}
	if cacheTTL <= 0 {
		cacheTTL = 300
	}
	return &Resolver{
		cache:       make(map[string]*CacheEntry),
		nameservers: nameservers,
		timeout:     time.Duration(timeoutSec) * time.Second,
		cacheTTL:    time.Duration(cacheTTL) * time.Second,
	}
}

func (r *Resolver) Lookup(host string) (net.IP, error) {
	r.mu.RLock()
	if entry, ok := r.cache[host]; ok && time.Now().Before(entry.ExpiresAt) {
		r.mu.RUnlock()
		return entry.IP, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.cache[host]; ok && time.Now().Before(entry.ExpiresAt) {
		return entry.IP, nil
	}

	for _, ns := range r.nameservers {
		ips, err := r.lookupUDP(host, ns)
		if err != nil {
			log.Printf("[dns] nameserver %s failed for %s: %v", ns, host, err)
			continue
		}
		if len(ips) > 0 {
			entry := &CacheEntry{
				IP:        ips[0],
				ExpiresAt: time.Now().Add(r.cacheTTL),
			}
			r.cache[host] = entry
			log.Printf("[dns] resolved %s -> %s via %s", host, ips[0], ns)
			return ips[0], nil
		}
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("dns lookup failed for %s: %w", host, err)
	}
	if len(ips) > 0 {
		entry := &CacheEntry{
			IP:        ips[0],
			ExpiresAt: time.Now().Add(r.cacheTTL),
		}
		r.cache[host] = entry
		return ips[0], nil
	}
	return nil, fmt.Errorf("no IPs found for %s", host)
}

func (r *Resolver) lookupUDP(host, nameserver string) ([]net.IP, error) {
	conn, err := net.DialTimeout("udp", nameserver, r.timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(r.timeout))

	msg := makeDNSQuery(host)
	_, err = conn.Write(msg)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return parseDNSResponse(buf[:n])
}

func (r *Resolver) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*CacheEntry)
}

func (r *Resolver) CacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.cache)
}

func (r *Resolver) Stats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return map[string]interface{}{
		"cache_size":     len(r.cache),
		"nameservers":    r.nameservers,
		"timeout_sec":    int(r.timeout.Seconds()),
		"cache_ttl_sec":  int(r.cacheTTL.Seconds()),
	}
}

func makeDNSQuery(domain string) []byte {
	msg := []byte{
		0x12, 0x34, // Transaction ID
		0x01, 0x00, // Flags: standard query
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answer RRs: 0
		0x00, 0x00, // Authority RRs: 0
		0x00, 0x00, // Additional RRs: 0
	}

	for _, label := range splitDomain(domain) {
		msg = append(msg, byte(len(label)))
		msg = append(msg, []byte(label)...)
	}
	msg = append(msg, 0x00) // root
	msg = append(msg, 0x00, 0x01) // Type A
	msg = append(msg, 0x00, 0x01) // Class IN
	return msg
}

func splitDomain(domain string) []string {
	var labels []string
	start := 0
	for i := 0; i < len(domain); i++ {
		if domain[i] == '.' {
			labels = append(labels, domain[start:i])
			start = i + 1
		}
	}
	if start < len(domain) {
		labels = append(labels, domain[start:])
	}
	return labels
}

func parseDNSResponse(data []byte) ([]net.IP, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("dns response too short")
	}

	ancount := int(data[6])<<8 | int(data[7])
	if ancount == 0 {
		return nil, fmt.Errorf("no answers in dns response")
	}

	offset := 12
	for i := 0; i < offset; i++ {
		if data[i] == 0 {
			offset = i + 5
			break
		}
	}

	var ips []net.IP
	for i := 0; i < ancount && offset+12 <= len(data); i++ {
		qtype := int(data[offset+2])<<8 | int(data[offset+3])
		rrlen := int(data[offset+10])<<8 | int(data[offset+11])
		offset += 12

		if qtype == 1 && rrlen == 4 && offset+4 <= len(data) {
			ip := net.IPv4(data[offset], data[offset+1], data[offset+2], data[offset+3])
			ips = append(ips, ip)
		}
		offset += rrlen
	}
	return ips, nil
}

type Config struct {
	Nameservers []string `json:"nameservers"`
	TimeoutSec  int      `json:"timeout_sec"`
	CacheTTL    int      `json:"cache_ttl"`
	Enabled     bool     `json:"enabled"`
}

var DefaultConfig = Config{
	Nameservers: []string{"1.1.1.1:53", "8.8.8.8:53"},
	TimeoutSec:  3,
	CacheTTL:    300,
	Enabled:     true,
}

func LoadConfig(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig, nil
	}
	return cfg, nil
}

type Monitor struct {
	resolver *Resolver
	ctx      context.Context
	cancel   context.CancelFunc
	interval time.Duration
	results  map[string]time.Duration
	mu       sync.RWMutex
}

func NewMonitor(resolver *Resolver, intervalSec int) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		resolver: resolver,
		ctx:      ctx,
		cancel:   cancel,
		interval: time.Duration(intervalSec) * time.Second,
		results:  make(map[string]time.Duration),
	}
}

func (m *Monitor) Start(hosts []string) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				for _, host := range hosts {
					start := time.Now()
					_, err := m.resolver.Lookup(host)
					elapsed := time.Since(start)
					m.mu.Lock()
					m.results[host] = elapsed
					m.mu.Unlock()
					if err != nil {
						log.Printf("[dns-monitor] %s: %v", host, err)
					} else {
						log.Printf("[dns-monitor] %s: %v", host, elapsed)
					}
				}
			}
		}
	}()
}

func (m *Monitor) Stop() {
	m.cancel()
}

func (m *Monitor) Results() map[string]time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]time.Duration, len(m.results))
	for k, v := range m.results {
		out[k] = v
	}
	return out
}
