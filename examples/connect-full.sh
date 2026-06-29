#!/usr/bin/env bash
# Example: Connect with all options
# Replace IP and password with your actual values

nyxora connect 91.107.243.237 \
  --user root \
  --password YOUR_SSH_PASSWORD \
  --mode full \
  --transports wireguard,ssh,shadowsocks,hysteria,quic \
  --ports wg=51820,ss=8388,hy=8444
