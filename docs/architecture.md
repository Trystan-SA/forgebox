# ForgeBox Architecture

> **Version:** 0.1.0-draft | **Status:** Living document | **License:** Apache 2.0

ForgeBox is an open-source AI automation platform written in Go. It runs AI agent
tasks inside Firecracker microVMs, providing hardware-level KVM isolation between the
host and any code the AI generates. It draws architectural inspiration from Claude Code
(tool-calling LLM loop, permission system, agent orchestration) and OpenClaw
(multi-channel gateway, plugin-first extensibility, broad LLM provider support).

---

## Table of Contents

1. [High-Level Overview](#1-high-level-overview)
2. [VM Orchestrator](#2-vm-orchestrator)
3. [Execution Engine](#3-execution-engine)
4. [Gateway Architecture](#4-gateway-architecture)
5. [Plugin System](#5-plugin-system)
6. [Permission System](#6-permission-system)
7. [Channel System](#7-channel-system)
8. [Session & Context Management](#8-session--context-management)
9. [In-VM Agent (fb-agent)](#9-in-vm-agent-fb-agent)
10. [Storage Architecture](#10-storage-architecture)
11. [Observability](#11-observability)
12. [Configuration System](#12-configuration-system)
13. [Security Model](#13-security-model)
14. [Deployment Patterns](#14-deployment-patterns)
15. [Technology Decisions](#15-technology-decisions)

---

## 1. High-Level Overview

A user request (natural-language prompt) enters through a channel, passes through the
gateway, drives the execution engine's LLM loop, and produces tool calls that execute
inside isolated Firecracker microVMs. Results stream back through the same path.

```
 User Request               Result
      |                       ^
      v                       |
 +----------+           +----------+
 | Channel  |           | Channel  |
 | (inbound)|           |(outbound)|
 +----+-----+           +----+-----+
      |                       |
      v                       |
 +----+-----+           +----+-----+
 | Gateway  +-----------+ Gateway  |
 | (HTTP /  |  events   | (SSE /   |
 |  gRPC)   |           |  stream) |
 +----+-----+           +----+-----+
      |                       ^
      v                       |
 +----+-----+           +----+-----+
 | Execution|  results  | Execution|
 | Engine   +-----------+ Engine   |
 | (LLM     |           | (result  |
 |  loop)   |           |  collect)|
 +----+-----+           +----+-----+
      |                       ^
      | tool calls            | tool results
      v                       |
 +----+-----+           +----+-----+
 |    VM    | vsock/gRPC |    VM    |
 | Orchestr.+-----------+ fb-agent |
 +----------+           +----------+
```

**Key components:**

| Component        | Package                 | Responsibility                                        |
|------------------|-------------------------|-------------------------------------------------------|
| Gateway          | `internal/gateway/`     | HTTP REST + gRPC entry point, auth, rate limiting     |
| Execution Engine | `internal/engine/`      | LLM tool-call loop, context assembly, cost tracking   |
| VM Orchestrator  | `internal/vm/`          | Firecracker lifecycle, pool, networking, vsock         |
| In-VM Agent      | `cmd/fb-agent/`         | Tool execution inside the VM, sandbox hardening       |
| Plugin Registry  | `internal/plugins/`     | Discovery, loading, and lifecycle of all plugin types  |
| Providers        | `internal/providers/`   | LLM provider adapters (Anthropic, OpenAI, Ollama)     |
| Channels         | `internal/channels/`    | Messaging integrations (Slack, Discord, webhook)      |
| Sessions         | `internal/sessions/`    | Multi-turn conversation state and transcript storage  |
| Permissions      | `internal/permissions/` | 5-layer permission evaluation and audit logging       |
| Storage          | `internal/storage/`     | Persistence backends (SQLite, PostgreSQL)              |
| Telemetry        | `internal/telemetry/`   | OpenTelemetry traces, Prometheus metrics, slog        |
| Config           | `internal/config/`      | Layered configuration loading and validation          |
| Scheduler        | `internal/scheduler/`   | Task queue and VM pool management                     |

**Binary outputs:** The `forgebox` binary (`cmd/forgebox/`) is the single host process
embedding the gateway, engine, and scheduler. The `fb-agent` binary (`cmd/fb-agent/`)
is cross-compiled for the guest and baked into the VM root filesystem.

---

## 2. VM Orchestrator

The VM orchestrator (`internal/vm/`) manages the full lifecycle of Firecracker microVMs.
Every tool call from an LLM executes inside one of these VMs -- no tool execution ever
happens on the host.

### Why Firecracker, Not Docker

| Property                 | Firecracker                         | Docker (runc)                       |
|--------------------------|-------------------------------------|-------------------------------------|
| Isolation boundary       | KVM hardware virtualization         | Linux namespaces + cgroups          |
| Kernel sharing           | Guest has its own kernel            | Shares host kernel                  |
| Attack surface           | ~50k LoC VMM, seccomp-locked       | Full container runtime + host kernel|
| Container escape risk    | Requires KVM escape (extremely rare)| Kernel exploits, misconfigured caps |
| Boot time (from snapshot)| ~5ms                                | ~200-500ms (cold)                   |

AI-generated code is untrusted by definition. An LLM might produce shell commands that
attempt privilege escalation or host filesystem access. Docker's shared-kernel model
means a single kernel vulnerability could let generated code break out. Firecracker's
KVM boundary makes the blast radius a single microVM with no path to the host kernel.

### Pool Management

Creating a Firecracker VM from scratch takes ~125ms. Restoring from a snapshot takes
~5ms. ForgeBox maintains a warm pool of pre-snapshotted VMs.

```
                    +-------------------+
                    |   Pool Manager    |
                    +--------+----------+
                             |
              +--------------+--------------+
              |              |              |
         +----+----+   +----+----+   +----+----+
         | Warm VM |   | Warm VM |   | Warm VM |    <- pre-snapshotted,
         | (idle)  |   | (idle)  |   | (idle)  |       ready in ~5ms
         +---------+   +---------+   +---------+
```

Pool lifecycle: boot fresh VM, wait for fb-agent healthy signal via vsock, pause and
snapshot (mem + vmstate). On demand: restore snapshot, attach overlay, configure
network namespace (if needed), connect vsock, go. Configurable pool sizes: `min_warm`
(default 5), `max_warm` (20), `max_active` (50).

### Filesystem: Base Image + Overlay

Each VM boots from a shared read-only ext4 base image. Task-specific writes go to a
copy-on-write overlay that is discarded when the VM terminates.

```
 +---------------------------+
 |  Overlay (ext4, per-task) |  <- writable, ephemeral, size-limited
 +---------------------------+
 |  Base Image (ext4, R/O)   |  <- shared across all VMs
 +---------------------------+
```

The base image (`rootfs/base/`) contains minimal Linux userspace, `fb-agent`, and
common tools (git, curl, python3, node, ripgrep). No SSH, no runtime package manager.
Custom images via `rootfs/custom/`.

### Networking

VMs have **no network access** by default. When a task explicitly requests it:

1. A Linux network namespace is created for the VM
2. A TAP device is bridged to the host via CNI
3. iptables rules allow only allowlisted domains
4. A DNS forwarder returns NXDOMAIN for non-allowlisted names

```
 Host Network NS              VM Network NS
 +-------------------+       +-------------------+
 | eth0              |       | eth0 (tap)        |
 |  +------+  veth   |       |  +------+         |
 |  | br0  +---------+-------+  | tap0 |         |
 |  +------+         |       |  +------+         |
 |  iptables:        |       |  fb-agent         |
 |  ALLOW allowlist  |       +-------------------+
 |  DROP *           |
 +-------------------+
```

### Host-Guest Communication (vsock)

Host and guest communicate over Virtio vsock -- no IP configuration, not affected by
network namespace restrictions, ~10us round-trip. The fb-agent listens on vsock CID 3,
port 6000 and exposes `AgentService` (defined in `pkg/proto/agent.proto`) with three
RPCs: `ExecuteTool`, `Heartbeat`, and `Shutdown`.

### Resource Budgets

| Resource   | Default | Enforcement                |
|------------|---------|----------------------------|
| vCPUs      | 1       | Firecracker machine config |
| RAM        | 256 MB  | Firecracker machine config |
| Disk       | 512 MB  | Overlay file size limit    |
| Wall time  | 300s    | Host-side timer            |
| Network BW | 1 Mbps  | Firecracker rate limiter   |

### Monitoring

- **Heartbeat:** Host polls every 5s; two misses trigger force-kill
- **OOM:** Detected via Firecracker metrics FIFO; VM terminated with clear error
- **Timeout:** Host-side goroutine enforces wall-clock budget
- **Zombie cleanup:** 30s sweep terminates orphaned VMs

---

## 3. Execution Engine

The execution engine (`internal/engine/`) implements the core LLM tool-call loop,
modeled after Claude Code's QueryEngine pattern.

### The Tool-Call Loop

```
                     +------------------+
                     | Assemble Context |
                     +--------+---------+
                              |
                              v
                   +----------+----------+
          +------->| Call LLM Provider   |
          |        | (streaming)         |
          |        +----------+----------+
          |                   |
          |         +---------+---------+
          |         |         |         |
          |     text delta  tool call  stop
          |         |         |         |
          |     stream to     |      return
          |     client        |      result
          |                   v
          |        +----------+----------+
          |        | Permission Check    |
          |        +----+----------+-----+
          |          allow        deny
          |            |            |
          |            v            v
          |        +---+----+  return denial
          |        | Execute|  to LLM as
          |        | in VM  |  tool error
          |        +---+----+
          |            |
          +-- append result to context
```

Each iteration: (1) assemble context from system prompt, session history, and project
files; (2) stream to the LLM provider via `ProviderPlugin.Stream()`; (3) forward text
deltas to the client in real time; (4) for tool calls, check permissions then dispatch
to the VM via `AgentService.ExecuteTool()`; (5) append results and loop. Terminates on
`finish_reason = "stop"` or budget exhaustion (turns, tokens, cost, wall time).

### Cost Tracking

Every LLM call returns `Usage` with token counts. The engine accumulates cost on the
`TaskRecord` and terminates if it exceeds `max_cost_usd`.

### Streaming Events

| Event             | When                                                 |
|-------------------|------------------------------------------------------|
| `StatusUpdate`    | Task transitions (pending -> running -> completed)   |
| `TextDelta`       | Incremental text from the LLM                       |
| `ToolCallEvent`   | LLM requests a tool invocation                      |
| `ToolResultEvent` | Tool returns a result                                |
| `ErrorEvent`      | Non-recoverable error                                |
| `DoneEvent`       | Task finished; includes duration and status          |

---

## 4. Gateway Architecture

The gateway (`internal/gateway/`) is the single entry point for all external traffic.
HTTP REST and gRPC are served on the same port using `cmux` for protocol multiplexing.

```
                          Port 8080
                             |
                          +--+--+
                          | cmux |
                          +--+--+
                     HTTP    |    gRPC
               +-------------+-------------+
               |                           |
        +------+------+            +-------+------+
        | HTTP Router |            | gRPC Server  |
        +------+------+            +-------+------+
               |                           |
    +----------+--------+       (ForgeBoxGateway service --
    |          |        |        mirrors REST endpoints
 +--+--+  +---+--+ +---+-+      with protobuf streaming)
 | REST|  | SSE  | | Web |
 | API |  |stream| | UI  |
 +-----+  +------+ +-----+
```

### REST Endpoints

| Method | Path                        | Description                   |
|--------|-----------------------------|-------------------------------|
| POST   | `/api/v1/tasks`             | Create task (returns SSE)     |
| GET    | `/api/v1/tasks/{id}`        | Get task status               |
| DELETE | `/api/v1/tasks/{id}`        | Cancel a running task         |
| POST   | `/api/v1/sessions/{id}/msg` | Send message to session (SSE) |
| GET    | `/api/v1/sessions/{id}`     | Get session with transcript   |
| GET    | `/api/v1/providers`         | List providers and models     |
| GET    | `/healthz` / `/readyz`      | Liveness / readiness probes   |

Streaming uses Server-Sent Events. Each `TaskEvent` is JSON-serialized:

```
event: text_delta
data: {"content":"Here is the file content..."}

event: tool_call
data: {"id":"tc_01","name":"bash","input":{"command":"ls -la"}}

event: done
data: {"task_id":"task_abc","status":"COMPLETED","total_duration_ms":4523}
```

### Middleware Stack

Applied in order in `internal/gateway/middleware/`:

1. **Request ID** -- X-Request-ID generation
2. **Structured Logging** -- slog with request ID, method, path
3. **Panic Recovery** -- catch panics, return 500, log stack trace
4. **CORS** -- configurable allowed origins
5. **Authentication** -- API key, JWT, or OAuth2 bearer token
6. **Rate Limiting** -- per-user token bucket
7. **Request Validation** -- payload size, content-type

The admin dashboard (`web/`) is compiled to static files, embedded via Go's `embed`
package, and served as a fallback route under `/`.

---

## 5. Plugin System

ForgeBox is built around four plugin interfaces in `pkg/sdk/`. Every major extension
point follows the same pattern.

### Plugin Types

| Type     | Interface        | Purpose                    | Examples                    |
|----------|------------------|----------------------------|-----------------------------|
| Provider | `ProviderPlugin` | LLM API integration        | Anthropic, OpenAI, Ollama   |
| Channel  | `ChannelPlugin`  | Messaging platform bridge  | Slack, Discord, webhook     |
| Tool     | `ToolPlugin`     | Actions the AI can perform | bash, file_read, web_search |
| Storage  | `StoragePlugin`  | Persistence backend        | SQLite, PostgreSQL, S3      |

All plugins implement the base `Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Init(ctx context.Context, config map[string]any) error
    Shutdown(ctx context.Context) error
}
```

### Loading Mechanisms

```
 +---------------------------------------------------------------+
 |                     Plugin Registry                            |
 |                 (internal/plugins/)                             |
 +-------+--------+--------+--------+---------------------------+
         |        |        |        |
    +----+--+ +---+---+ +--+----+ +-+----------+
    |Compiled| |Go .so | |gRPC   | |MCP Server |
    |in-tree | |plugin | |sidecar| |auto-disc. |
    +--------+ +-------+ +-------+ +------------+
```

1. **Compiled-in:** Plugins in `plugins/` are compiled into the binary and register
   via `init()` calling `registry.Register()`. Default for first-party plugins.

2. **Go plugin (.so):** Dynamically loaded at startup from a configurable directory.
   Must export `NewPlugin() sdk.Plugin`.

3. **gRPC sidecar:** Out-of-process plugins implementing `PluginService` plus a
   type-specific service from `pkg/proto/plugin.proto`. Allows plugins in any
   language. Host manages the sidecar process lifecycle.

4. **MCP server auto-discovery:** `internal/mcp/` scans for MCP server configs and
   wraps each as a `ToolPlugin`, providing access to the MCP tool ecosystem.

### Plugin Lifecycle

`Discover -> Init(cfg) -> Serving -> Shutdown()`. Init errors are logged and the plugin
is skipped. Shutdown has a 10s grace period; sidecars get SIGTERM then SIGKILL.

### Plugin SDK (`pkg/sdk/`)

| File           | Interface        | Key Methods                                    |
|----------------|------------------|------------------------------------------------|
| `plugin.go`    | `Plugin`         | `Name`, `Version`, `Init`, `Shutdown`          |
| `provider.go`  | `ProviderPlugin` | `Models`, `Stream`, `Complete`                 |
| `channel.go`   | `ChannelPlugin`  | `Listen`, `Send`                               |
| `tool.go`      | `ToolPlugin`     | `Schema`, `Execute`, `IsReadOnly`, `IsDestructive` |
| `storage.go`   | `StoragePlugin`  | `TaskStore`, `SessionStore`, `AuditStore`, `UserStore` |

Changes to `pkg/sdk/` interfaces require an RFC and deprecation cycle.

---

## 6. Permission System

Five layers, evaluated in order. Deny-by-default -- access must be explicitly granted.

```
 +--------------------------------------------------+
 | Layer 1: Organization Policies                    |
 | (global: "no shell access", "no internet")        |
 +--------------------------------------------------+
                      | pass?
 +--------------------------------------------------+
 | Layer 2: RBAC                                     |
 | (admin, developer, operator, viewer)              |
 +--------------------------------------------------+
                      | pass?
 +--------------------------------------------------+
 | Layer 3: Task-Scoped Permissions                  |
 | (per-task grants: "this task may use bash")       |
 +--------------------------------------------------+
                      | pass?
 +--------------------------------------------------+
 | Layer 4: Tool-Level Classification                |
 | (read-only vs. write vs. destructive)             |
 +--------------------------------------------------+
                      | pass?
 +--------------------------------------------------+
 | Layer 5: VM-Enforced Sandbox                      |
 | (seccomp, dropped capabilities, non-root)         |
 +--------------------------------------------------+
```

**Layer 1 -- Org Policies:** Administrator-defined rules that cannot be overridden.
Example: `deny_tools: [bash]`, `max_cost_per_task_usd: 5.00`.

**Layer 2 -- RBAC:** Four roles with additive permissions:

| Role      | Create Tasks | Shell | Network | View Audit | Manage Users |
|-----------|-------------|-------|---------|------------|--------------|
| admin     | yes         | yes   | yes     | yes        | yes          |
| developer | yes         | yes   | config. | yes        | no           |
| operator  | yes         | no    | no      | yes        | no           |
| viewer    | no          | no    | no      | read-only  | no           |

**Layer 3 -- Task-Scoped:** Callers request capabilities at task creation; granted
only if allowed by RBAC and org policy.

**Layer 4 -- Tool Classification:** Each tool reports risk via `IsReadOnly()` /
`IsDestructive()`:

| Classification | Examples                 | Behavior                     |
|----------------|--------------------------|------------------------------|
| Read-only      | file_read, grep, ls      | Execute immediately          |
| Write          | file_write, git_commit   | Execute with logging         |
| Destructive    | bash (rm -rf), git_push  | Require explicit approval    |

**Layer 5 -- VM Sandbox:** Even if all software layers are bypassed, the VM enforces
seccomp (blocks mount, reboot, ptrace, etc.), dropped capabilities, non-root (UID 1000),
and read-only base filesystem.

### Audit Logging

Every permission decision is persisted as an append-only `AuditEntry` (defined in
`pkg/sdk/storage.go`) recording user, task, action, tool, decision, and reason.

---

## 7. Channel System

Channels (`internal/channels/`) implement `ChannelPlugin` and handle bidirectional
communication with external messaging platforms.

### Inbound Flow

```
 Slack: "@forgebot refactor auth"
          |
 +--------+--------+
 | Slack Channel    |   Listen() pushes InboundMessage
 +--------+--------+   to the registered MessageHandler
          |
 +--------+--------+
 | Gateway          |   Creates a task from the message
 +--------+--------+
          |
 +--------+--------+
 | Engine loop      |   Streams TaskEvents
 +--------+--------+
          |
 +--------+--------+
 | Slack Channel    |   Send() formats results as thread reply
 +--------+--------+
```

### Approval Flow for Destructive Actions

When the engine encounters a destructive tool call requiring approval (Layer 4), it
pauses and sends an approval request through the originating channel:

```
 Engine                    Slack                     User
   |                          |                        |
   | "bash: rm -rf /tmp/old"  |                        |
   +--- ApprovalRequest ----->|--- message ----------->|
   |                          |  [Approve] [Deny]      |
   |<-- ApprovalResponse -----|<-- button click -------|
   |                          |                        |
   | (approved: execute)      |                        |
   | (denied: tell LLM)       |                        |
```

The engine blocks with a configurable timeout (default 5 min) waiting for the response.
For Slack, this uses interactive message buttons. For REST, a separate approval endpoint.

---

## 8. Session & Context Management

### Session Lifecycle

A session is a multi-turn conversation. Follow-up messages share full history.

```
 +-----+-----+-----+-----+
 |Turn1|Turn2|Turn3|Turn4| ...     Each user turn triggers a task.
 | user| asst| user| asst|         Session persists across tasks.
 +-----+-----+-----+-----+
```

States: **active** (in use), **idle** (no activity >30 min, VM resources released),
**archived** (past retention period, read-only).

### Transcript Storage

Every message is persisted via `SessionStore.AppendMessage()` with role, content, tool
calls, tool results, and timestamp. Full transcripts are available for replay.

### Context Assembly and Token Budgeting

Before each LLM call, context is assembled from four layers:

1. System prompt (role definition, tool descriptions, constraints)
2. Project context (CLAUDE.md-style files, relevant code snippets)
3. Conversation history (all prior turns)
4. Current user message

Token budget calculation for a 200k-token model:

```
 Total window:     200,000
 System prompt:      2,000 (fixed)
 Tool definitions:   3,000 (fixed)
 Output reserve:     4,096 (configurable)
 -----------------------------------
 History budget:   190,904

 If history exceeds budget:
   1. Summarize oldest turns; keep last 10 verbatim
   2. Replace old tool results with one-line summaries
   3. If still over: progressive summarization (cached)
```

---

## 9. In-VM Agent (fb-agent)

The `fb-agent` binary runs inside every Firecracker microVM as PID 1 (or launched by
a minimal init). It receives tool invocations from the host over vsock and executes them.

```
 +-----------------------------------------------+
 | Firecracker microVM                            |
 |  +------------------+                          |
 |  | fb-agent (PID 1) |                          |
 |  +--------+---------+                          |
 |    +------+------+------+------+               |
 |  +-+-+ +-+-+ +-+-+  +-+-+  +-+-+              |
 |  |bash| |file| |grep|  |web |  |git|           |
 |  |exec| |ops | |    |  |fetch| |   |           |
 |  +---+  +---+  +---+  +---+  +---+            |
 |  +------------------------------------------+  |
 |  | Overlay FS (writable)                    |  |
 |  +------------------------------------------+  |
 |  | Base Image (read-only)                   |  |
 |  +------------------------------------------+  |
 +-----------------------------------------------+
```

### Startup

1. Boot, start fb-agent as PID 1
2. Drop all Linux capabilities, apply seccomp profile
3. Open vsock listener on port 6000, start gRPC `AgentService`
4. Host connects and confirms readiness via `Heartbeat()`

### Tool Execution

On `ExecuteTool()`: look up tool, deserialize input, spawn subprocess (for bash/git)
or execute in-process (for file ops), enforce timeout, capture stdout/stderr/exit code,
return result via gRPC.

| Tool         | Type        | Description                         |
|--------------|-------------|-------------------------------------|
| `bash`       | subprocess  | Execute a shell command              |
| `file_read`  | in-process  | Read file contents                   |
| `file_write` | in-process  | Write or patch a file                |
| `file_edit`  | in-process  | Targeted string replacement          |
| `grep`       | subprocess  | Search file contents (via ripgrep)   |
| `glob`       | in-process  | Find files by pattern                |
| `web_fetch`  | in-process  | Fetch a URL (if network allowed)     |
| `git`        | subprocess  | Git operations                       |

### Sandbox Hardening

- **seccomp:** Strict allowlist of ~60 syscalls. Blocked: `mount`, `reboot`, `ptrace`,
  `init_module`, `unshare`, `setns`, `kexec_load`
- **Capabilities:** All dropped after startup; empty capability set
- **User:** UID 1000 (non-root), no root password or login shell
- **Filesystem:** Base image read-only, overlay writable with size limit,
  `/proc` mounted with `hidepid=2`

---

## 10. Storage Architecture

Pluggable storage via `StoragePlugin` (`internal/storage/`).

### Backends

| Backend    | Package                      | Use Case                |
|------------|------------------------------|-------------------------|
| SQLite     | `internal/storage/sqlite/`   | Development, single-node|
| PostgreSQL | `internal/storage/postgres/` | Production, multi-node  |
| S3 / MinIO | (artifact storage only)      | Task artifacts, VM logs |

### Data Model

```
 users 1---N sessions 1---N tasks
                |                \
                | 1---N           \--- audit_entries
                v
            messages
            (role, content, tool_calls, tool_results, timestamp)
```

Key stored entities:

| Entity        | Key Fields                                                      |
|---------------|-----------------------------------------------------------------|
| `TaskRecord`  | id, status, prompt, result, provider, model, cost, tokens, timestamps |
| `SessionRecord` | id, user_id, provider, model, timestamps                    |
| `Message`     | session_id, role, content, tool_calls, tool_results, timestamp  |
| `AuditEntry`  | id, timestamp, user_id, task_id, action, tool, decision, reason |
| `UserRecord`  | id, name, email, role, team_ids, disabled                       |

### Artifact Storage

Large outputs and VM logs go to object storage: `tasks/{task_id}/artifacts/{filename}`
and `tasks/{task_id}/vm.log`. Configured independently from the relational backend.

---

## 11. Observability

### OpenTelemetry Traces

Every significant operation produces a span via `internal/telemetry/`, exported over
OTLP to Jaeger, Grafana Tempo, or any compatible backend.

| Span                      | Attributes                               |
|---------------------------|------------------------------------------|
| `gateway.CreateTask`      | user_id, provider, model                 |
| `engine.Run`              | session_id, task_id                      |
| `engine.LLMCall`          | model, input_tokens, output_tokens       |
| `engine.ToolDispatch`     | tool_name, is_read_only                  |
| `vm.Acquire`              | pool_size, was_warm                      |
| `vm.ExecuteTool`          | tool_name, duration_ms, exit_code        |
| `permission.Evaluate`     | layers_checked, decision                 |

### Prometheus Metrics (`/metrics`)

| Metric                                     | Type      | Labels                       |
|--------------------------------------------|-----------|------------------------------|
| `forgebox_vm_pool_warm_count`              | gauge     |                              |
| `forgebox_vm_pool_active_count`            | gauge     |                              |
| `forgebox_vm_boot_duration_seconds`        | histogram |                              |
| `forgebox_engine_tasks_total`              | counter   | status                       |
| `forgebox_engine_llm_calls_total`          | counter   | provider, model              |
| `forgebox_engine_tool_calls_total`         | counter   | tool, result                 |
| `forgebox_engine_tokens_total`             | counter   | direction, provider          |
| `forgebox_gateway_requests_total`          | counter   | method, path, status_code    |
| `forgebox_cost_usd_total`                  | counter   | provider, model, user_id     |

### Structured Logging

All logging uses `log/slog` with JSON output including trace IDs for correlation:

```json
{"time":"2025-01-15T10:30:00Z","level":"INFO","msg":"tool executed",
 "trace_id":"abc123def456","task_id":"task_01","tool":"bash","duration_ms":342}
```

---

## 12. Configuration System

### Layered Loading (highest priority wins)

```
 5. CLI flags           --vm.pool.min-warm=10
 4. Environment vars    FORGEBOX_VM_POOL_MIN_WARM=10
 3. Local file          ./forgebox.yaml
 2. System file         /etc/forgebox/forgebox.yaml
 1. Compiled defaults
```

### Reference Configuration (abridged)

```yaml
server:
  host: 0.0.0.0
  port: 8080
  tls: { enabled: false, cert_file: "", key_file: "" }

auth:
  method: api_key                           # api_key | jwt | oauth2
  api_keys: [{ name: default, key: "${FORGEBOX_API_KEY}" }]

vm:
  firecracker_bin: /usr/local/bin/firecracker
  kernel_path:     /var/lib/forgebox/vmlinux
  rootfs_path:     /var/lib/forgebox/rootfs.ext4
  pool:     { min_warm: 5, max_warm: 20, max_active: 50 }
  defaults: { vcpus: 1, memory_mb: 256, disk_mb: 512, timeout_seconds: 300 }

engine:
  default_provider: anthropic
  default_model:    claude-sonnet-4-20250514
  max_turns: 50
  max_cost_usd: 10.00

providers:
  anthropic: { api_key: "${ANTHROPIC_API_KEY}" }
  openai:    { api_key: "${OPENAI_API_KEY}" }
  ollama:    { base_url: "http://localhost:11434" }

storage:
  backend: sqlite                           # sqlite | postgres
  sqlite:   { path: /var/lib/forgebox/forgebox.db }
  postgres: { dsn: "${DATABASE_URL}" }

telemetry:
  traces:  { enabled: false, exporter: otlp, endpoint: "localhost:4317" }
  metrics: { enabled: true, path: /metrics }
  log_level: info
```

### Secret Interpolation

Any value can reference an environment variable via `${VAR_NAME}`. Resolved at load
time. Missing variables produce a warning; provider `Init()` fails if a required
secret is absent.

---

## 13. Security Model

Defense-in-depth with five layers. A compromise at one layer is contained by those below.

```
 +--------------------------------------------------+
 | L1: Authentication                                |
 |   API keys, JWT, OAuth2; TLS in transit           |
 +--------------------------------------------------+
 | L2: Authorization                                 |
 |   RBAC, org policies, task-scoped permissions     |
 +--------------------------------------------------+
 | L3: VM Isolation                                  |
 |   KVM boundary, separate kernel, no shared fs/net |
 +--------------------------------------------------+
 | L4: In-VM Hardening                               |
 |   seccomp, no capabilities, non-root, R/O rootfs  |
 +--------------------------------------------------+
 | L5: Observability                                 |
 |   Audit log, OTel traces, Prometheus metrics      |
 +--------------------------------------------------+
```

### Threat Model

| Threat                                  | Mitigation                                          |
|-----------------------------------------|-----------------------------------------------------|
| AI generates malicious shell commands   | VM isolation; runs in disposable microVM             |
| AI exfiltrates data via network         | No network by default; domain allowlist              |
| AI attempts privilege escalation in VM  | No root, no capabilities, seccomp, separate kernel   |
| Prompt injection                        | Tool permissions, approval flow for destructive ops  |
| Stolen API key                          | RBAC limits blast radius, audit log for detection    |
| Malicious plugin                        | Sidecars in separate processes with limited access   |
| DoS via VM exhaustion                   | Pool limits, per-task budgets, rate limiting         |

---

## 14. Deployment Patterns

### Single Binary (Development)

Simplest option. One `forgebox` binary, SQLite, local Firecracker.

```bash
# Requirements: Linux + /dev/kvm, firecracker, vmlinux, rootfs.ext4
forgebox serve --config ./forgebox.yaml
```

### Docker

Requires `--privileged` or `/dev/kvm` device passthrough. Configuration in
`deploy/docker/`. The container itself does not add isolation -- Firecracker's KVM
boundary is the real isolation layer. Typically paired with a PostgreSQL container.

### Kubernetes

Requires a device plugin for `/dev/kvm`. Helm chart in `deploy/kubernetes/helm/`.
ForgeBox pods request `github.com/kvm: 1` as a resource. HPA scales on active VM count.
PostgreSQL StatefulSet (or external DB) and S3/MinIO for artifacts.

### Bare Metal + systemd

Unit file in `deploy/systemd/`. Runs as `forgebox` user in `kvm` group with systemd
hardening (`NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths=/var/lib/forgebox`).

---

## 15. Technology Decisions

### Why Go

| Factor             | Go                                   | Alternatives                          |
|--------------------|--------------------------------------|---------------------------------------|
| Single binary      | `go build` -> one static binary      | Rust: same; Python/Node: runtime deps |
| Concurrency        | Goroutines + channels, lightweight   | Rust: async requires runtime choice   |
| Compile speed      | ~2s full build                       | Rust: 30-60s clean                    |
| Standard library   | HTTP server, JSON, crypto, embed     | Node: many npm deps                   |
| Cross-compilation  | `GOOS=linux GOARCH=amd64` trivially  | Rust: workable but harder             |
| Contributor pool   | Large, easy onboarding               | Rust: steeper curve                   |

### Why Firecracker

| Factor              | Firecracker              | gVisor                   | Docker              |
|---------------------|--------------------------|--------------------------|---------------------|
| Isolation           | KVM hardware             | Syscall interception     | Namespaces          |
| Kernel exposure     | None (separate kernel)   | Reduced (Sentry)         | Full host kernel    |
| Boot (snapshot)     | ~5ms                     | ~50ms                    | ~200-500ms          |
| Escape risk         | KVM escape required      | Sentry bugs possible     | Kernel exploits     |
| Proven at scale     | AWS Lambda, Fly.io       | Google Cloud Run         | Everywhere          |

### Why Dual Protocol (gRPC + REST)

| Audience             | Protocol | Reason                                  |
|----------------------|----------|-----------------------------------------|
| CLI clients          | gRPC     | Type-safe, binary encoding, streaming   |
| Web browsers         | REST+SSE | Native browser support, no gRPC-web     |
| Webhooks / channels  | REST     | Universal compatibility                 |
| Host-guest (vsock)   | gRPC     | Efficient, bidirectional streaming      |
| Debugging / curl     | REST     | Human-readable, no special tooling      |

Both served from one port via `cmux`. gRPC is the source of truth; REST handlers are
thin adapters.

### Why SQLite + PostgreSQL

SQLite is the right default for dev/single-node (zero config, embedded). PostgreSQL for
multi-node production. The `StoragePlugin` interface abstracts the difference --
switching is a configuration change.

---

## Appendix: End-to-End Request Lifecycle

```
Client              Gateway           Engine         VM Orch.     fb-agent
  |                    |                 |              |             |
  | POST /v1/tasks     |                 |              |             |
  |------------------->| auth+validate   |              |             |
  | SSE: running       |                 |              |             |
  |<-------------------| engine.Run()    |              |             |
  |                    |---------------->| call LLM     |             |
  | SSE: text_delta    |<- text_delta ---|              |             |
  |<-------------------|                 |              |             |
  | SSE: tool_call     |<- tool_call ----|              |             |
  |<-------------------|                 | perm OK      |             |
  |                    |                 | acquire VM-->|             |
  |                    |                 |<-- ready ----|             |
  |                    |                 | ExecuteTool--+------------>|
  |                    |                 |<-- result ---+-------------|
  | SSE: tool_result   |<- tool_result --|              |             |
  |<-------------------|                 | ... (loop)   |             |
  | SSE: done          |<- done ---------|  release VM->|             |
  |<-------------------|                 |              |             |
```
