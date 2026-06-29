#!/bin/bash
# SSH Tunnel Package for NYXORA

install_ssh() {
    echo "[nyxora] ssh is already available on the system"
    which ssh || { echo "ERROR: ssh not found"; return 1; }
}

create_tunnel() {
    local remote=$1
    local user=${2:-root}
    local port=${3:-22}
    local local_port=${4:-1080}

    ssh -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o ServerAliveInterval=10 \
        -o ServerAliveCountMax=3 \
        -N -D "127.0.0.1:${local_port}" \
        -p "$port" \
        "${user}@${remote}"
}

case "${1:-}" in
    install) install_ssh ;;
    create-tunnel) create_tunnel "$2" "$3" "$4" "$5" ;;
    *)
        echo "usage: $0 {install|create-tunnel <remote> [user] [port] [local-port]}"
        exit 1
        ;;
esac
