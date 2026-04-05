# ForgeBox Security Model

This document describes the security architecture of ForgeBox. It is intended for
operators evaluating ForgeBox for deployment, security researchers reviewing the
system, and contributors working on security-sensitive components.

## Table of Contents

- [Threat Model](#threat-model)
- [Defense Layers](#defense-layers)
- [Firecracker VM Isolation](#firecracker-vm-isolation)
- [In-VM Hardening](#in-vm-hardening)
- [Network Isolation](#network-isolation)
- [Permission Model](#permission-model)
- [Authentication and Authorization](#authentication-and-authorization)
- [Audit Logging](#audit-logging)
- [Supply Chain Security](#supply-chain-security)

## Threat Model

ForgeBox assumes the following threat actors and attack scenarios:

### What We Protect Against

1. **Malicious LLM output.** The LLM may produce arbitrary code, shell commands, or
   tool calls that attempt to access the host filesystem, exfiltrate data, mine
   cryptocurrency, or attack other services. This is the primary threat and the
   reason every tool execution happens inside a disposable microVM.

2. **Prompt injection leading to code execution.** An attacker may craft inputs
   (via Slack messages, emails, API calls) that hijack the LLM's instructions and
   cause it to execute unintended tool calls. ForgeBox treats all LLM-generated
   tool calls as untrusted regardless of the original prompt.

3. **User privilege escalation.** A non-admin user may attempt to access tasks,
   data, or configuration belonging to other users or teams. The permission model
   enforces strict boundaries.

4. **Data exfiltration from VMs.** Code running inside a VM may attempt to send
   sensitive data (API keys, documents, database contents) to external endpoints.
   Network isolation and domain allowlisting prevent this.

5. **Lateral movement.** A compromised VM may attempt to reach other VMs, the host,
   or internal services. VMs are network-isolated from each other and from host
   services by default.

6. **Supply chain attacks.** Compromised dependencies, tampered build artifacts, or
   malicious rootfs images could introduce vulnerabilities.

### What Is Out of Scope

- Physical access to the host machine.
- Compromise of the host operating system kernel (Firecracker relies on KVM; a KVM
  zero-day is out of scope for ForgeBox but mitigated by keeping kernels patched).
- Social engineering of administrators.

## Defense Layers

ForgeBox implements defense in depth with the following layers:

```
  User Request
       |
       v
  +-----------+     Authentication, rate limiting, input validation
  |  Gateway   |
  +-----------+
       |
       v
  +-----------+     Permission checks, task authorization
  | Scheduler  |
  +-----------+
       |
       v
  +-----------+     KVM hardware isolation, ephemeral VM
  | Firecracker|
  | MicroVM    |
  +-----------+
       |
       v
  +-----------+     seccomp-bpf, dropped capabilities, non-root, read-only rootfs
  | Guest Agent|
  +-----------+
       |
       v
  +-----------+     No network by default, domain allowlisting
  |  Network   |
  |  Policy    |
  +-----------+
```

Each layer is independent. A failure in one layer does not compromise the others.

## Firecracker VM Isolation

[Firecracker](https://firecracker-microvm.github.io/) is a virtual machine monitor
built by AWS for serverless workloads. ForgeBox chose it for these properties:

### KVM-Based Hardware Virtualization

Each task runs in its own microVM backed by Linux KVM. The VM has its own kernel,
its own memory space, and no shared filesystem with the host. This is fundamentally
stronger isolation than containers, which share the host kernel.

### Minimal Device Model

Firecracker exposes only the devices a VM needs: a virtio network device, a virtio
block device, a serial console, and a minimal keyboard controller for shutdown.
There is no GPU passthrough, no USB, no PCI, and no graphics. This drastically
reduces the attack surface compared to QEMU or full hypervisors.

### No Disk Passthrough

VMs receive a copy-on-write overlay of the rootfs. They cannot modify the base
image. Any writes are discarded when the VM is destroyed.

### No Network Passthrough

VMs do not share the host's network namespace. Each VM gets a TAP device connected
to a bridge that the ForgeBox network policy engine controls.

### Ephemeral VMs

VMs are created for a single task and destroyed immediately after completion. There
is no VM reuse across tasks, eliminating persistent compromise risks.

### Resource Limits

Each VM has hard limits on CPU time, memory, and disk I/O configured before boot.
A runaway process inside a VM cannot starve the host.

| Resource | Default Limit | Configurable |
|----------|---------------|--------------|
| vCPUs    | 1             | Yes          |
| Memory   | 256 MB        | Yes          |
| Disk     | 512 MB (CoW)  | Yes          |
| Wall time| 5 minutes     | Yes          |

## In-VM Hardening

Even inside the VM, ForgeBox applies defense in depth:

### seccomp-bpf Profile

The guest agent applies a seccomp-bpf filter that restricts the system calls
available to tool processes. The allowlist includes only the syscalls needed for
common tool operations (file I/O, process management, networking). Dangerous
syscalls like `mount`, `reboot`, `kexec_load`, `ptrace`, and `init_module` are
blocked.

The profile is defined in `internal/vm/seccomp/profile.go` and can be customized
per tool type.

### Dropped Capabilities

The tool process runs with all Linux capabilities dropped. Even if a process
escalates to UID 0 inside the VM (which should not be possible), it cannot perform
privileged operations.

### Non-Root Execution

The guest agent runs as the `forgebox` user (UID 1000). Tool processes are spawned
as the same user. The root account has no password and no login shell.

### Read-Only Root Filesystem

The base rootfs is mounted read-only. Tool processes can write to `/tmp` and
`/workspace` (a tmpfs), but cannot modify system binaries or configuration.

### No Package Manager

The production rootfs does not include a package manager. Tools must be pre-installed
in the rootfs image. This prevents a compromised process from installing additional
software.

## Network Isolation

### Default: No Network

By default, VMs have no network connectivity. The TAP device is created but not
connected to any bridge. Tools that do not need network access (file manipulation,
code analysis, data transformation) run fully offline.

### Opt-In Network with Domain Allowlisting

When a task requires network access, the task definition must explicitly declare it:

```yaml
task:
  name: fetch-weather
  network:
    enabled: true
    allowed_domains:
      - api.weather.gov
      - "*.openai.com"
    max_bandwidth: 10mbps
    max_connections: 5
```

The ForgeBox network policy engine configures iptables rules on the host bridge to
permit DNS resolution only for the listed domains and blocks all other outbound
traffic. IP-based bypasses are mitigated by resolving domains at the host level
and allowlisting only the resolved IPs.

### DNS Control

VMs use a ForgeBox-controlled DNS resolver that only resolves allowlisted domains.
Queries for non-allowlisted domains return NXDOMAIN.

## Permission Model

ForgeBox uses an additive, deny-by-default permission model.

### Principals

- **Users** belong to one or more **Teams**.
- **Teams** have **Roles** that grant permissions.
- **API keys** are scoped to specific permissions.

### Permissions

Permissions are granular and hierarchical:

```
tasks:create          - Create new tasks
tasks:read            - View task status and output
tasks:cancel          - Cancel running tasks
tools:shell           - Allow tasks to use the shell tool
tools:http            - Allow tasks to use the HTTP tool
network:internet      - Allow tasks to request internet access
admin:users           - Manage users
admin:config          - Modify system configuration
```

### Task-Level Restrictions

Each task inherits the permissions of the user who created it. A user with
`tools:shell` but without `network:internet` can create tasks that use the shell
tool but cannot grant those tasks internet access.

### Approval Workflows

High-risk actions (internet access, access to sensitive storage buckets) can be
configured to require explicit approval from a team admin before the task proceeds.

## Authentication and Authorization

### Gateway Authentication

The gateway supports multiple authentication methods:

- **API keys** with scoped permissions (for programmatic access).
- **OAuth 2.0 / OIDC** with configurable identity providers (for user login).
- **Webhook signatures** for channel plugins (Slack, GitHub, etc.).

All authentication happens at the gateway. Internal services communicate over
mutual TLS on a private network.

### Inter-Service Authentication

The gateway, scheduler, and any internal services authenticate to each other using
mutual TLS with short-lived certificates rotated automatically.

## Audit Logging

ForgeBox logs all security-relevant events in a structured, append-only audit log.

### Events Logged

- User authentication (success and failure).
- Task creation, execution, and completion.
- Permission checks (grants and denials).
- Configuration changes.
- VM lifecycle events (create, start, stop, destroy).
- Network policy changes.
- Plugin loading and initialization.

### Log Format

Audit logs are structured JSON emitted via `slog`. Each entry includes:

- Timestamp (RFC 3339, nanosecond precision).
- Event type.
- Actor (user ID, API key ID, or system).
- Resource (task ID, VM ID, etc.).
- Action and outcome (allowed/denied).
- Source IP address.

### Retention and Integrity

Audit logs should be shipped to an external log aggregator. ForgeBox does not provide
built-in tamper-proofing, but the structured format is compatible with append-only
storage backends and SIEM systems.

## Supply Chain Security

### Dependency Verification

- All Go dependencies are verified against `go.sum` checksums.
- Dependabot is enabled for automated dependency update PRs.
- Dependencies are reviewed for known vulnerabilities using `govulncheck` in CI.

### Build Reproducibility

- Release builds are produced by GitHub Actions with pinned runner images.
- Build inputs (Go version, dependency versions, build flags) are recorded in the
  release metadata.

### Software Bill of Materials (SBOM)

Every release includes an SBOM in SPDX format generated by `syft`. This allows
operators to inventory all components and check for vulnerabilities.

### Signed Releases

Release binaries and container images are signed using Sigstore (cosign). Operators
can verify the signature before deployment:

```bash
cosign verify-blob --signature forgebox-v0.5.0.sig --certificate-identity \
  https://github.com/forgebox-dev/forgebox/.github/workflows/release.yml@refs/tags/v0.5.0 \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  forgebox-v0.5.0-linux-amd64.tar.gz
```

### Rootfs Image Integrity

Rootfs images are built in CI from Dockerfiles checked into the repository. The
build process is deterministic. Image digests are recorded in the release manifest
so operators can verify they are using unmodified images.
