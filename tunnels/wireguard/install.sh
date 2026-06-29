#!/bin/bash
# WireGuard Tunnel Package for NYXORA

install_wireguard() {
    echo "[nyxora] installing wireguard..."
    if command -v apt &>/dev/null; then
        apt install -y -qq wireguard wireguard-tools
    elif command -v yum &>/dev/null; then
        yum install -y -q wireguard-tools
    elif command -v apk &>/dev/null; then
        apk add wireguard-tools
    else
        echo "[nyxora] no package manager found for wireguard"
        return 1
    fi
}

generate_keys() {
    local priv=$(wg genkey)
    local pub=$(echo "$priv" | wg pubkey)
    echo "private_key=$priv"
    echo "public_key=$pub"
}

create_config() {
    local remote=$1
    local interface=${2:-nyxora0}
    local private_key=$3

    cat > "/etc/wireguard/${interface}.conf" <<EOF
[Interface]
PrivateKey = ${private_key}
Address = 10.100.0.2/24
DNS = 1.1.1.1
MTU = 1420

[Peer]
PublicKey = ${remote}-pub
Endpoint = ${remote}:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
EOF
}

start_tunnel() {
    local interface=${1:-nyxora0}
    wg-quick up "$interface"
}

stop_tunnel() {
    local interface=${1:-nyxora0}
    wg-quick down "$interface" 2>/dev/null
}

case "${1:-}" in
    install) install_wireguard ;;
    genkeys) generate_keys ;;
    create-config) create_config "$2" "$3" "$4" ;;
    start) start_tunnel "$2" ;;
    stop) stop_tunnel "$2" ;;
    *)
        echo "usage: $0 {install|genkeys|create-config|start|stop}"
        exit 1
        ;;
esac
