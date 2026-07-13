FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /nyxora ./cmd/nyxora

FROM alpine:3.20

RUN apk add --no-cache bash curl iproute2 wireguard-tools openssh-client

COPY --from=builder /nyxora /usr/local/bin/nyxora

RUN mkdir -p /etc/nyxora/tunnels /etc/nyxora/cache /var/log/nyxora

ENTRYPOINT ["nyxora"]
CMD ["version"]
