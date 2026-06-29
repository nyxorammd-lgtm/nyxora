#!/usr/bin/env bash
set -euo pipefail

REPO="nyxorammd-lgtm/nyxora"
BINARY="nyxora"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/nyxora"

GREEN='\033[32m'
YELLOW='\033[33m'
CYAN='\033[36m'
RED='\033[31m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

info()  { echo -e "${CYAN}${BOLD}●${RESET} ${BOLD}$1${RESET}"; }
ok()    { echo -e "  ${GREEN}✓${RESET} $1"; }
warn()  { echo -e "  ${YELLOW}△${RESET} $1"; }
err()   { echo -e "  ${RED}✗${RESET} $1"; }

echo ""
echo -e "  ${BOLD}${CYAN}NYXORA Installer${RESET} ${DIM}v0.2.0${RESET}"
echo -e "  ${DIM}Adaptive Tunnel Orchestrator${RESET}"
echo ""

if [ "$(id -u)" -ne 0 ]; then
	err "this script must be run as root (sudo)"
	exit 1
fi

ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) err "unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
	linux) OS="linux" ;;
	darwin) OS="darwin" ;;
	*) err "unsupported OS: $OS"; exit 1 ;;
esac

if command -v curl &>/dev/null; then
	DOWNLOADER="curl -fsSL"
elif command -v wget &>/dev/null; then
	DOWNLOADER="wget -qO-"
else
	err "curl or wget required"
	exit 1
fi

info "Detecting latest version..."
API="https://api.github.com/repos/$REPO/releases/latest"
if command -v curl &>/dev/null; then
	TAG=$(curl -fsSL "$API" | grep '"tag_name"' | cut -d'"' -f4 2>/dev/null || echo "")
else
	TAG=$(wget -qO- "$API" | grep '"tag_name"' | cut -d'"' -f4 2>/dev/null || echo "")
fi

if [ -z "$TAG" ]; then
	warn "could not detect latest version, using v0.2.0"
	TAG="v0.2.0"
fi

VERSION="${TAG#v}"
URL="https://github.com/$REPO/releases/download/$TAG/${BINARY}_${OS}_${ARCH}"

info "Downloading NYXORA ${TAG} (${OS}/${ARCH})..."
if command -v curl &>/dev/null; then
	curl -fsSL "$URL" -o "/tmp/$BINARY"
else
	wget -q "$URL" -O "/tmp/$BINARY"
fi

chmod +x "/tmp/$BINARY"
mv "/tmp/$BINARY" "$INSTALL_DIR/$BINARY"
ok "Installed to $INSTALL_DIR/$BINARY"

mkdir -p "$CONFIG_DIR/tunnels" "$CONFIG_DIR/cache" "/var/log/nyxora"
ok "Created directories"

info "Dependencies check:"
for dep in ping wg ssh sshpass curl; do
	if command -v "$dep" &>/dev/null; then
		ok "$(printf '%-12s' "$dep") $(command -v "$dep")"
	else
		warn "$(printf '%-12s' "$dep") not found (apt install $dep)"
	fi
done

echo ""
info "Installation complete!"
echo ""
echo -e "  ${BOLD}Quick start:${RESET}"
echo -e "  ${DIM}  nyxora install           # setup config${RESET}"
echo -e "  ${DIM}  nyxora connect <ip>      # connect to remote${RESET}"
echo -e "  ${DIM}  nyxora tui               # interactive menu${RESET}"
echo -e "  ${DIM}  nyxora dashboard         # live monitoring${RESET}"
echo ""
echo -e "  ${DIM}  https://t.me/NyxoraCore${RESET}"
echo ""
