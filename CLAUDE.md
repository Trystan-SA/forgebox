# CLAUDE.md - AI Assistant Guidelines for ForgeBox

This file provides context for AI coding assistants working on the ForgeBox codebase.

## Product Specifications (MANDATORY)

All domain rules and features live in `/specs/`. Specs are the source of truth for what
the product does and how each feature must behave.

**File conventions:**
- Each spec file is a top-level **major chapter** (`X.x.x`) — e.g. `1.0.0-agents.md`,
  `2.0.0-brain.md`. One feature domain per file.
- **Inside** a file, headings use semantic versioning to describe the feature:
  - `## X.Y.0 — Minor` for each sub-capability of the feature.
  - `### X.Y.Z — Patch` for each rule, behavior, or refinement under that capability.

**Workflow — this is non-negotiable:**
1. Before implementing any change or new feature, **read the relevant spec files in
   `/specs/`** to confirm the current product rules.
2. If the spec already covers the work, implement strictly to spec. Do not invent
   behavior not in the spec.
3. If the work introduces new behavior or changes existing behavior:
   - **Update the matching spec file** (bump minor/patch headings accordingly), or
   - **Create a new spec file** for a new major chapter if the feature is new.
4. Ship spec updates in the same commit as the code change. Code without a matching
   spec update is incomplete.
5. When reviewing your own work before declaring it done, re-check the spec to
   confirm the implementation respects every rule it defines.

If `/specs/` does not yet exist or is missing a chapter relevant to the work, create
it as part of the change.

## Project Overview

ForgeBox is an open-source AI automation platform written in Go. It runs AI-generated
tasks inside Firecracker microVMs so that non-technical users in organizations can
safely trigger AI automation without risk to host infrastructure.

The codebase is a Go monorepo. All production code compiles to a small set of binaries.
The plugin system allows extending providers, tools, channels, and storage backends.

## Directory Layout

```
forgebox/
  cmd/                    # Binary entry points
    gateway/              # API gateway and web server (single entry point for all traffic)
    scheduler/            # Task scheduler and VM lifecycle manager
    agent/                # Guest agent that runs inside each microVM
    forgebox-cli/         # CLI for administrators and developers
  internal/               # Private application code (not importable by plugins)
    gateway/              # HTTP handlers, auth middleware, WebSocket upgrade
    scheduler/            # Task queue, VM pool management, resource accounting
    vm/                   # Firecracker VM creation, configuration, teardown
    auth/                 # Authentication and authorization logic
    audit/                # Audit log recording
    config/               # Configuration loading and validation
  pkg/                    # Public libraries (importable by plugins and external tools)
    sdk/                  # Plugin SDK: interfaces, types, helpers
    api/                  # API client library for ForgeBox's REST/gRPC API
    permissions/          # Permission model types and evaluation
  plugins/                # Built-in plugin implementations
    providers/            # LLM provider plugins (OpenAI, Anthropic, etc.)
    channels/             # Input channel plugins (Slack, webhook, email)
    tools/                # Tool plugins (shell, http, file, code-interpreter)
    storage/              # Storage backend plugins (local, S3, GCS)
  web/                    # Admin dashboard frontend (embedded in gateway binary)
  rootfs/                 # Dockerfiles and scripts for building VM root filesystems
  deploy/                 # Deployment manifests (Docker Compose, Kubernetes, Terraform)
  scripts/                # Build and development helper scripts
  test/                   # E2E and integration test suites
    e2e/                  # Full-stack end-to-end tests
    fixtures/             # Shared test fixtures
  docs/                   # Project documentation
```

## Running the Project

The recommended way to run ForgeBox locally is via Docker Compose:

```bash
docker compose -f docker-compose.dev.yml up
```

This starts three services with no local Go/Node install required:
- **backend** (Go API server) on port 8420
- **dashboard** (Vite dev server) on port 3000
- **postgres** on port 5432

Set API keys in your environment before starting:
```bash
export ANTHROPIC_API_KEY=sk-...
export OPENAI_API_KEY=sk-...
```

## Build Commands

```bash
make build            # Compile all binaries (forgebox + fb-agent) to ./bin/
make build-agent      # Build fb-agent only (statically linked)
make dev              # Run with hot-reload (requires watchexec or air)
make dev-setup        # Install Go deps, check KVM, install dev tools, build rootfs
make test             # Run unit tests
make test-integration # Run integration tests (requires KVM)
make test-e2e         # Run end-to-end tests (requires KVM, slower)
make lint             # Run golangci-lint
make proto            # Generate protobuf Go code
make rootfs           # Build the default microVM rootfs image
make docker           # Build Docker image
make clean            # Remove build artifacts
```

## Testing Conventions

- Write **table-driven tests** as the default pattern for all unit tests.
- Integration tests use the build tag `//go:build integration`.
- E2E tests use the build tag `//go:build e2e` and live in `test/e2e/`.
- Use `testify/assert` and `testify/require`. Do not use bare `if` checks in tests.
- Call `t.Helper()` in every test helper function.
- Mock external dependencies using interfaces, not concrete types. Mocks live in
  `internal/*/mocks/` generated by `mockgen`.
- Test files must be in the same package as the code under test (white-box testing).

## Code Style

- **Formatter:** `gofumpt` (not `gofmt`).
- **Errors:** Always wrap with `fmt.Errorf("context: %w", err)`. Never use `errors.New`
  for wrapping. Never ignore returned errors without explicit comment.
- **Logging:** Use `log/slog` everywhere. Pass the logger via `context.Context` or
  struct fields, never as a global.
- **Context:** `context.Context` is always the first parameter in functions that do I/O.
- **Naming:** No package-name stutter. Use `vm.Config`, not `vm.VMConfig`.
- **Struct initialization:** Use named fields. Never rely on positional initialization.
- **Concurrency:** Prefer channels over mutexes. Document goroutine ownership.

## Frontend Design System

The web dashboard has a design system documented in `web/DESIGN_SYSTEM.md`. **Always read
this file before creating or modifying frontend components.** It defines the color palette,
typography, spacing, component patterns (cards, inputs, buttons, badges), layout conventions,
and SCSS architecture. All frontend code must follow these guidelines.

## Important Interfaces

These are the core plugin interfaces in `pkg/sdk/`. Understand them before modifying
the plugin system.

- **`ProviderPlugin`** -- Wraps an LLM provider. Methods: `Complete`, `Stream`,
  `ListModels`, `Capabilities`.
- **`ChannelPlugin`** -- Ingests user requests. Methods: `Listen`, `Send`, `Capabilities`.
- **`ToolPlugin`** -- Defines a tool callable by the LLM inside a VM. Methods:
  `Definition`, `Execute`, `Validate`.
- **`StoragePlugin`** -- Persists files and artifacts. Methods: `Put`, `Get`, `List`,
  `Delete`, `Capabilities`.

All plugin interfaces include a `Name() string` and `Init(config map[string]any) error`
method. Plugins are registered in `plugins/registry.go`.

## Architecture Decisions

These are non-negotiable design principles. Do not introduce changes that violate them.

1. **VM isolation is mandatory.** All LLM-invoked tools execute inside a Firecracker
   microVM. No tool execution on the host. No exceptions.
2. **Gateway is the single entry point.** All external traffic (API, WebSocket, admin
   dashboard) routes through the gateway binary. Do not add additional ingress paths.
3. **No internet by default in VMs.** VMs have no network access unless the task
   definition explicitly grants it with an allowlisted set of domains.
4. **Least privilege everywhere.** Tasks run with the minimum permissions required.
   The permission model is additive (deny by default, explicit grants only).
5. **Structured logging only.** All logs are structured JSON via `slog`. This is
   required for the audit trail and observability pipeline.
6. **Plugin interfaces are stable.** Changes to `pkg/sdk/` interfaces require an RFC
   in GitHub Discussions and a deprecation cycle.
