package transport

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"nhooyr.io/websocket"
)

type WebSocket struct {
	BaseTransport
	server     *http.Server
	listener   net.Listener
	connections []*websocket.Conn
	mu         sync.Mutex
}

func NewWebSocket() *WebSocket {
	w := &WebSocket{
		BaseTransport: NewBase("websocket", "websocket", 9925, ScoringWeights{
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
	ctx := w.CancelContext()
	w.KillOldProcess()
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

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", w.handleUpgrade)
	w.server = &http.Server{Handler: mux}

	go func() {
		if err := w.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			w.Logf("server error: %v", err)
		}
	}()

	w.SetStatusActive()
	w.Logf("connected via WebSocket at %s", addr)

	go func() {
		<-ctx.Done()
		w.server.Close()
	}()

	return nil
}

func (w *WebSocket) handleUpgrade(rw http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(rw, r, &websocket.AcceptOptions{
		Subprotocols: []string{"nyxora"},
	})
	if err != nil {
		w.Logf("websocket upgrade error: %v", err)
		return
	}

	w.mu.Lock()
	w.connections = append(w.connections, conn)
	w.mu.Unlock()

	w.handleConn(conn)
}

func (w *WebSocket) handleConn(conn *websocket.Conn) {
	defer func() {
		conn.Close(websocket.StatusNormalClosure, "done")
		w.mu.Lock()
		for i, c := range w.connections {
			if c == conn {
				w.connections = append(w.connections[:i], w.connections[i+1:]...)
				break
			}
		}
		w.mu.Unlock()
	}()

	ctx := conn.CloseRead(w.Context())
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msgType, reader, err := conn.Reader(ctx)
		if err != nil {
			return
		}
		if msgType != websocket.MessageBinary {
			continue
		}
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return
		}
		if n > 0 {
			wBuf, err := conn.Writer(ctx, websocket.MessageBinary)
			if err != nil {
				return
			}
			if _, err := wBuf.Write(buf[:n]); err != nil {
				return
			}
			wBuf.Close()
		}
	}
}

func (w *WebSocket) ConnectClient(remoteAddr string) error {
	ctx := w.CancelContext()
	w.KillOldProcess()
	if err := w.BaseConnectInit(remoteAddr); err != nil {
		return err
	}

	addr := fmt.Sprintf("ws://%s/ws", FormatEndpoint(remoteAddr, w.BasePort()))
	conn, _, err := websocket.Dial(ctx, addr, &websocket.DialOptions{
		Subprotocols: []string{"nyxora"},
	})
	if err != nil {
		w.Logf("websocket dial failed to %s: %v, falling back to ping-only", addr, err)
		w.SetStatusActive()
		return nil
	}

	w.mu.Lock()
	w.connections = append(w.connections, conn)
	w.mu.Unlock()

	w.SetStatusActive()
	w.Logf("client connected to %s via WebSocket", addr)

	go w.handleConn(conn)
	return nil
}

func (w *WebSocket) Disconnect() error {
	w.mu.Lock()
	conns := w.connections
	w.connections = nil
	w.mu.Unlock()

	for _, conn := range conns {
		conn.Close(websocket.StatusNormalClosure, "disconnect")
	}

	if w.server != nil {
		w.server.Close()
		w.server = nil
	}
	if w.listener != nil {
		w.listener.Close()
		w.listener = nil
	}
	w.BaseDisconnect()
	w.Logf("disconnected WebSocket")
	return nil
}

func (w *WebSocket) Name() string        { return w.BaseName() }
func (w *WebSocket) Type() string        { return w.BaseType() }
func (w *WebSocket) Status() Status      { return w.BaseStatus() }
func (w *WebSocket) Metrics() *Metrics   { return w.BaseMetrics() }
func (w *WebSocket) Health() bool        { return w.BaseHealth() }
func (w *WebSocket) Score() float64      { return w.BaseScore() }
