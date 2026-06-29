# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.2.x   | ✅ Active support  |
| < 0.2   | ❌ Not supported   |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in NYXORA,
please **do not** open a public issue.

Instead, send a report to our Telegram channel: [@NyxoraCore](https://t.me/NyxoraCore)

We will acknowledge receipt within 48 hours, and provide a detailed response
within 7 days.

### What to include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact
- Suggested fix (if any)

## Security Best Practices

When using NYXORA:

1. **Use strong SSH passwords** or key-based authentication
2. **Change default secrets** — NYXORA auto-generates passwords for Shadowsocks,
   Rathole, Hysteria, Backhaul, and IPsec, but you should set them explicitly
   via environment variables for production use
3. **Firewall rules** — Both `nyxora install` and the connect process set up
   iptables rules to restrict tunnel access
4. **Update regularly** — Run `nyxora update` or check the GitHub releases page
5. **Run as non-root** when possible (though tunnel setup requires root for
   WireGuard and iptables)
