# FAQ

## General

### What is NYXORA?
NYXORA is a self-healing multi-transport tunnel orchestrator. Install it on one server, and it automatically provisions, monitors, and fails over tunnels to a remote server via SSH.

### How is it different from Tailscale/WireGuard?
- **Tailscale** is a mesh VPN (all nodes are peers)
- **WireGuard** is a single tunnel protocol
- **NYXORA** is a tunnel **manager** — it runs 11 different tunnel types, scores them, and automatically fails over

### Do I need to install anything on the remote server?
No. NYXORA SSHs into the remote server and installs everything it needs automatically.

## Technical

### Why 11 transports?
Different networks block different protocols. Having 11 options maximizes the chance that at least one tunnel works.

### What is "scoring"?
Each transport is scored based on latency, packet loss, and a base weight. The orchestrator uses these scores to select the best tunnel and trigger failover.

### What happens if all tunnels fail?
NYXORA continuously retries all transports. When one comes back, it reconnects automatically.

### Can I add my own transport?
Yes. Implement the `Transport` interface in `internal/transport/` and register it in `registry.go`.

## Usage

### What OS are supported?
- **Local**: Linux (x86_64, arm64), macOS (x86_64, arm64)
- **Remote**: Ubuntu, Debian, CentOS (auto-detected)

### What's the minimum RAM?
- Minimal mode: 256MB
- Lite mode: 512MB
- Full mode: 2GB+

### Can I run this as a service?
Yes: `nyxora daemon` sets up a systemd service.

## Troubleshooting

### Connection fails with "SSH error"
Check: password correct? Port open? Root login enabled?

### Dashboard shows all tunnels as "failed"
The remote server might be blocking ports. Try `--mode lite` or manually specify transports with `--transports ssh,shadowsocks`.
