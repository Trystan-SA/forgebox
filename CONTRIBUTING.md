# Contributing to ForgeBox

Thank you for your interest in contributing to ForgeBox. This guide covers everything you
need to set up your development environment, write quality code, and get your changes merged.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Code Style](#code-style)
- [Running Tests](#running-tests)
- [Pull Request Process](#pull-request-process)
- [Writing Plugins](#writing-plugins)
- [Building Custom Rootfs Images](#building-custom-rootfs-images)
- [Issue and PR Labels](#issue-and-pr-labels)
- [Code of Conduct](#code-of-conduct)
- [Communication Channels](#communication-channels)

## Development Environment Setup

### Prerequisites

| Dependency       | Minimum Version | Purpose                          |
|------------------|-----------------|----------------------------------|
| Go               | 1.23+           | Primary language                 |
| Linux kernel     | 5.10+           | KVM support for Firecracker      |
| Firecracker      | 1.6+            | MicroVM runtime                  |
| Docker           | 24+             | Building rootfs images           |
| Make             | 4.0+            | Build automation                 |
| golangci-lint    | 1.61+           | Linter                           |
| gofumpt          | 0.7+            | Code formatter                   |

### Getting Started

```bash
# Clone the repository
git clone https://github.com/forgebox-dev/forgebox.git
cd forgebox

# Install Go dependencies
make deps

# Verify KVM is available (required for integration tests)
ls -la /dev/kvm

# Download the default Firecracker kernel and rootfs
make fetch-assets

# Build all binaries
make build

# Run the quick smoke test
make test-unit
```

### KVM Access

Firecracker requires `/dev/kvm`. Add your user to the `kvm` group:

```bash
sudo usermod -aG kvm $USER
# Log out and back in for the group change to take effect
```

If you are developing on macOS or Windows, use the provided Vagrant or Docker-based
Linux VM in `deploy/dev-vm/`. Nested virtualization must be enabled.

### Editor Setup

We recommend VS Code or GoLand. A `.editorconfig` is provided at the repo root.
For VS Code, install the Go extension and set `"go.formatTool": "gofumpt"` in your
workspace settings.

## Code Style

ForgeBox follows strict Go style conventions enforced by CI.

- **Formatter:** `gofumpt` (a stricter superset of `gofmt`). Run `make fmt` before committing.
- **Linter:** `golangci-lint` with the configuration in `.golangci.yml`. Run `make lint`.
- **Error wrapping:** Always wrap errors with `fmt.Errorf("context: %w", err)`. Never discard errors silently.
- **Logging:** Use `log/slog` exclusively. No `fmt.Println` or `log.Printf` in production code.
- **Context:** Every exported function that performs I/O or may block must accept `context.Context` as its first parameter.
- **Naming:** Follow standard Go naming conventions. Avoid stutter (e.g., `vm.VMConfig` is wrong; `vm.Config` is correct).
- **Comments:** All exported types, functions, and methods must have doc comments.

```bash
# Format all code
make fmt

# Run the linter
make lint

# Run both
make check
```

## Running Tests

ForgeBox has three test tiers.

### Unit Tests

Unit tests do not require KVM or external services. They run on any OS.

```bash
make test-unit
```

### Integration Tests

Integration tests spin up real Firecracker microVMs and require `/dev/kvm`. They are
gated behind the `integration` build tag.

```bash
# Requires KVM
make test-integration
```

### End-to-End Tests

E2E tests exercise the full stack: gateway, scheduler, VM lifecycle, and plugin
execution. They require a running ForgeBox instance (started by the test harness).

```bash
# Requires KVM; takes several minutes
make test-e2e
```

### Running Everything

```bash
make test       # unit + integration
make test-all   # unit + integration + e2e
```

### Test Conventions

- Use table-driven tests wherever possible.
- Name test files `*_test.go` in the same package as the code under test.
- Integration tests must use the `//go:build integration` directive.
- E2E tests live in `test/e2e/` and use the `//go:build e2e` directive.
- Use `testify/assert` and `testify/require` for assertions.
- Provide `t.Helper()` in all test helper functions.

## Pull Request Process

1. **Fork** the repository and create a feature branch from `main`.
   ```bash
   git checkout -b feat/my-feature
   ```
2. **Write your code** following the style guide above.
3. **Add or update tests** to cover your changes.
4. **Run the full check suite:**
   ```bash
   make check
   make test
   ```
5. **Commit** with clear, descriptive messages. Use [Conventional Commits](https://www.conventionalcommits.org/) format:
   ```
   feat(scheduler): add priority queue for task ordering
   fix(gateway): handle nil auth token on WebSocket upgrade
   docs(plugins): add channel plugin example
   ```
6. **Push** and open a Pull Request against `main`.
7. **Fill out the PR template** completely. PRs missing the checklist will not be reviewed.
8. **Address review feedback** by pushing additional commits (do not force-push during review).

### PR Requirements

- All CI checks must pass (lint, unit tests, integration tests).
- At least one maintainer approval is required.
- Breaking changes require a discussion in GitHub Discussions before implementation.
- New features must include documentation updates.

## Writing Plugins

ForgeBox supports four plugin types:

| Type       | Interface         | Purpose                                      |
|------------|-------------------|----------------------------------------------|
| Provider   | `ProviderPlugin`  | Integrates LLM providers (OpenAI, Anthropic) |
| Channel    | `ChannelPlugin`   | Connects input sources (Slack, email, API)    |
| Tool       | `ToolPlugin`      | Exposes tools that run inside microVMs        |
| Storage    | `StoragePlugin`   | Backends for file and artifact persistence    |

All plugin interfaces are defined in `pkg/sdk/`. For a complete walkthrough with
code examples, see [docs/plugin-development.md](docs/plugin-development.md).

## Building Custom Rootfs Images

ForgeBox microVMs boot a minimal Linux rootfs. You can customize it for your tools.

```bash
# Build the default rootfs
make rootfs

# Build a custom rootfs with additional packages
cd rootfs/
cp default.Dockerfile my-custom.Dockerfile
# Edit my-custom.Dockerfile to add your packages
make rootfs IMAGE=my-custom
```

Rootfs images must:
- Be ext4 filesystem images under 512 MB (configurable).
- Include the ForgeBox guest agent binary at `/usr/local/bin/forgebox-agent`.
- Not run services as root. The agent runs as `forgebox` (UID 1000).

See `rootfs/README.md` for detailed instructions.

## Issue and PR Labels

| Label                | Description                                      |
|----------------------|--------------------------------------------------|
| `bug`                | Confirmed bug report                             |
| `enhancement`        | Feature request or improvement                   |
| `good first issue`   | Suitable for new contributors                    |
| `help wanted`        | Maintainers would appreciate community help      |
| `security`           | Security-related issue (may be restricted)        |
| `breaking-change`    | Introduces a breaking API or behavior change      |
| `needs-discussion`   | Requires community input before implementation    |
| `plugin`             | Related to the plugin system                      |
| `vm`                 | Related to Firecracker VM lifecycle               |
| `gateway`            | Related to the API gateway                        |
| `docs`               | Documentation improvements                        |
| `ci`                 | CI/CD pipeline changes                            |

## Code of Conduct

ForgeBox follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
We are committed to providing a welcoming and inclusive experience for everyone.
See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for the full text.

Report unacceptable behavior to conduct@forgebox.dev.

## Communication Channels

- **GitHub Discussions:** Design proposals, feature requests, and general questions.
  Use this as the primary async communication channel.
- **Discord:** Real-time chat for contributors. Invite link: https://discord.gg/forgebox
- **GitHub Issues:** Bug reports and actionable tasks only. Not for questions.

For security vulnerabilities, see [SECURITY.md](SECURITY.md). Do not open public issues
for security reports.
