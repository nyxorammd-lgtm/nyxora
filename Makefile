.PHONY: build install clean run test vet tunnels daemon uninstall help lint docs release

BINARY=nyxora
INSTALL_PATH=/usr/local/bin
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "dev")

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/nyxora

install: build
	cp $(BINARY) $(INSTALL_PATH)/$(BINARY)
	mkdir -p /etc/nyxora/tunnels /etc/nyxora/cache /var/log/nyxora
	cp -r tunnels/*.tar.gz /etc/nyxora/tunnels/ 2>/dev/null || true

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf /etc/nyxora/cache/*

test:
	go test -v -count=1 -timeout 60s ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./... 2>/dev/null || go vet ./...

tunnels:
	@for dir in tunnels/*/; do \
		name=$$(basename $$dir); \
		echo "  packing $$name..."; \
		(cd tunnels && tar czf /etc/nyxora/tunnels/$$name.tar.gz $$name); \
	done

daemon: build
	systemctl daemon-reload
	systemctl enable nyxora
	systemctl restart nyxora
	systemctl status nyxora --no-pager

uninstall:
	systemctl stop nyxora 2>/dev/null || true
	systemctl disable nyxora 2>/dev/null || true
	rm -f $(INSTALL_PATH)/$(BINARY)
	rm -f /etc/systemd/system/nyxora.service
	rm -rf /etc/nyxora
	rm -rf /var/log/nyxora

docs:
	@echo "  generating documentation..."
	@mkdir -p docs
	@goreadme -export > docs/README.md 2>/dev/null || echo "  install goreadme: go install github.com/posener/goreadme/cmd/goreadme@latest"
	@echo "  done"

release: clean test vet build
	@echo "  building release $(VERSION)..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)_linux_amd64 ./cmd/nyxora
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY)_linux_arm64 ./cmd/nyxora
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY)_darwin_amd64 ./cmd/nyxora
	@echo "  release assets:"
	@ls -la $(BINARY)_*

help:
	@echo "  ┌─────────────────────────────────────────────────────┐"
	@echo "  │  NYXORA — Makefile Targets                          │"
	@echo "  ├─────────────────────────────────────────────────────┤"
	@echo "  │  build     Compile binary                           │"
	@echo "  │  install   Install to /usr/local/bin                │"
	@echo "  │  run       Build and run                            │"
	@echo "  │  test      Run all tests                            │"
	@echo "  │  vet       Run go vet                               │"
	@echo "  │  lint      Run golangci-lint / go vet               │"
	@echo "  │  clean     Remove binary and cache                  │"
	@echo "  │  tunnels   Package tunnel scripts                   │"
	@echo "  │  daemon    Setup systemd service                    │"
	@echo "  │  docs      Generate documentation                   │"
	@echo "  │  release   Build binaries for all platforms         │"
	@echo "  │  uninstall Remove NYXORA from system               │"
	@echo "  └─────────────────────────────────────────────────────┘"
