# ForgeBox

Open-source AI automation platform. Runs AI-generated tasks inside Firecracker
microVMs so non-technical users in organizations can safely trigger AI automation
without risk to host infrastructure.

## Why ForgeBox

- **Safe by default** — every LLM-invoked tool runs inside an isolated Firecracker
  microVM. No tool execution on the host, ever.
- **No network unless granted** — VMs have no internet access unless the task
  definition explicitly allowlists specific domains.
- **Pluggable** — providers (OpenAI, Anthropic, ...), channels (Slack, webhook,
  email), tools, and storage backends are all plugins behind stable SDK interfaces.
- **Single entry point** — API, WebSocket, and admin dashboard all route through
  one gateway binary.

## Quick Start

Requires Docker. Everything else (Go, Node, Postgres) runs in containers.

```bash
# Set provider API keys
export ANTHROPIC_API_KEY=sk-...
export OPENAI_API_KEY=sk-...

# Bootstrap the first admin account
export FORGEBOX_FIRST_PASSWORD=change-me

# Start the stack
docker compose -f docker-compose.dev.yml up
```

Services:

| Service   | Port | Description                    |
|-----------|------|--------------------------------|
| backend   | 8420 | Go API server (gateway)        |
| backend   | 8421 | gRPC listener                  |
| dashboard | 3000 | Vite dev server (web UI)       |
| postgres  | 5432 | Optional; SQLite is the default |

Open http://localhost:3000 and sign in with `FORGEBOX_FIRST_PASSWORD`.

## Build Commands

```bash
make build            # Compile all binaries to ./bin/
make build-agent      # Build the in-VM guest agent (statically linked)
make dev              # Hot-reload dev server
make test             # Unit tests
make test-integration # Integration tests (requires KVM)
make test-e2e         # End-to-end tests (requires KVM)
make lint             # golangci-lint
make rootfs           # Build the default microVM rootfs image
```

## Repository Layout

```
cmd/                  # Binary entry points (gateway, scheduler, agent, CLI)
internal/             # Private application code
pkg/                  # Public libraries — plugin SDK, API client, permissions
plugins/              # Built-in providers, channels, tools, storage backends
web/                  # Admin dashboard (SvelteKit, embedded in gateway binary)
desktop/              # Tauri desktop shell
rootfs/               # Dockerfiles and scripts for microVM root filesystems
deploy/               # Docker Compose, Kubernetes, Terraform manifests
test/e2e/             # End-to-end test suites
docs/                 # Project documentation
```

## Architecture Principles

1. VM isolation is mandatory — no tool execution on the host.
2. The gateway is the only ingress path.
3. Deny-by-default networking and permissions; grants are explicit and additive.
4. Structured JSON logging via `slog` for audit trail and observability.
5. Plugin interfaces in `pkg/sdk/` are stable and versioned.

## Documentation

- [`CONTRIBUTING.md`](./CONTRIBUTING.md) — dev setup, code style, PR process, plugin authoring.
- [`SECURITY.md`](./SECURITY.md) — reporting vulnerabilities.
- [`web/DESIGN_SYSTEM.md`](./web/DESIGN_SYSTEM.md) — frontend design system.
- [`docs/`](./docs/) — architecture and operational docs.

## License

See [`LICENSE`](./LICENSE).