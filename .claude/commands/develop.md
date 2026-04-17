# Feature Development (DDD/TDD)

You are a senior software engineer building features for ForgeBox following Domain-Driven Design and Test-Driven Development principles. This skill guides complete feature implementation across the full stack (Go backend + SvelteKit frontend).

Your argument is: $ARGUMENTS

---

## Phase 1: Domain Discovery

Before writing any code, understand the domain:

1. **Identify the Bounded Context.** Which part of the system does this feature belong to? (e.g., `auth`, `engine`, `gateway`, `vm`, `plugins`, `agents`, `automations`)
2. **Define the Ubiquitous Language.** List the domain terms this feature introduces or uses. These exact terms must appear in code (struct names, function names, route names, API endpoints). No translation — the code IS the domain language.
3. **Map the Aggregates & Entities.** What are the core domain objects? What are their invariants (rules that must always be true)?
4. **Identify Domain Events.** What happens when this feature executes? (e.g., `TaskCreated`, `AgentAssigned`, `WorkflowCompleted`)
5. **Define the Value Objects.** Immutable types that describe characteristics (e.g., `Permission`, `TokenUsage`, `VMConfig`).

Write a brief domain model summary before proceeding. Get user confirmation.

## Phase 2: Spec First

Before implementation, create or update a spec in `docs/superpowers/specs/`:

1. **Create the spec file** named `YYYY-MM-DD-feature-name.md` (use today's date).
2. **Spec structure:**
   - Overview: what and why
   - Domain model: entities, value objects, aggregates, events
   - API design: endpoints, request/response types
   - Frontend: routes, components, user flows
   - Architecture decisions: tradeoffs made and why
   - Test strategy: what will be tested and how
3. Get user confirmation on the spec before writing code.

## Phase 3: Backend Development (Go) — TDD

Follow strict Red-Green-Refactor for every piece of backend logic:

### 3.1 Domain Layer (`internal/<context>/`)

Build from the inside out — domain first, then infrastructure.

1. **Define domain types** — structs, interfaces, enums in a dedicated file (e.g., `internal/agents/agent.go`).
   - Use named fields, no positional initialization
   - No package-name stutter (`agents.Agent`, not `agents.AgentsAgent`)
   - Value objects are immutable (no pointer receivers that mutate)

2. **Write domain logic tests FIRST** (`internal/<context>/<name>_test.go`):
   - Table-driven tests with `testify/assert` and `testify/require`
   - Test the domain rules (invariants, validations, state transitions)
   - `t.Helper()` in all test helpers
   - Tests in same package (white-box)

   ```go
   func TestAgent_Validate(t *testing.T) {
       tests := []struct {
           name    string
           agent   Agent
           wantErr string
       }{
           {name: "valid agent", agent: Agent{Name: "test"}, wantErr: ""},
           {name: "empty name", agent: Agent{}, wantErr: "name is required"},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := tt.agent.Validate()
               if tt.wantErr == "" {
                   require.NoError(t, err)
               } else {
                   require.ErrorContains(t, err, tt.wantErr)
               }
           })
       }
   }
   ```

3. **Run the test — it must FAIL (Red).**
4. **Write the minimum code to pass (Green).**
5. **Refactor** while keeping tests green.

### 3.2 Repository / Storage Layer

1. **Define the repository interface** in the domain package:
   ```go
   type AgentRepository interface {
       Create(ctx context.Context, agent *Agent) error
       Get(ctx context.Context, id string) (*Agent, error)
       List(ctx context.Context) ([]*Agent, error)
       Delete(ctx context.Context, id string) error
   }
   ```

2. **Write repository tests first** with a real SQLite database (no mocks for storage tests — per project convention, integration tests hit real DBs).
3. **Implement the repository** in `internal/storage/`.

### 3.3 Service / Application Layer

1. **Define the service** that orchestrates domain logic + repository:
   ```go
   type AgentService struct {
       repo   AgentRepository
       logger *slog.Logger
   }
   ```
2. **Write service tests first** — mock the repository interface.
3. **Implement the service** — this is where cross-cutting concerns live (logging, audit, permissions).

### 3.4 HTTP Handler Layer (`internal/gateway/`)

1. **Write handler tests first** — mock the service interface.
2. **Implement the handler** — thin layer, validates input, calls service, returns JSON.
3. **Register routes** in the gateway router.
4. **Error handling:** Always wrap with `fmt.Errorf("context: %w", err)`.
5. **Logging:** `slog` via context, structured JSON.
6. **Auth:** Every new endpoint must be behind auth middleware.

### 3.5 Integration Tests

After unit tests pass:
1. Write integration tests tagged `//go:build integration` in `test/` or alongside the code.
2. These tests hit the real HTTP server + database.
3. Test the full request lifecycle: HTTP request -> handler -> service -> repo -> DB -> response.

## Phase 4: Frontend Development (SvelteKit) — TDD

### 4.1 API Types & Client

1. **Add TypeScript types** in `web/src/lib/api/types.ts` mirroring the Go domain types exactly (same field names, snake_case).
2. **Add API client functions** in `web/src/lib/api/client.ts` — one function per endpoint.
3. Write tests for any complex client-side logic (transformations, validations).

### 4.2 Components & Pages

Follow the design system in `web/DESIGN_SYSTEM.md`:

1. **Read `web/DESIGN_SYSTEM.md`** before creating any component. Follow its color palette, typography, spacing, component patterns, and SCSS architecture.
2. **Svelte 5 runes** — use `$state`, `$derived`, `$effect`, `$props` (not legacy stores in components).
3. **SCSS conventions:**
   - Scoped `<style lang="scss">` in each component
   - BEM-style with `&__` nesting
   - Use design tokens from `_variables.scss` (auto-imported by Vite)
   - NO `//` comments in SCSS — use `/* */`
4. **Accessibility:** aria labels, keyboard navigation, focus management.
5. **Component structure:** Props interface -> reactive state -> derived values -> effects -> event handlers -> template -> scoped styles.

### 4.3 Routes

1. **Page files** in `web/src/routes/(app)/<feature>/+page.svelte`.
2. **Loading states:** Show spinners during async operations.
3. **Error states:** Display user-friendly error messages.
4. **Empty states:** Use `EmptyState` component when no data.

## Phase 5: Commit & Review

When the feature is complete and you are asked to commit:

**IMPORTANT: Before committing, launch a sub-agent to run the `/review` skill.**

The review agent will:
- Check code quality and SOLID principles
- Verify security posture
- Confirm tests are present and adequate
- Check if `docs/superpowers/specs/` needs a new or updated spec

**Do not commit until the review passes.** If the review finds critical issues, fix them first.

After the review passes:
1. Stage the relevant files (be specific, don't `git add -A`).
2. Write a conventional commit message: `feat(<scope>): <description>` or `fix(<scope>): <description>`.
3. Commit.

## DDD/TDD Quick Reference

### Domain-Driven Design Layers (inside-out)

```
Domain Layer     → Entities, Value Objects, Aggregates, Domain Events, Repository interfaces
Application Layer → Services (orchestrate domain + infra), DTOs
Infrastructure   → Repository implementations, HTTP handlers, external service clients
Presentation     → SvelteKit routes, components, API client
```

### TDD Cycle

```
1. RED    → Write a failing test that defines the desired behavior
2. GREEN  → Write the minimum code to make the test pass
3. REFACTOR → Improve the code while keeping tests green
```

### Rules

- Never write production code without a failing test first
- Each test should test one behavior
- Domain logic has zero infrastructure dependencies
- Repository interfaces live in the domain package, implementations in infrastructure
- Services depend on interfaces, never concrete implementations
- Frontend types mirror backend types exactly
- Every public function needs a test