# ForgeBox — Build AI tasks inside Firecracker microVMs
# -------------------------------------------------------

BINARY       := forgebox
AGENT_BINARY := fb-agent
MODULE       := github.com/forgebox/forgebox

# Version info injected at build time
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X $(MODULE)/internal/config.Version=$(VERSION) \
           -X $(MODULE)/internal/config.Commit=$(COMMIT) \
           -X $(MODULE)/internal/config.BuildDate=$(BUILD_DATE)

GO       := go
GOFLAGS  := -trimpath
TAGS     :=

BIN_DIR      := bin
INSTALL_DIR  := /usr/local/bin
SYSTEMD_DIR  := /etc/systemd/system

PROTO_DIR    := pkg/proto
ROOTFS_SCRIPT := scripts/build-rootfs.sh

# Docker
DOCKER_IMAGE  := forgebox
DOCKER_TAG    ?= $(VERSION)

.PHONY: all build build-agent dev dev-setup test test-integration test-e2e \
        lint proto rootfs install uninstall docker clean help

all: build

# ---------- Build -----------------------------------------------------------

build: $(BIN_DIR)/$(BINARY) $(BIN_DIR)/$(AGENT_BINARY)

$(BIN_DIR)/$(BINARY):
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) ./cmd/forgebox

$(BIN_DIR)/$(AGENT_BINARY):
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) \
		-ldflags '$(LDFLAGS) -extldflags "-static"' \
		-o $(BIN_DIR)/$(AGENT_BINARY) ./cmd/fb-agent

build-agent: $(BIN_DIR)/$(AGENT_BINARY)

# ---------- Development -----------------------------------------------------

dev: build
	@command -v watchexec >/dev/null 2>&1 && \
		watchexec -r -e go -- ./$(BIN_DIR)/$(BINARY) serve --config forgebox.yaml || \
	( command -v air >/dev/null 2>&1 && air || \
		echo "Install watchexec or air for hot-reload: cargo install watchexec-cli / go install github.com/air-verse/air@latest" )

dev-setup:
	@echo "==> Installing Go dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "==> Checking KVM availability..."
	@bash scripts/setup-kvm.sh || true
	@echo "==> Installing dev tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "==> Building rootfs..."
	@bash $(ROOTFS_SCRIPT) || echo "WARN: rootfs build skipped (requires root + Linux)"
	@echo "==> Starting development stack..."
	docker compose -f docker-compose.dev.yml up -d
	@echo "==> Dev setup complete. Services running on http://localhost:8420 (API) and http://localhost:3000 (dashboard)"

# ---------- Testing ---------------------------------------------------------

test:
	$(GO) test -race -count=1 ./internal/... ./pkg/... ./cmd/...

test-integration:
	$(GO) test -race -count=1 -tags=integration ./test/integration/...

test-e2e:
	$(GO) test -race -count=1 -tags=e2e -timeout 10m ./test/e2e/...

# ---------- Linting ---------------------------------------------------------

lint:
	golangci-lint run ./...

# ---------- Protobuf --------------------------------------------------------

proto:
	@command -v protoc >/dev/null 2>&1 || { echo "protoc not found — install protobuf compiler"; exit 1; }
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# ---------- Rootfs ----------------------------------------------------------

rootfs: build-agent
	@bash $(ROOTFS_SCRIPT)

# ---------- Install ---------------------------------------------------------

install: build
	install -m 0755 $(BIN_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@if [ -d /run/systemd/system ]; then \
		install -m 0644 deploy/systemd/forgebox.service $(SYSTEMD_DIR)/forgebox.service; \
		systemctl daemon-reload; \
		echo "Installed systemd unit. Enable with: systemctl enable --now forgebox"; \
	fi

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	rm -f $(SYSTEMD_DIR)/forgebox.service
	@if [ -d /run/systemd/system ]; then systemctl daemon-reload; fi

# ---------- Docker ----------------------------------------------------------

docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-f deploy/docker/Dockerfile \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# ---------- Clean -----------------------------------------------------------

clean:
	rm -rf $(BIN_DIR)
	rm -f rootfs/*.ext4
	rm -f *.sock
	$(GO) clean -testcache

# ---------- Help ------------------------------------------------------------

help:
	@echo "ForgeBox build targets:"
	@echo "  build            Build forgebox and fb-agent to bin/"
	@echo "  build-agent      Build fb-agent (statically linked)"
	@echo "  dev              Run with hot-reload (watchexec or air)"
	@echo "  dev-setup        Install deps, check KVM, build rootfs"
	@echo "  test             Run unit tests"
	@echo "  test-integration Run integration tests"
	@echo "  test-e2e         Run end-to-end tests"
	@echo "  lint             Run golangci-lint"
	@echo "  proto            Generate protobuf Go code"
	@echo "  rootfs           Build Firecracker rootfs image"
	@echo "  install          Install binary + systemd unit"
	@echo "  docker           Build Docker image"
	@echo "  clean            Remove build artifacts"
