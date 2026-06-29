# NYXORA API Reference

## Package: `internal/config`

### Types
- `Config` — Main configuration struct
- `ServerMode` — Enum: ModeFull, ModeLite, ModeMinimal
- `DefaultConfig` — Default configuration preset

### Functions
- `Load(path string) (*Config, error)` — Load config from JSON file
- `(c *Config) Save(path string) error` — Save config to JSON file
- `(c *Config) Validate() error` — Validate configuration
- `(c *Config) GetEffectiveTransports() []string` — Get enabled transport list
- `ServerInfo() map[string]interface{}` — Get local server information
- `GetTransportsForMode(mode ServerMode) []string` — Get transports for a mode

## Package: `internal/transport`

### Interface: `Transport`
```go
type Transport interface {
    Name() string
    Type() string
    DefaultPort() int
    Start() error
    Stop() error
    Status() (string, error)
    Metrics() (*Metrics, error)
}
```

### Types
- `Metrics` — Latency, Loss, Score, BytesSent, BytesRecv
- `BaseTransport` — Shared transport logic
- `Manager` — Transport lifecycle manager
- `Registry` — Tunnel metadata registry

## Package: `internal/orchestrator`

### Types
- `Orchestrator` — Core orchestrator struct
- `StepStatus` — Status of a setup step

### Functions
- `New(cfg *config.Config) *Orchestrator`
- `(o *Orchestrator) Init() error`
- `(o *Orchestrator) Start() error`
- `(o *Orchestrator) Stop()`
- `(o *Orchestrator) ConnectToRemote(addr string, port int, user, password string) error`
- `(o *Orchestrator) Status() map[string]interface{}`
- `(o *Orchestrator) OnStepUpdate(fn func(StepStatus))`

## Package: `internal/multipath`

### Types
- `Scheduler` — Multipath scheduler
- `Mode` — Scheduling mode enum

### Modes
- `ModeWeighted` — Weighted distribution
- `ModeLowestLatency` — Lowest latency path
- `ModeLowestLoss` — Lowest loss path
- `ModeEven` — Equal distribution
- `ModeAll` — All active simultaneously

## Package: `internal/failover`

### Types
- `Engine` — Failover engine
- `State` — Transport state (healthy, degraded, down)

### Functions
- `NewEngine() *Engine`
- `(e *Engine) Evaluate(metrics map[string]*transport.Metrics) string`
- `(e *Engine) Reset()`

## Package: `internal/monitor`

### Types
- `Monitor` — Ping-based monitor

### Functions
- `NewMonitor() *Monitor`
- `(m *Monitor) Ping(addr string, count, timeoutSec int) (latency float64, loss float64, err error)`

## Package: `internal/dashboard`

### Types
- `TUI` — Terminal dashboard using ANSI escapes
- `StatusProvider` — Interface for providing status data

### Functions
- `NewTUI(intervalSec int) *TUI`
- `(t *TUI) SetProvider(p StatusProvider)`
- `(t *TUI) Start() error`
- `(t *TUI) Stop()`

## Package: `internal/interactive`

### Types
- `Theme` — Color theme configuration
- `TransportStatus` — Transport status for display

### Functions
- `RunMenu() (int, error)` — Launch interactive menu
- `RunTransportStatus(transports []TransportStatus) error` — Show transport status
- `RunUpdateChecker() error` — Check for updates

### Theme Colors (TrueColor hex)
| Theme | Primary | Surface | Success |
|-------|---------|---------|---------|
| Catppuccin Mocha | `#CBA6F7` | `#313244` | `#A6E3A1` |
| Tokyo Night | `#7AA2F7` | `#24283B` | `#9ECE6A` |
| Catppuccin Latte | `#8839EF` | `#E6E9EF` | `#40A02B` |
