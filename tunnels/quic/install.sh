#!/bin/bash
# QUIC Tunnel Package for NYXORA

install_quic() {
    echo "[nyxora] installing quic tunnel..."
    if command -v apt &>/dev/null; then
        apt install -y -qq golang-github-lucas-clemente-quic-go-dev 2>/dev/null || true
    fi
    echo "[nyxora] quic tunnel uses built-in go quic support"
}

start_server() {
    local port=${1:-9923}
    echo "[nyxora] starting quic server on port $port"
    # Built-in QUIC server handled by nyxora agent
}

start_client() {
    local remote=$1
    local port=${2:-9923}
    echo "[nyxora] connecting quic client to $remote:$port"
}

case "${1:-}" in
    install) install_quic ;;
    start-server) start_server "$2" ;;
    start-client) start_client "$2" "$3" ;;
    *)
        echo "usage: $0 {install|start-server|start-client}"
        exit 1
        ;;
esac
