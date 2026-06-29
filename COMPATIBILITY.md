# Compatibility

## Local Server

| OS | Arch | Status |
|----|------|--------|
| Linux (Ubuntu 20.04+) | amd64 | ✅ Tested |
| Linux (Ubuntu 20.04+) | arm64 | ✅ Tested |
| Linux (Debian 11+) | amd64 | ✅ Tested |
| Linux (Debian 11+) | arm64 | ✅ Tested |
| Linux (CentOS 8+) | amd64 | ✅ Tested |
| macOS 13+ | amd64 | ✅ Builds |
| macOS 14+ (Apple Silicon) | arm64 | ✅ Builds |

## Remote Server (auto-provisioned)

| OS | Status |
|----|--------|
| Ubuntu 20.04+ | ✅ Tested |
| Ubuntu 22.04+ | ✅ Tested |
| Ubuntu 24.04+ | ✅ Tested |
| Debian 11+ | ✅ Tested |
| Debian 12+ | ✅ Tested |
| CentOS 8+ | ✅ Tested |
| Rocky Linux 9+ | ✅ Tested |

## Required Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| Go | 1.25+ | Compilation |
| sshpass | latest | SSH automation |
| wireguard-tools | latest | WireGuard setup |
| curl | latest | Downloads on remote |
| ping | any | Latency measurement |

## Transport Dependencies (installed automatically on remote)

| Transport | Remote Dependency |
|-----------|------------------|
| WireGuard | `wireguard-tools` |
| OpenVPN | `openvpn`, `easy-rsa` |
| FRP | `frps` (downloaded) |
| Rathole | `rathole` (downloaded) |
| Shadowsocks | `shadowsocks-rust` (downloaded) |
| Hysteria | `hysteria` (downloaded) |
| Backhaul | `backhaul` (downloaded) |
