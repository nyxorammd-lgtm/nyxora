# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-06-23

### Added
- Interactive Bubble Tea TUI with keyboard navigation
- 3 professional color themes (Catppuccin Mocha, Tokyo Night, Catppuccin Latte)
- ASCII art boot splash with animated gradient
- Connect wizard with step-by-step guidance
- Transport status viewer with animated score bars
- Live system monitoring (CPU, RAM, goroutines)
- Tunnel topology view
- Help screen with keyboard shortcuts
- TrueColor gradient rendering engine

### Changed
- Major refactor of internal architecture
- Dashboard now uses Catppuccin TrueColor palette
- Improved ANSI escape handling in dashboard
- Better error handling and user feedback

### Fixed
- WireGuard IPv6 endpoint formatting
- WireGuard remote key passing and subnet alignment
- WireGuard iptables rules to prevent SSH loss
- FRP/Rathole install scripts with API-based URL resolution

### Added
- Backhaul transport implementation
- All 11 transport types completed
- Single-side install flow
- TUI wizard for first-time setup

## [0.1.0] - 2026-06-01

### Added
- Initial MVP release
- Core orchestrator with connect/disconnect flow
- WireGuard transport with full remote provisioning
- SSH-based remote setup and management
- Ping-based monitoring system
- Automatic failover engine
- Multipath scheduler with 5 distribution modes
- Scoring engine (latency + packet loss)
- ANSI terminal dashboard
- Configuration management with JSON load/save
- Secret/token auto-generation
- Tar.gz packaging for tunnel assets
- 7 initial tunnel transports (WG, SSH, OpenVPN, QUIC, FRP, Rathole, TCP)
- Docker multi-stage build
- Makefile with build, test, install targets
