# ForgeBox Dashboard: Svelte + Tauri Migration

**Date:** 2026-04-05
**Status:** Draft

## Overview

Replace the existing React admin dashboard with a SvelteKit SPA that serves both a web deployment and a Tauri v2 desktop app from the same codebase. The dashboard has two role-based surfaces: an admin/tech panel (observability, user management, token usage) and a user-facing panel (task execution, workflows, team features).

## Key Decisions

- **Framework:** SvelteKit with `adapter-static` in SPA mode
- **Styling:** SCSS (global partials + scoped component styles), fresh design system, no Tailwind
- **Desktop:** Tauri v2 wrapping the same static SPA output
- **API client:** TypeScript fetch-based client with Svelte stores
- **Monorepo approach:** Single SvelteKit app, Tauri config in `src-tauri/`

## Project Structure

```
web/
├── src/
│   ├── routes/
│   │   ├── +layout.svelte            # Root layout (auth check, role routing)
│   │   ├── +page.svelte              # Redirect based on role
│   │   ├── login/+page.svelte        # Login page
│   │   ├── (admin)/
│   │   │   ├── +layout.svelte        # Admin shell (sidebar, topbar)
│   │   │   ├── dashboard/+page.svelte
│   │   │   ├── users/+page.svelte
│   │   │   ├── token-usage/+page.svelte
│   │   │   ├── observability/+page.svelte
│   │   │   ├── providers/+page.svelte
│   │   │   ├── channels/+page.svelte
│   │   │   ├── vm-settings/+page.svelte
│   │   │   └── audit/+page.svelte
│   │   └── (user)/
│   │       ├── +layout.svelte        # User shell (sidebar, topbar)
│   │       ├── home/+page.svelte
│   │       ├── tasks/
│   │       │   ├── +page.svelte      # My Tasks list
│   │       │   ├── new/+page.svelte  # Task Runner
│   │       │   └── [id]/+page.svelte # Task detail + stream
│   │       ├── workflows/+page.svelte
│   │       ├── team/+page.svelte
│   │       └── settings/+page.svelte
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts             # Fetch-based API client
│   │   │   └── types.ts              # TypeScript types matching Go SDK
│   │   ├── stores/
│   │   │   ├── auth.ts               # Auth state, current user, role
│   │   │   └── tasks.ts              # Task list, active streams
│   │   ├── components/               # Shared Svelte components
│   │   │   ├── Sidebar.svelte
│   │   │   ├── TopBar.svelte
│   │   │   ├── StatusBadge.svelte
│   │   │   ├── ProviderBadge.svelte
│   │   │   ├── EmptyState.svelte
│   │   │   ├── Spinner.svelte
│   │   │   └── TaskStream.svelte
│   │   ├── platform.ts               # Platform detection (web vs Tauri)
│   │   └── styles/
│   │       ├── _variables.scss       # Design tokens
│   │       ├── _mixins.scss          # Reusable SCSS mixins
│   │       ├── _reset.scss           # CSS reset/normalize
│   │       └── global.scss           # Imports partials, base styles
│   └── app.html                      # SvelteKit HTML shell
├── src-tauri/
│   ├── tauri.conf.json               # Tauri v2 config
│   ├── Cargo.toml                    # Rust dependencies
│   ├── src/
│   │   └── main.rs                   # Tauri entry, sidecar management
│   └── sidecar/                      # Optional bundled Go binary
├── static/                           # Static assets (fonts, icons)
├── svelte.config.js                  # SvelteKit config (adapter-static, SPA)
├── vite.config.ts                    # Vite config (SCSS, dev proxy)
├── tsconfig.json
└── package.json
```

## Pages

### Admin Panel (8 pages)

| Route | Purpose |
|---|---|
| `/dashboard` | System health, active VMs, resource usage overview |
| `/users` | Manage users, roles, team assignments |
| `/token-usage` | Usage stats per user/team/provider, cost tracking |
| `/observability` | Logs, task execution traces, error rates |
| `/providers` | Configure LLM providers (Anthropic, OpenAI, etc.) |
| `/channels` | Configure input channels (Slack, webhook, email) |
| `/vm-settings` | Firecracker VM defaults, network policies, resource limits |
| `/audit` | Who did what, when, with what outcome |

### Shared Pages

| Route | Purpose |
|---|---|
| `/login` | Authentication page (visible when not logged in) |

### User Panel (6 pages)

| Route | Purpose |
|---|---|
| `/home` | Personal dashboard, recent tasks, quick actions |
| `/tasks/new` | Submit prompts, pick provider/model, stream output |
| `/tasks` | History of own tasks, status, results, re-run |
| `/workflows` | Create/manage reusable automation workflows |
| `/team` | View team members, shared tasks/workflows |
| `/settings` | Personal preferences, API keys, notifications |

## Routing & Auth

### Roles

- Two roles: `admin` and `user`
- Role comes from the backend auth response (JWT or session)
- Admin can access both `(admin)` and `(user)` routes
- User can only access `(user)` routes

### Auth Flow

1. Root `+layout.svelte` checks auth state on load
2. If not authenticated, redirect to `/login`
3. If authenticated, store user + role in `auth` store
4. Root `+page.svelte` redirects to `/home` (user) or `/dashboard` (admin)
5. `(admin)/+layout.svelte` guards against non-admin access, redirects to `/home`

### Navigation

- Admin sidebar: Dashboard, Users & Teams, Token Usage, Observability, Providers, Channels, VM Settings, Audit Log
- User sidebar: Home, Task Runner, My Tasks, Workflows, Team, Settings
- Admin users see a toggle to switch between admin and user views

### Backend URL Configuration

- Web: `VITE_API_URL` environment variable at build time, defaults to same-origin `/api`
- Tauri: configurable in settings screen, persisted via `tauri-plugin-store`, defaults to `http://localhost:8420/api`

## API Client & Stores

### API Client (`lib/api/client.ts`)

- Fetch-based HTTP client
- Base URL resolved from config (web: env var, Tauri: stored setting)
- Auth token injected via header on every request
- Returns typed responses using `types.ts`
- SSE support for task streaming (EventSource or fetch with ReadableStream)

### Stores

**`auth.ts`:**
- `user`: `{ id, name, email, role }`
- `token`: string
- `isAdmin`: derived boolean
- `login(email, password)`: sets user + token
- `logout()`: clears state, redirects to `/login`

**`tasks.ts`:**
- `tasks`: `Task[]`
- `activeStream`: writable store for current SSE connection
- `fetchTasks()`: GET /tasks
- `createTask(prompt, options)`: POST /tasks
- `streamTask(id)`: SSE /tasks/{id}/stream, updates store reactively
- `cancelTask(id)`: DELETE /tasks/{id}

Additional stores created as needed. Page-local data uses `onMount` fetches or `$effect` when shared state isn't needed.

### Types (`lib/api/types.ts`)

Carried over from current React app:
- Task, Session, Message, ToolCall, ToolResult, AuditEntry, Provider, TaskStatus, TaskEvent

New types added as pages require them:
- User, Team, Workflow, TokenUsage

## Styling Architecture

### SCSS Structure

```
lib/styles/
  _variables.scss    # Design tokens (colors, typography, spacing, radii, shadows, breakpoints)
  _mixins.scss       # Layout helpers, responsive breakpoints, form elements, scrollbars
  _reset.scss        # CSS reset/normalize
  global.scss        # Imports partials, base element styles
```

### Approach

- Fresh design system — no carryover from current Tailwind theme
- Global SCSS for tokens, reset, and base element styles
- Scoped `<style lang="scss">` in Svelte components for component-specific styles
- Components access tokens via `@use '$lib/styles/variables'` and `@use '$lib/styles/mixins'`
- Vite `css.preprocessorOptions.scss.additionalData` auto-imports variables and mixins into all component style blocks
- Semantic class names, no utility classes

### Design Tokens (`_variables.scss`)

- Color palette: primary, neutral, success, warning, error (each with a scale)
- Typography: font families, size scale, weight scale, line heights
- Spacing: consistent increment scale
- Border radii, box shadows, transitions
- Responsive breakpoints

## Tauri Integration

### Two Deployment Modes

**Thin client:**
- Tauri wraps the SPA, no bundled backend
- User configures backend URL in a connection settings screen
- URL persisted via `tauri-plugin-store`

**Thick client (sidecar):**
- Go backend compiled for target OS, placed in `src-tauri/sidecar/`
- Tauri's sidecar feature launches it alongside the app
- `tauri.conf.json` declares the sidecar binary with allowed arguments
- Rust code in `main.rs` manages sidecar lifecycle (start on launch, stop on quit)

### Platform Detection

- Single `lib/platform.ts` utility
- Checks for Tauri via `window.__TAURI__` or `$app/environment`
- Used only for: backend URL resolution, sidecar management, future desktop features
- Platform checks do not leak into components

### Build Outputs

- **Web:** `npm run build` → `build/` directory (static SPA, embeddable in Go binary via `//go:embed`)
- **Tauri:** `npm run tauri build` → native installers (.msi, .dmg, .deb/.AppImage)

## Build & Dev Workflow

### Scripts

| Script | Description |
|---|---|
| `dev` | Vite dev server on port 3000, proxies `/api` to `localhost:8420` |
| `dev:tauri` | Tauri dev window pointing at Vite dev server |
| `build` | `svelte-kit sync && vite build` → static SPA |
| `build:tauri` | `tauri build` → native app with installer |
| `preview` | Serve built SPA locally |
| `lint` | ESLint |
| `typecheck` | `svelte-check` + `tsc` |

### Docker Compose

`docker-compose.dev.yml` dashboard service updated:
- Switches from React/Vite to SvelteKit/Vite
- Same port 3000, same proxy behavior
- No changes to backend or postgres services

### Go Embedding (Production)

- `npm run build` produces `build/` with the static SPA
- `//go:embed` directive in the gateway serves it
- Fallback handler serves `index.html` for all non-API routes (client-side routing)

### Dependencies

**Runtime:** svelte, @sveltejs/kit, @sveltejs/adapter-static, date-fns

**Dev:** vite, sass, typescript, svelte-check, eslint, @tauri-apps/cli

**Tauri plugins:** @tauri-apps/api, @tauri-apps/plugin-store, @tauri-apps/plugin-shell
