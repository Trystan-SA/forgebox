---
name: create-pr-with-review
description: Use when the user asks to create a PR or open a new PR. Runs lint, build, and test locally first. Only opens the PR if all checks pass, then immediately dispatches a review subagent to surface security, code quality, architecture, and domain/business-rule issues, ranked Critical / High / Medium.
---

# Create PR with Review

## Overview

Two-phase PR creation:

1. **Pre-flight**: lint, build, test must all pass locally before the PR is opened.
2. **Post-create**: a review subagent inspects the just-pushed branch and returns a prioritized issue list (Critical / High / Medium).

If pre-flight fails, **do not push, do not open the PR**. Report the failure and stop.

## When to Use

Trigger on user messages like:

- "create a PR"
- "create a new PR"
- "open a PR"
- "PR for that"
- "make a PR"

Do NOT trigger for: amending an existing PR, force-pushing to an existing PR, drafting commits without intent to PR.

## Workflow

### Step 1 — Detect commands

Look for a `Makefile` at the repo root first. If common targets exist (`lint`, `build`, `test`), use them. Otherwise fall back to language-defaults based on what's in the repo:

| Stack signal | Lint | Build | Test |
|---|---|---|---|
| `Makefile` with target | `make lint` | `make build` | `make test` |
| `go.mod` | `golangci-lint run ./...` | `go build ./...` | `go test ./...` |
| `package.json` (web/) | `npm run lint` (if defined) | `npm run build` | `npm test` (if defined) |
| `pyproject.toml` | `ruff check` | (skip) | `pytest` |

Run only the commands that actually exist. If a command is missing for a stack present in the diff, note it explicitly — do not fabricate one.

For monorepos with both backend and frontend changes (e.g. Go + web/), run both sets in parallel.

### Step 2 — Run pre-flight checks

Run lint → build → test. Stop on the first failure.

If any check fails:
- Show which check failed and the relevant error output (truncate to the actionable lines).
- **STOP**. Do not push. Do not open the PR.
- Wait for the user to fix or override.

### Step 3 — Verify branch state

Confirm:
- Current branch is not the default branch (`main` / `master`). If it is, ask the user to create a feature branch first.
- Working tree is clean (`git status --porcelain` empty). If not, list what's uncommitted and ask before continuing.
- Commits exist that aren't on the base branch.

### Step 4 — Create the PR

If all checks pass:
- Push the branch with `-u` if it has no upstream.
- Run `gh pr create --base <default-branch> --title ... --body ...` using the project's commit-message style (check `git log --oneline -5` for reference).
- Capture the PR URL and number.

### Step 5 — Dispatch the review subagent

Immediately after the PR is created, dispatch a subagent. Prefer `pr-code-reviewer` if available; otherwise `general-purpose` or `code-review:code-review`.

Brief the subagent with:

- The PR URL and number.
- The base ref and head ref.
- Instruction to review on **four axes**:
  1. **Security** — auth, input validation, secrets, injection, timing attacks, RBAC bypass.
  2. **Code quality** — bugs, dead code, error handling gaps, race conditions, test coverage of new logic.
  3. **Architecture** — boundary violations, abstraction leaks, coupling, deviations from established patterns in the repo.
  4. **Domain / business rules** — for projects with a `/specs/` directory or equivalent, verify the change matches the spec; flag undocumented behavior changes.
- Output format: findings grouped under `### Critical`, `### High`, `### Medium`. Skip Low. Each finding cites `file:line — <one-line summary> (<axis>)`.

Example dispatch prompt:

> Review PR #N at <URL>. Diff is between `<base>` and `<head>`. Review on four axes: (1) security risks, (2) code quality / correctness, (3) architecture, (4) domain or business-rule violations — if `/specs/` exists, cross-check the change against it. Return findings ranked Critical → High → Medium. Skip Low. Each line: `path:line — summary (axis)`. If no findings at a tier, omit the tier header. Be specific — generic advice is not allowed.

### Step 6 — Present results

Show the user:

```
✅ Pre-flight: lint / build / test passed
✅ PR opened: https://github.com/owner/repo/pull/N

### Critical
- internal/auth/handler.go:88 — token comparison uses == (security: timing attack)

### High
- internal/storage/postgres/storage.go:621 — SQL built with fmt.Sprintf using untrusted filter (security: injection-adjacent)

### Medium
- specs/2.0.0-foo.md — change adds new endpoint not reflected in spec (domain)
```

If the review returns no issues, say so explicitly: `Review returned no Critical/High/Medium findings.`

## Common Mistakes

- **Skipping pre-flight on "small" changes.** Always run the checks. A successful no-op build is still proof.
- **Running review before push.** The review must run against the pushed PR head so it inspects the same code GitHub will show.
- **Silently substituting commands.** If `make lint` doesn't exist, say so — do not invent a replacement.
- **Mixing up base / head.** The review subagent needs both refs to scope the diff correctly.
- **Including Low findings.** Only Critical / High / Medium. Low noise drowns the signal.
- **Forgetting the spec check.** If the project has `/specs/` (e.g. ForgeBox), the domain axis is non-optional.

## Quick Reference

| Phase | Action | Stop on failure? |
|---|---|---|
| 1 | Detect commands | n/a |
| 2 | Run lint / build / test | **Yes** |
| 3 | Verify branch state | **Yes** |
| 4 | Push + `gh pr create` | **Yes** |
| 5 | Dispatch review subagent | No (PR is already open) |
| 6 | Present grouped findings | n/a |
