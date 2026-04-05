# Security Policy

ForgeBox runs AI-generated code inside sandboxed microVMs. We take security
seriously because our users trust us to contain unpredictable workloads.

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 0.x     | Latest minor only  |
| < 0.1   | Not supported      |

Once ForgeBox reaches 1.0, we will support the current major release and the
previous major release with security patches.

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Send your report to **security@forgebox.dev** with the following information:

1. Description of the vulnerability.
2. Steps to reproduce or a proof of concept.
3. Affected component(s) and version(s).
4. Your assessment of severity (critical, high, medium, low).
5. Any suggested fix, if you have one.

You will receive an acknowledgment within **48 hours**. We aim to provide a
substantive response (triage, severity assessment, timeline) within **5 business
days**.

If you do not receive a response within 48 hours, follow up at the same address
or reach out to a maintainer on Discord (see [CONTRIBUTING.md](CONTRIBUTING.md)).

### PGP Key

Our PGP key for encrypted reports is available at:
https://forgebox.dev/.well-known/security-pgp-key.asc

Fingerprint: (will be published when key is generated)

## What Qualifies as a Security Issue

The following are considered security issues and should be reported privately:

- **VM escape:** Any path that allows code running inside a Firecracker microVM
  to access the host system, other VMs, or resources outside its sandbox.
- **Authentication bypass:** Circumventing authentication on the gateway API,
  admin console, or inter-service communication.
- **Authorization / privilege escalation:** Accessing resources, tasks, or
  administrative functions beyond the caller's granted permissions.
- **Data exfiltration:** Bypassing network isolation to send data from a VM to
  an unauthorized external destination.
- **Injection attacks:** Prompt injection, command injection, or SQL injection
  that leads to unintended code execution or data access.
- **Secret leakage:** Exposure of API keys, credentials, or tokens through logs,
  error messages, or API responses.
- **Denial of service:** Resource exhaustion attacks that can take down the
  platform or affect other tenants.
- **Supply chain compromise:** Vulnerabilities in dependencies, build pipeline,
  or release artifacts.

The following are **not** security issues (file as regular bugs):

- Crashes that do not lead to privilege escalation or data exposure.
- Performance issues.
- UI rendering bugs.

## Response Timeline

| Stage                | Target              |
|----------------------|---------------------|
| Acknowledgment       | 48 hours            |
| Triage and severity  | 5 business days     |
| Fix for critical     | 7 days              |
| Fix for high         | 14 days             |
| Fix for medium/low   | Next release cycle  |
| Public disclosure    | After fix is released, coordinated with reporter |

We follow coordinated disclosure. We will credit reporters in the release notes
unless they prefer to remain anonymous.

## Security Design Overview

ForgeBox's security architecture is built on multiple defense layers:

1. **Firecracker microVM isolation** backed by KVM hardware virtualization.
2. **In-VM hardening** with seccomp-bpf, dropped capabilities, and non-root execution.
3. **Network isolation** with no internet access by default.
4. **Permission model** with explicit, least-privilege grants per task.
5. **Audit logging** of all administrative and security-relevant actions.

For the full security model, including threat model, defense layers, and supply
chain protections, see [docs/security-model.md](docs/security-model.md).

## Bug Bounty

We are planning a formal bug bounty program. Until the program launches, we will
recognize and credit all valid security reports in our release notes and security
advisories. Reporters of critical vulnerabilities will be acknowledged in our
Hall of Fame at https://forgebox.dev/security/thanks.

## Security Advisories

Published advisories are available at:
https://github.com/forgebox-dev/forgebox/security/advisories

Subscribe to the repository's security alerts to be notified of new advisories.
