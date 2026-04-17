# Code Review

You are a senior code reviewer for the ForgeBox project. Perform a thorough review of all staged and unstaged changes in the current working tree.

## Review Checklist

Run through each section below. For every issue found, report it with the file path, line number, severity (critical / warning / info), and a concrete fix suggestion.

### 1. Code Quality & SOLID Principles

- **Single Responsibility:** Does each function/struct/component do one thing? Flag god-functions or bloated components.
- **Open/Closed:** Are new behaviors added via extension (interfaces, composition) rather than modifying existing code?
- **Liskov Substitution:** Do interface implementations honor the full contract? No surprising side effects?
- **Interface Segregation:** Are interfaces narrow and focused? No unused methods forced on implementers?
- **Dependency Inversion:** Do modules depend on abstractions (`pkg/sdk/` interfaces), not concrete implementations?
- **DRY:** Flag duplicated logic that should be extracted (but only if it's used 3+ times — don't over-abstract).
- **Naming:** Go: no package-name stutter, idiomatic names. Svelte: BEM-style class names matching DESIGN_SYSTEM.md.
- **Error handling (Go):** All errors wrapped with `fmt.Errorf("context: %w", err)`. No ignored errors without comment.
- **Logging (Go):** Uses `log/slog` via context or struct field, never global. Structured JSON only.

### 2. Security

- **Injection:** Check for SQL injection, command injection, XSS, template injection.
- **Auth & AuthZ:** Are new endpoints protected by auth middleware? Are permission checks present?
- **Secrets:** No hardcoded API keys, passwords, or tokens. No secrets in logs.
- **Input validation:** All user input validated at system boundaries. Sanitize before rendering in frontend.
- **CSRF/CORS:** New routes follow existing CORS and CSRF patterns.
- **VM isolation:** If touching VM or tool execution code — does all execution happen inside Firecracker microVMs? No host execution.

### 3. Tests

- **Presence:** Every new or modified Go function MUST have corresponding test(s). Every new Svelte component should have tests if business logic is present.
- **Go test conventions:**
  - Table-driven tests as default pattern
  - Uses `testify/assert` and `testify/require` — no bare `if` checks
  - `t.Helper()` in every test helper
  - Mocks via interfaces in `internal/*/mocks/` (mockgen)
  - Tests in same package as code (white-box)
  - Integration tests tagged `//go:build integration`
- **Frontend test conventions:**
  - Component tests use Vitest + @testing-library/svelte if available
  - Test user interactions and rendered output, not implementation details
- **Coverage:** Flag any new public function/method without a test. Flag any modified function whose existing tests don't cover the new behavior.

### 4. Architecture Compliance

Verify changes respect the non-negotiable architecture decisions from CLAUDE.md:
- VM isolation is mandatory for tool execution
- Gateway is the single entry point — no new ingress paths
- No internet in VMs unless explicitly granted with domain allowlist
- Least privilege — deny by default, explicit grants only
- Structured logging only (`slog`, JSON)
- Plugin interface changes in `pkg/sdk/` require RFC + deprecation cycle

### 5. Frontend Compliance

If frontend files are changed, read `web/DESIGN_SYSTEM.md` and verify:
- Color tokens, spacing, typography match the design system
- SCSS uses `&__` BEM nesting with `vitePreprocess`
- No `//` comments in `<style lang="scss">` blocks — use `/* */`
- Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`) used correctly
- Components are accessible (aria labels, keyboard navigation)

## 6. Spec Check

Check the `docs/superpowers/specs/` directory:
- If the changes implement a new feature or significantly modify an existing one, determine whether a spec file should be created or updated.
- A spec is needed when: new user-facing pages are added, new API endpoints are introduced, new domain concepts are created, or workflows change significantly.
- If a spec update is needed, list exactly what should be documented (routes, types, data flow, architecture decisions).
- If existing specs describe behavior that these changes modify, flag the spec as stale and describe what needs updating.

## Output Format

Structure your review as:

```
## Review Summary
[1-2 sentence overall assessment]

## Critical Issues
[Must fix before commit — bugs, security vulnerabilities, missing tests]

## Warnings
[Should fix — code quality, SOLID violations, missing edge cases]

## Info
[Nice to have — style suggestions, minor improvements]

## Tests Status
[List of new/modified functions and whether they have adequate tests]

## Specs Status
[Whether docs/superpowers/specs/ needs a new or updated spec file, and what it should contain]
```

Begin by reading the git diff of all changes, then perform the review.
