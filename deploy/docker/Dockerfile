# ForgeBox — Production multi-stage Dockerfile
# ==============================================

# ---- Stage 1: Builder -------------------------------------------------------
FROM golang:1.23-bookworm AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Build the main server binary
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-X github.com/forgebox/forgebox/internal/config.Version=${VERSION} \
              -X github.com/forgebox/forgebox/internal/config.Commit=${COMMIT} \
              -X github.com/forgebox/forgebox/internal/config.BuildDate=${BUILD_DATE} \
              -extldflags '-static'" \
    -o /out/forgebox ./cmd/forgebox

# Build the VM agent binary (static)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
    -ldflags "-X github.com/forgebox/forgebox/internal/config.Version=${VERSION} \
              -X github.com/forgebox/forgebox/internal/config.Commit=${COMMIT} \
              -X github.com/forgebox/forgebox/internal/config.BuildDate=${BUILD_DATE} \
              -extldflags '-static'" \
    -o /out/fb-agent ./cmd/fb-agent

# ---- Stage 2: Runtime -------------------------------------------------------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        iproute2 \
        iptables \
        curl \
    && rm -rf /var/lib/apt/lists/*

# Install Firecracker
ARG FIRECRACKER_VERSION=1.7.0
RUN ARCH=$(uname -m) && \
    curl -fsSL "https://github.com/firecracker-microvm/firecracker/releases/download/v${FIRECRACKER_VERSION}/firecracker-v${FIRECRACKER_VERSION}-${ARCH}.tgz" \
    | tar -xz -C /tmp && \
    mv /tmp/release-v${FIRECRACKER_VERSION}-${ARCH}/firecracker-v${FIRECRACKER_VERSION}-${ARCH} /usr/local/bin/firecracker && \
    mv /tmp/release-v${FIRECRACKER_VERSION}-${ARCH}/jailer-v${FIRECRACKER_VERSION}-${ARCH} /usr/local/bin/jailer && \
    chmod +x /usr/local/bin/firecracker /usr/local/bin/jailer && \
    rm -rf /tmp/release-*

COPY --from=builder /out/forgebox /usr/local/bin/forgebox
COPY --from=builder /out/fb-agent /usr/local/share/forgebox/fb-agent
COPY forgebox.example.yaml /etc/forgebox/forgebox.yaml

RUN useradd --system --no-create-home forgebox

EXPOSE 8420 8421

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8420/healthz || exit 1

USER forgebox
ENTRYPOINT ["forgebox"]
CMD ["serve", "--config", "/etc/forgebox/forgebox.yaml"]
