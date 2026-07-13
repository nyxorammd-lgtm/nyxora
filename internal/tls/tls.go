package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"
	"time"
)

type CertManager struct {
	mu       sync.RWMutex
	certPath string
	keyPath  string
	cert     *tls.Certificate
	pool     *x509.CertPool
}

func NewCertManager(certPath, keyPath string) *CertManager {
	return &CertManager{
		certPath: certPath,
		keyPath:  keyPath,
		pool:     x509.NewCertPool(),
	}
}

func (cm *CertManager) GenerateSelfSigned(host string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"NyXoRa"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("load keypair: %w", err)
	}

	cm.mu.Lock()
	cm.cert = &cert
	cm.mu.Unlock()

	log.Printf("[tls] generated self-signed certificate for %s", host)
	return nil
}

func (cm *CertManager) GetTLSConfig(serverName string) *tls.Config {
	cm.mu.RLock()
	cert := cm.cert
	pool := cm.pool
	cm.mu.RUnlock()

	cfg := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	if serverName != "" {
		cfg.ServerName = serverName
	}
	return cfg
}

func WrapTCPListener(addr string, certManager *CertManager) (net.Listener, error) {
	tlsCfg := certManager.GetTLSConfig("")
	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("tls listen %s: %w", addr, err)
	}
	log.Printf("[tls] listening on %s", addr)
	return ln, nil
}

func DialTLS(addr, serverName string, certManager *CertManager) (*tls.Conn, error) {
	tlsCfg := certManager.GetTLSConfig(serverName)
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("tls dial %s: %w", addr, err)
	}
	return conn, nil
}

type Config struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	AutoGen  bool   `json:"auto_gen"`
}

var DefaultConfig = Config{
	Enabled: true,
	AutoGen: true,
}
