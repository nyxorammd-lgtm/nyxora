# Transports

NYXORA supports 11 transport types, each with different characteristics.

## Transport Comparison

| Name | Protocol | Category | Use Case | Score |
|------|----------|----------|----------|-------|
| WireGuard | UDP | VPN | General purpose, fastest | 95 |
| OpenVPN | UDP | VPN | Legacy compatibility | 75 |
| SSH | TCP | Tunnel | Universal, works everywhere | 60 |
| QUIC | UDP | Tunnel | Anti-censorship, fast handshake | 80 |
| FRP | TCP | Relay | NAT traversal | 70 |
| Rathole | TCP | Relay | Rust-based, low latency | 85 |
| IPsec | UDP | VPN | Enterprise, native support | 70 |
| Shadowsocks | TCP | Proxy | Anti-censorship, obfuscation | 55 |
| Hysteria | UDP | Tunnel | Anti-censorship, modified QUIC | 90 |
| Backhaul | TCP | Relay | Reverse tunnel, simple setup | 82 |
| TCP | TCP | Tunnel | Raw TCP, fallback only | 50 |

## Scoring

Each transport is scored on a 0–100 scale based on:
- **Latency** (lower is better, 50% weight)
- **Packet loss** (lower is better, 30% weight)
- **Base weight** (protocol quality, 20% weight)

## Adding a Transport

1. Create `internal/transport/<name>.go`
2. Implement the `Transport` interface
3. Register in `internal/transport/registry.go`
4. Add install script in `tunnels/<name>/`
5. Submit a PR
