FROM alpine:3.21
RUN apk add --no-cache bash iproute2 wireguard-tools openssh-client sshpass curl ip6tables
COPY nyxora /usr/local/bin/nyxora
RUN chmod +x /usr/local/bin/nyxora && \
    mkdir -p /etc/nyxora/tunnels /etc/nyxora/cache /var/log/nyxora
ENTRYPOINT ["nyxora"]
CMD ["--help"]
