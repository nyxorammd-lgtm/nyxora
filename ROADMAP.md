# Roadmap

## v0.3.0 — Performance & Stability
- [ ] Connection pool for SSH sessions
- [ ] Parallel transport scoring (reduce initial connect time)
- [ ] Persistent state / crash recovery
- [ ] Benchmark suite for transport scoring
- [ ] Memory optimizations (reduce allocs in hot paths)

## v0.4.0 — Web UI
- [ ] Web dashboard (React/Vue + Go API server)
- [ ] Real-time WebSocket updates
- [ ] Remote server browser
- [ ] Tunnel config editor (web form)
- [ ] Mobile-responsive layout

## v0.5.0 — Advanced Networking
- [ ] TUN bonding (aggregate multiple tunnels into one interface)
- [ ] Per-packet load balancing
- [ ] NAT traversal / STUN support
- [ ] IPv6-only mode
- [ ] DNS-over-HTTPS for tunnel DNS

## v0.6.0 — Enterprise
- [ ] Multi-user RBAC
- [ ] Audit logging (all commands + config changes)
- [ ] LDAP/OIDC authentication
- [ ] SLA reporting (uptime %, failover time)
- [ ] Terraform provider for tunnel provisioning

## v1.0.0 — Stable Release
- [ ] 90%+ test coverage
- [ ] Fuzz testing for all transport parsers
- [ ] Formal security audit
- [ ] Signed releases (cosign + SLSA)
- [ ] LTS support policy
