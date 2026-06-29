#!/bin/bash
# TCP Tunnel Package for NYXORA

install_tcp() {
    echo "[nyxora] tcp tunnel is built-in, no additional dependencies"
}

start_server() {
    local port=${1:-9924}
    echo "[nyxora] starting tcp tunnel server on port $port"
    nc -lk -p "$port" -e /bin/cat 2>/dev/null || \
    ncat -lk -p "$port" --sh-exec /bin/cat 2>/dev/null || \
    echo "[nyxora] install ncat for tcp tunnel server"
}

start_client() {
    local remote=$1
    local port=${2:-9924}
    echo "[nyxora] connecting tcp tunnel to $remote:$port"
}

case "${1:-}" in
    install) install_tcp ;;
    start-server) start_server "$2" ;;
    start-client) start_client "$2" "$3" ;;
    *)
        echo "usage: $0 {install|start-server|start-client}"
        exit 1
        ;;
esac
