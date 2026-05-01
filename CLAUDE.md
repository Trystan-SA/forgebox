# CLAUDE.md — ForgeBox

## Specs (MANDATORY)

Source of truth lives in `/specs/`. One feature domain per file, named `X.0.0-<name>.md`.
Inside a file: `## X.Y.0` for sub-capabilities, `### X.Y.Z` for individual rules.

Workflow (non-negotiable):
1. Read the relevant spec before changing anything.
2. Implement strictly to spec — do not invent behavior.
3. New / changed behavior → update the spec (or create a new chapter) **in the same commit** as the code.
4. Re-check the spec before declaring work done.

If `/specs/` is missing a relevant chapter, create it.

## Overview

Open-source AI automation platform in Go. Runs AI-generated tasks inside Firecracker microVMs so non-technical users can trigger automation safely. Go monorepo, plugin system for providers / tools / channels / storage.

## Layout

```
cmd/                    binary entry points (gateway, scheduler, agent, forgebox-cli)
internal/               private app code (gateway, scheduler, vm, auth, audit, config)
pkg/                    public libs (sdk, api, permissions) — importable by plugins
plugins/                built-in plugins (providers, channels, tools, storage)
web/                    admin dashboard (embedded in gateway binary)
rootfs/                 microVM rootfs Dockerfiles + scripts
deploy/                 Compose / K8s / Terraform manifests
test/{e2e,fixtures}/    e2e + integration suites
specs/, docs/, scripts/
```

## Run / Build

Local dev (no Go/Node needed):
```bash
docker compose -f docker-compose.dev.yml up
# backend :8420, dashboard :3000, postgres :5432
# requires ANTHROPIC_API_KEY / OPENAI_API_KEY in env
```

Make targets: `build`, `build-agent`, `dev`, `dev-setup`, `test`, `test-integration`, `test-e2e`, `lint`, `proto`, `rootfs`, `docker`, `clean`.

## Testing

- Table-driven tests by default.
- Integration: `//go:build integration`. E2E: `//go:build e2e` (in `test/e2e/`).
- `testify/assert` + `testify/require` — no bare `if` checks. `t.Helper()` in every helper.
- Mock via interfaces only; mocks in `internal/*/mocks/` via `mockgen`.
- White-box: tests share the package under test.

## Code style

- Formatter: `gofumpt`.
- Errors: always `fmt.Errorf("context: %w", err)`. Never `errors.New` for wrapping. Never silently ignore errors.
- Logging: `log/slog` only. Pass via context or struct, never global.
- `context.Context` is the first parameter for any I/O function.
- No package-name stutter (`vm.Config`, not `vm.VMConfig`).
- Named struct fields only.
- Prefer channels over mutexes; document goroutine ownership.

## Frontend

Design system in `web/DESIGN_SYSTEM.md` — read it before creating or modifying components. Defines palette, typography, spacing, component patterns, layout, SCSS architecture. All frontend code must follow it.

## Plugin interfaces (`pkg/sdk/`)

Stable surface — changes require an RFC + deprecation cycle.

- `ProviderPlugin` — `Complete`, `Stream`, `ListModels`, `Capabilities`.
- `ChannelPlugin` — `Listen`, `Send`, `Capabilities`.
- `ToolPlugin` — `Definition`, `Execute`, `Validate`.
- `StoragePlugin` — `Put`, `Get`, `List`, `Delete`, `Capabilities`.

All plugins implement `Name() string` and `Init(map[string]any) error`. Registered in `plugins/registry.go`.

## Project skills

Local skills in `.claude/skills/`. Invoke via the `Skill` tool before acting on the trigger.

- **`create-pr-with-review`** — triggers on "create a PR" / "open a PR". Runs lint+build+test locally; opens the PR only if all pass; then dispatches a review subagent for Critical/High/Medium findings across security, code quality, architecture, and spec compliance.

New skills → `.claude/skills/<name>/SKILL.md` + one line here.

## Architecture (non-negotiable)

1. **VM isolation is mandatory.** All LLM-invoked tools run inside a Firecracker microVM. No host execution, no exceptions.
2. **Gateway is the only ingress.** API, WebSocket, dashboard — all through the gateway binary.
3. **No VM internet by default.** Tasks must explicitly allowlist domains.
4. **Least privilege.** Permission model is additive: deny by default, explicit grants only.
5. **Structured logging only** (`slog` JSON) — required for audit + observability.
6. **`pkg/sdk/` is stable.** Changes need an RFC + deprecation cycle.
