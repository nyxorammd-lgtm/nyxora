package transport

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type WebSocket struct {
	BaseTransport
	listener    net.Listener
 connections []net.Conn
	mu          sync.Mutex
}

func NewWebSocket() *WebSocket {
	w := &WebSocket{
		BaseTransport: NewBase("websocket", CatTunnel, 9925, ScoringWeights{
			Latency: 0.35, Loss: 0.25, Jitter: 0.15, Stability: 0.25,
		}, 500),
	}
	w.SetScoringFn(func() float64 {
		return ComputeScore(w.BaseMetrics(), w.weights)
	})
	return w
}

func (w *WebSocket) Init(cfg map[string]string) error {
	w.Logf("initialized (WebSocket transport)")
	return nil
}

func (w *WebSocket) Connect(remoteAddr string) error {
	if err := w.BaseConnectInit(remoteAddr); err != nil {
		return err
	}

	addr := FormatEndpoint(remoteAddr, w.BasePort())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		w.SetStatusFailed()
		return fmt.Errorf("websocket listen: %w", err)
	}
	w.listener = ln
	w.SetStatusActive()
	w.Logf("connected via WebSocket at %s", addr)

	go w.acceptLoop()
	return nil
}

func (w *WebSocket) acceptLoop() {
	for {
		select {
		case <-w.Context().Done():
			return
		default:
		}
		if w.listener == nil {
			return
		}
		conn, err := w.listener.Accept()
		if err != nil {
			continue
		}
		w.mu.Lock()
		w.connections = append(w.connections, conn)
		w.mu.Unlock()
		go w.handleConn(conn)
	}
}

func (w *WebSocket) handleConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-w.Context().Done():
			return
		default:
		}
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		_, err = conn.Write(buf[:n])
		if err != nil {
			return
		}
	}
}

func (w *WebSocket) Disconnect() error {
	w.mu.Lock()
	for _, conn := range w.connections {
		conn.Close()
	}
	w.connections = nil
	w.mu.Unlock()

	if w.listener != nil {
		w.listener.Close()
		w.listener = nil
	}
	w.BaseDisconnect()
	w.Logf("disconnected WebSocket")
	return nil
}

func (w *WebSocket) Name() string   { return w.BaseName() }
func (w *WebSocket) Type() string   { return w.BaseType() }
func (w *WebSocket) Status() Status { return w.BaseStatus() }
func (w *WebSocket) Metrics() *Metrics { return w.BaseMetrics() }
func (w *WebSocket) Health() bool { return w.BaseHealth() }
func (w *WebSocket) Score() float64 { return w.BaseScore() }

func (w *WebSocket) ServeWebSocketUpgrader(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	log.Printf("[websocket] server listening on %s", addr)
	<-ctx.Done()
	return nil
}
