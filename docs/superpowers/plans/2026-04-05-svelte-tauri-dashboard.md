# SvelteKit + Tauri Dashboard Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the React admin dashboard with a SvelteKit SPA that serves both web and Tauri v2 desktop from the same codebase, with role-based admin/user panels.

**Architecture:** SvelteKit with `adapter-static` in SPA mode. SCSS design system (no Tailwind). Fetch-based API client with Svelte writable stores. Tauri v2 wraps the same static output. Two layout groups: `(admin)` and `(user)` with role-based guards.

**Tech Stack:** SvelteKit 2, Svelte 5, TypeScript, SCSS, Vite, Tauri v2, date-fns

---

## File Structure

```
web/
├── src/
│   ├── routes/
│   │   ├── +layout.svelte            # Root layout (auth check, role routing)
│   │   ├── +layout.ts                # Root load (SPA disable SSR)
│   │   ├── +page.svelte              # Redirect based on role
│   │   ├── login/+page.svelte        # Login page
│   │   ├── (admin)/
│   │   │   ├── +layout.svelte        # Admin shell (sidebar, topbar, guard)
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
│   │   ├── components/
│   │   │   ├── Sidebar.svelte
│   │   │   ├── TopBar.svelte
│   │   │   ├── StatusBadge.svelte
│   │   │   ├── ProviderBadge.svelte
│   │   │   ├── EmptyState.svelte
│   │   │   ├── Spinner.svelte
│   │   │   └── TaskStream.svelte
│   │   ├── platform.ts               # Platform detection (web vs Tauri)
│   │   └── styles/
│   │       ├── _variables.scss
│   │       ├── _mixins.scss
│   │       ├── _reset.scss
│   │       └── global.scss
│   └── app.html                      # SvelteKit HTML shell
├── src-tauri/
│   ├── tauri.conf.json
│   ├── Cargo.toml
│   └── src/
│       └── main.rs
├── static/                           # Static assets
├── svelte.config.js
├── vite.config.ts
├── tsconfig.json
└── package.json
```

---

### Task 1: Scaffold SvelteKit Project

Remove React dependencies and scaffold SvelteKit with adapter-static in SPA mode.

**Files:**
- Rewrite: `web/package.json`
- Rewrite: `web/vite.config.ts`
- Rewrite: `web/tsconfig.json`
- Create: `web/svelte.config.js`
- Rewrite: `web/src/app.html`
- Delete: `web/postcss.config.js`, `web/tailwind.config.ts`, `web/index.html`
- Delete: All `web/src/*.tsx`, `web/src/components/*.tsx`, `web/src/pages/*.tsx`, `web/src/hooks/*.ts`, `web/src/index.css`

- [ ] **Step 1: Remove old React source files**

```bash
cd web
rm -rf src/App.tsx src/main.tsx src/index.css src/components src/pages src/hooks
rm -f postcss.config.js tailwind.config.ts index.html
```

- [ ] **Step 2: Write `package.json`**

```json
{
  "name": "@forgebox/dashboard",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite dev",
    "dev:tauri": "tauri dev",
    "build": "svelte-kit sync && vite build",
    "build:tauri": "tauri build",
    "preview": "vite preview",
    "lint": "eslint .",
    "typecheck": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json"
  },
  "dependencies": {
    "date-fns": "^4.1.0"
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.0",
    "@sveltejs/kit": "^2.0.0",
    "@sveltejs/vite-plugin-svelte": "^4.0.0",
    "sass": "^1.80.0",
    "svelte": "^5.0.0",
    "svelte-check": "^4.0.0",
    "typescript": "^5.7.0",
    "vite": "^6.0.0"
  }
}
```

- [ ] **Step 3: Write `svelte.config.js`**

```js
import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			fallback: 'index.html'
		}),
		alias: {
			'$lib': './src/lib'
		}
	}
};

export default config;
```

- [ ] **Step 4: Write `vite.config.ts`**

```ts
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		port: 3000,
		proxy: {
			'/api': {
				target: process.env.VITE_API_URL || 'http://localhost:8420',
				changeOrigin: true
			}
		}
	},
	css: {
		preprocessorOptions: {
			scss: {
				additionalData: `
					@use '$lib/styles/variables' as *;
					@use '$lib/styles/mixins' as *;
				`
			}
		}
	}
});
```

- [ ] **Step 5: Write `tsconfig.json`**

```json
{
	"extends": "./.svelte-kit/tsconfig.json",
	"compilerOptions": {
		"allowJs": true,
		"checkJs": true,
		"esModuleInterop": true,
		"forceConsistentCasingInFileNames": true,
		"resolveJsonModule": true,
		"skipLibCheck": true,
		"sourceMap": true,
		"strict": true,
		"moduleResolution": "bundler"
	}
}
```

- [ ] **Step 6: Write `src/app.html`**

```html
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<link rel="icon" href="%sveltekit.assets%/favicon.png" />
		<title>ForgeBox</title>
		%sveltekit.head%
	</head>
	<body>
		<div id="app">%sveltekit.body%</div>
	</body>
</html>
```

- [ ] **Step 7: Install dependencies and verify**

```bash
cd web && rm -rf node_modules package-lock.json && npm install
```

Expected: Clean install with no errors.

- [ ] **Step 8: Commit**

```bash
git add -A web/
git commit -m "feat(web): scaffold SvelteKit with adapter-static, replace React"
```

---

### Task 2: SCSS Design System

Create the fresh design system with variables, mixins, reset, and global styles.

**Files:**
- Create: `web/src/lib/styles/_variables.scss`
- Create: `web/src/lib/styles/_mixins.scss`
- Create: `web/src/lib/styles/_reset.scss`
- Create: `web/src/lib/styles/global.scss`

- [ ] **Step 1: Write `_variables.scss`**

```scss
// --- Colors ---
$primary-50: #eef2ff;
$primary-100: #e0e7ff;
$primary-200: #c7d2fe;
$primary-300: #a5b4fc;
$primary-400: #818cf8;
$primary-500: #6366f1;
$primary-600: #4f46e5;
$primary-700: #4338ca;
$primary-800: #3730a3;
$primary-900: #312e81;

$neutral-0: #ffffff;
$neutral-50: #f9fafb;
$neutral-100: #f3f4f6;
$neutral-200: #e5e7eb;
$neutral-300: #d1d5db;
$neutral-400: #9ca3af;
$neutral-500: #6b7280;
$neutral-600: #4b5563;
$neutral-700: #374151;
$neutral-800: #1f2937;
$neutral-900: #111827;

$success-50: #ecfdf5;
$success-100: #d1fae5;
$success-500: #10b981;
$success-600: #059669;
$success-700: #047857;

$warning-50: #fffbeb;
$warning-100: #fef3c7;
$warning-500: #f59e0b;
$warning-600: #d97706;
$warning-700: #b45309;

$error-50: #fef2f2;
$error-100: #fee2e2;
$error-500: #ef4444;
$error-600: #dc2626;
$error-700: #b91c1c;

$info-50: #eff6ff;
$info-100: #dbeafe;
$info-500: #3b82f6;
$info-600: #2563eb;

// --- Typography ---
$font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
$font-mono: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;

$text-xs: 0.75rem;
$text-sm: 0.875rem;
$text-base: 1rem;
$text-lg: 1.125rem;
$text-xl: 1.25rem;
$text-2xl: 1.5rem;
$text-3xl: 1.875rem;

$font-normal: 400;
$font-medium: 500;
$font-semibold: 600;
$font-bold: 700;

$leading-tight: 1.25;
$leading-normal: 1.5;
$leading-relaxed: 1.625;

// --- Spacing ---
$space-1: 0.25rem;
$space-2: 0.5rem;
$space-3: 0.75rem;
$space-4: 1rem;
$space-5: 1.25rem;
$space-6: 1.5rem;
$space-8: 2rem;
$space-10: 2.5rem;
$space-12: 3rem;
$space-16: 4rem;

// --- Border Radius ---
$radius-sm: 0.25rem;
$radius-md: 0.375rem;
$radius-lg: 0.5rem;
$radius-xl: 0.75rem;
$radius-2xl: 1rem;
$radius-full: 9999px;

// --- Shadows ---
$shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
$shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1);
$shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1);

// --- Transitions ---
$transition-fast: 150ms ease;
$transition-base: 200ms ease;
$transition-slow: 300ms ease;

// --- Breakpoints ---
$bp-sm: 640px;
$bp-md: 768px;
$bp-lg: 1024px;
$bp-xl: 1280px;

// --- Layout ---
$sidebar-width: 16rem;
$topbar-height: 3.5rem;
```

- [ ] **Step 2: Write `_mixins.scss`**

```scss
// --- Responsive ---
@mixin sm { @media (min-width: $bp-sm) { @content; } }
@mixin md { @media (min-width: $bp-md) { @content; } }
@mixin lg { @media (min-width: $bp-lg) { @content; } }
@mixin xl { @media (min-width: $bp-xl) { @content; } }

// --- Layout ---
@mixin flex-center {
	display: flex;
	align-items: center;
	justify-content: center;
}

@mixin flex-between {
	display: flex;
	align-items: center;
	justify-content: space-between;
}

// --- Components ---
@mixin card {
	background: $neutral-0;
	border: 1px solid $neutral-200;
	border-radius: $radius-xl;
	box-shadow: $shadow-sm;
}

@mixin input-base {
	display: block;
	width: 100%;
	padding: $space-2 $space-3;
	font-size: $text-sm;
	line-height: $leading-normal;
	color: $neutral-800;
	background: $neutral-0;
	border: 1px solid $neutral-300;
	border-radius: $radius-lg;
	box-shadow: $shadow-sm;
	transition: border-color $transition-fast, box-shadow $transition-fast;

	&::placeholder {
		color: $neutral-400;
	}

	&:focus {
		outline: none;
		border-color: $primary-500;
		box-shadow: 0 0 0 1px $primary-500;
	}

	&:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}
}

@mixin btn {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	padding: $space-2 $space-4;
	font-size: $text-sm;
	font-weight: $font-medium;
	border-radius: $radius-lg;
	transition: all $transition-fast;
	cursor: pointer;
	border: none;

	&:focus-visible {
		outline: 2px solid $primary-500;
		outline-offset: 2px;
	}

	&:disabled {
		opacity: 0.5;
		pointer-events: none;
	}
}

@mixin badge {
	display: inline-flex;
	align-items: center;
	gap: $space-1;
	padding: $space-1 $space-2;
	font-size: $text-xs;
	font-weight: $font-medium;
	border-radius: $radius-full;
}

// --- Scrollbar ---
@mixin scrollbar-thin {
	scrollbar-width: thin;
	scrollbar-color: $neutral-300 transparent;

	&::-webkit-scrollbar {
		width: 6px;
	}
	&::-webkit-scrollbar-track {
		background: transparent;
	}
	&::-webkit-scrollbar-thumb {
		background: $neutral-300;
		border-radius: 3px;
	}
}

// --- Truncate ---
@mixin truncate {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}
```

- [ ] **Step 3: Write `_reset.scss`**

```scss
*,
*::before,
*::after {
	box-sizing: border-box;
	margin: 0;
	padding: 0;
}

html {
	-webkit-text-size-adjust: 100%;
	tab-size: 4;
}

body {
	font-family: $font-sans;
	font-size: $text-base;
	line-height: $leading-normal;
	color: $neutral-800;
	background: $neutral-50;
	-webkit-font-smoothing: antialiased;
	-moz-osx-font-smoothing: grayscale;
}

img, svg {
	display: block;
	max-width: 100%;
}

button, input, select, textarea {
	font: inherit;
	color: inherit;
}

a {
	color: inherit;
	text-decoration: none;
}

table {
	border-collapse: collapse;
	width: 100%;
}

h1, h2, h3, h4 {
	line-height: $leading-tight;
}
```

- [ ] **Step 4: Write `global.scss`**

```scss
@use 'variables' as *;
@use 'reset';

// --- Base element styles ---
h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
h2 { font-size: $text-lg; font-weight: $font-semibold; color: $neutral-900; }

// --- Utility classes (minimal set) ---
.btn-primary {
	@include btn;
	background: $primary-600;
	color: $neutral-0;

	&:hover { background: $primary-700; }
}

.btn-secondary {
	@include btn;
	background: $neutral-0;
	color: $neutral-700;
	border: 1px solid $neutral-300;

	&:hover { background: $neutral-50; }
}

.btn-danger {
	@include btn;
	background: $error-600;
	color: $neutral-0;

	&:hover { background: $error-700; }
}

.btn-ghost {
	@include btn;
	background: transparent;
	color: $neutral-600;

	&:hover { background: $neutral-100; }
}
```

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/styles/
git commit -m "feat(web): add SCSS design system with tokens, mixins, reset"
```

---

### Task 3: API Client & Types

Port the TypeScript API client and types from the React app to `$lib/api/`.

**Files:**
- Create: `web/src/lib/api/types.ts`
- Create: `web/src/lib/api/client.ts`

- [ ] **Step 1: Write `types.ts`**

Port existing types and add new ones for the expanded dashboard:

```ts
// Types matching the ForgeBox Go SDK (pkg/sdk/)

export type TaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export type UserRole = 'admin' | 'user';

export interface User {
	id: string;
	name: string;
	email: string;
	role: UserRole;
	team_id?: string;
	created_at: string;
}

export interface Task {
	id: string;
	status: TaskStatus;
	prompt: string;
	result?: string;
	provider: string;
	model: string;
	user_id: string;
	session_id: string;
	cost: number;
	tokens_in: number;
	tokens_out: number;
	error?: string;
	created_at: string;
	started_at?: string;
	completed_at?: string;
}

export interface Session {
	id: string;
	user_id: string;
	provider: string;
	model: string;
	created_at: string;
	updated_at: string;
}

export interface Message {
	role: 'user' | 'assistant' | 'system';
	content?: string;
	tool_calls?: ToolCall[];
	tool_results?: ToolResult[];
}

export interface ToolCall {
	id: string;
	name: string;
	input: string;
}

export interface ToolResult {
	tool_call_id: string;
	content: string;
	is_error: boolean;
}

export interface Provider {
	name: string;
	version: string;
	type: 'provider';
	builtin: boolean;
}

export interface ToolSchema {
	name: string;
	description: string;
	input_schema?: Record<string, unknown>;
}

export interface AuditEntry {
	id: string;
	timestamp: string;
	user_id: string;
	task_id: string;
	action: string;
	tool?: string;
	decision: 'allow' | 'deny';
	reason?: string;
}

export interface CreateTaskRequest {
	prompt: string;
	provider?: string;
	model?: string;
	timeout?: string;
	memory_mb?: number;
	vcpus?: number;
	network_access?: boolean;
}

export type TaskEventType =
	| 'connected'
	| 'status_update'
	| 'text_delta'
	| 'tool_call'
	| 'tool_result'
	| 'error'
	| 'done';

export interface TaskEvent {
	type: TaskEventType;
	text?: string;
	tool_call?: ToolCall;
	result?: ToolResult;
	error?: string;
	status?: TaskStatus;
}

export interface VMPoolStatus {
	pool_size: number;
	active_count: number;
}

export interface Team {
	id: string;
	name: string;
	members: string[];
	created_at: string;
}

export interface Workflow {
	id: string;
	name: string;
	description: string;
	prompt_template: string;
	provider?: string;
	model?: string;
	created_by: string;
	created_at: string;
	updated_at: string;
}

export interface TokenUsage {
	user_id: string;
	provider: string;
	model: string;
	tokens_in: number;
	tokens_out: number;
	cost: number;
	period: string;
}

export interface LoginRequest {
	email: string;
	password: string;
}

export interface LoginResponse {
	token: string;
	user: User;
}
```

- [ ] **Step 2: Write `client.ts`**

```ts
import type {
	Task,
	Session,
	Provider,
	ToolSchema,
	AuditEntry,
	CreateTaskRequest,
	TaskEvent,
	LoginRequest,
	LoginResponse
} from './types';
import { getBaseUrl } from '$lib/platform';

function getToken(): string | null {
	if (typeof window === 'undefined') return null;
	return localStorage.getItem('forgebox_token');
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const base = getBaseUrl();
	const token = getToken();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...(token ? { Authorization: `Bearer ${token}` } : {})
	};

	const res = await fetch(`${base}${path}`, {
		headers,
		...init
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error || `HTTP ${res.status}`);
	}

	return res.json();
}

// --- Auth ---
export async function login(req: LoginRequest): Promise<LoginResponse> {
	return request('/auth/login', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

// --- Tasks ---
export async function createTask(
	req: CreateTaskRequest
): Promise<{ task_id: string; status: string }> {
	return request('/tasks', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function getTask(id: string): Promise<Task> {
	return request(`/tasks/${id}`);
}

export async function listTasks(): Promise<Task[]> {
	return (await request<Task[] | null>('/tasks')) ?? [];
}

export async function cancelTask(
	id: string
): Promise<{ task_id: string; status: string }> {
	return request(`/tasks/${id}`, { method: 'DELETE' });
}

export function streamTask(
	id: string,
	onEvent: (event: TaskEvent) => void,
	onError?: (error: Error) => void
): () => void {
	const base = getBaseUrl();
	const source = new EventSource(`${base}/tasks/${id}/stream`);

	source.onmessage = (e) => {
		try {
			const event: TaskEvent = JSON.parse(e.data);
			onEvent(event);
		} catch {
			// Ignore parse errors for heartbeat/keepalive messages.
		}
	};

	source.onerror = () => {
		onError?.(new Error('SSE connection lost'));
		source.close();
	};

	return () => source.close();
}

// --- Sessions ---
export async function listSessions(): Promise<Session[]> {
	return (await request<Session[] | null>('/sessions')) ?? [];
}

export async function getSession(id: string): Promise<Session> {
	return request(`/sessions/${id}`);
}

export async function sendMessage(
	sessionId: string,
	text: string
): Promise<{ status: string }> {
	return request(`/sessions/${sessionId}/message`, {
		method: 'POST',
		body: JSON.stringify({ text })
	});
}

// --- Discovery ---
export async function listProviders(): Promise<Provider[]> {
	return (await request<Provider[] | null>('/providers')) ?? [];
}

export async function listTools(): Promise<ToolSchema[]> {
	return (await request<ToolSchema[] | null>('/tools')) ?? [];
}

// --- Audit ---
export async function listAuditEntries(): Promise<AuditEntry[]> {
	return (await request<AuditEntry[] | null>('/audit')) ?? [];
}
```

- [ ] **Step 3: Commit**

```bash
git add web/src/lib/api/
git commit -m "feat(web): add TypeScript API client and types for SvelteKit"
```

---

### Task 4: Platform Detection & Svelte Stores

Create platform detection utility and Svelte stores for auth and tasks.

**Files:**
- Create: `web/src/lib/platform.ts`
- Create: `web/src/lib/stores/auth.ts`
- Create: `web/src/lib/stores/tasks.ts`

- [ ] **Step 1: Write `platform.ts`**

```ts
export function isTauri(): boolean {
	return typeof window !== 'undefined' && '__TAURI__' in window;
}

export function getBaseUrl(): string {
	if (isTauri()) {
		const stored = localStorage.getItem('forgebox_api_url');
		return stored || 'http://localhost:8420/api/v1';
	}
	return '/api/v1';
}
```

- [ ] **Step 2: Write `stores/auth.ts`**

```ts
import { writable, derived } from 'svelte/store';
import type { User, UserRole } from '$lib/api/types';
import { login as apiLogin } from '$lib/api/client';

interface AuthState {
	user: User | null;
	token: string | null;
}

function createAuthStore() {
	const initial: AuthState = {
		user: null,
		token: typeof window !== 'undefined' ? localStorage.getItem('forgebox_token') : null
	};

	// Try to restore user from localStorage
	if (typeof window !== 'undefined') {
		const stored = localStorage.getItem('forgebox_user');
		if (stored) {
			try {
				initial.user = JSON.parse(stored);
			} catch {
				// ignore
			}
		}
	}

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,
		async login(email: string, password: string) {
			const res = await apiLogin({ email, password });
			localStorage.setItem('forgebox_token', res.token);
			localStorage.setItem('forgebox_user', JSON.stringify(res.user));
			set({ user: res.user, token: res.token });
		},
		logout() {
			localStorage.removeItem('forgebox_token');
			localStorage.removeItem('forgebox_user');
			set({ user: null, token: null });
		},
		setUser(user: User, token: string) {
			localStorage.setItem('forgebox_token', token);
			localStorage.setItem('forgebox_user', JSON.stringify(user));
			set({ user, token });
		}
	};
}

export const auth = createAuthStore();
export const currentUser = derived(auth, ($auth) => $auth.user);
export const isAuthenticated = derived(auth, ($auth) => !!$auth.token && !!$auth.user);
export const isAdmin = derived(auth, ($auth) => $auth.user?.role === 'admin');
export const userRole = derived(auth, ($auth): UserRole | null => $auth.user?.role ?? null);
```

- [ ] **Step 3: Write `stores/tasks.ts`**

```ts
import { writable, derived } from 'svelte/store';
import type { Task, TaskEvent } from '$lib/api/types';
import {
	listTasks as apiListTasks,
	createTask as apiCreateTask,
	cancelTask as apiCancelTask,
	streamTask as apiStreamTask
} from '$lib/api/client';
import type { CreateTaskRequest } from '$lib/api/types';

export const tasks = writable<Task[]>([]);
export const tasksLoading = writable(false);
export const tasksError = writable<string | null>(null);

export async function fetchTasks() {
	tasksLoading.set(true);
	tasksError.set(null);
	try {
		const result = await apiListTasks();
		tasks.set(result);
	} catch (err) {
		tasksError.set(err instanceof Error ? err.message : 'Failed to fetch tasks');
	} finally {
		tasksLoading.set(false);
	}
}

export async function submitTask(
	req: CreateTaskRequest,
	onEvent: (event: TaskEvent) => void,
	onError?: (error: Error) => void
): Promise<{ taskId: string; stop: () => void }> {
	const res = await apiCreateTask(req);
	const stop = apiStreamTask(res.task_id, onEvent, onError);
	return { taskId: res.task_id, stop };
}

export async function cancelRunningTask(id: string): Promise<void> {
	await apiCancelTask(id);
}
```

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/platform.ts web/src/lib/stores/
git commit -m "feat(web): add platform detection, auth store, tasks store"
```

---

### Task 5: Shared Components

Create all reusable Svelte components.

**Files:**
- Create: `web/src/lib/components/Spinner.svelte`
- Create: `web/src/lib/components/StatusBadge.svelte`
- Create: `web/src/lib/components/ProviderBadge.svelte`
- Create: `web/src/lib/components/EmptyState.svelte`
- Create: `web/src/lib/components/TopBar.svelte`
- Create: `web/src/lib/components/Sidebar.svelte`
- Create: `web/src/lib/components/TaskStream.svelte`

- [ ] **Step 1: Write `Spinner.svelte`**

```svelte
<script lang="ts">
	interface Props {
		size?: 'sm' | 'md' | 'lg';
	}

	let { size = 'md' }: Props = $props();
</script>

<svg
	class="spinner spinner--{size}"
	viewBox="0 0 24 24"
	fill="none"
	xmlns="http://www.w3.org/2000/svg"
>
	<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" opacity="0.25" />
	<path
		d="M12 2a10 10 0 0 1 10 10"
		stroke="currentColor"
		stroke-width="3"
		stroke-linecap="round"
	/>
</svg>

<style lang="scss">
	.spinner {
		animation: spin 0.8s linear infinite;

		&--sm { width: 1rem; height: 1rem; }
		&--md { width: 1.5rem; height: 1.5rem; }
		&--lg { width: 2rem; height: 2rem; }
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
```

- [ ] **Step 2: Write `StatusBadge.svelte`**

```svelte
<script lang="ts">
	import type { TaskStatus } from '$lib/api/types';

	interface Props {
		status: TaskStatus;
	}

	let { status }: Props = $props();

	const config: Record<TaskStatus, { label: string; className: string }> = {
		pending: { label: 'Pending', className: 'badge--neutral' },
		running: { label: 'Running', className: 'badge--info' },
		completed: { label: 'Completed', className: 'badge--success' },
		failed: { label: 'Failed', className: 'badge--error' },
		cancelled: { label: 'Cancelled', className: 'badge--warning' }
	};

	const current = $derived(config[status] ?? config.pending);
</script>

<span class="badge {current.className}" class:badge--animated={status === 'running'}>
	{current.label}
</span>

<style lang="scss">
	.badge {
		@include badge;

		&--neutral { background: $neutral-100; color: $neutral-700; }
		&--info { background: $info-100; color: $info-600; }
		&--success { background: $success-100; color: $success-700; }
		&--error { background: $error-100; color: $error-700; }
		&--warning { background: $warning-100; color: $warning-700; }

		&--animated::before {
			content: '';
			width: 6px;
			height: 6px;
			border-radius: 50%;
			background: currentColor;
			animation: pulse 1.5s ease-in-out infinite;
		}
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
</style>
```

- [ ] **Step 3: Write `ProviderBadge.svelte`**

```svelte
<script lang="ts">
	interface Props {
		name: string;
	}

	let { name }: Props = $props();

	const colors: Record<string, string> = {
		anthropic: 'badge--amber',
		openai: 'badge--emerald',
		google: 'badge--blue',
		ollama: 'badge--purple',
		bedrock: 'badge--orange',
		vertex: 'badge--cyan',
		openrouter: 'badge--pink'
	};

	const colorClass = $derived(colors[name.toLowerCase()] ?? 'badge--neutral');
</script>

<span class="badge {colorClass}">{name}</span>

<style lang="scss">
	.badge {
		@include badge;

		&--amber { background: #fef3c7; color: #92400e; }
		&--emerald { background: $success-100; color: $success-700; }
		&--blue { background: $info-100; color: $info-600; }
		&--purple { background: #ede9fe; color: #5b21b6; }
		&--orange { background: #ffedd5; color: #9a3412; }
		&--cyan { background: #cffafe; color: #155e75; }
		&--pink { background: #fce7f3; color: #9d174d; }
		&--neutral { background: $neutral-100; color: $neutral-700; }
	}
</style>
```

- [ ] **Step 4: Write `EmptyState.svelte`**

```svelte
<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description: string;
		icon?: Snippet;
		action?: Snippet;
	}

	let { title, description, icon, action }: Props = $props();
</script>

<div class="empty">
	<div class="empty__icon">
		{#if icon}
			{@render icon()}
		{:else}
			<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
			</svg>
		{/if}
	</div>
	<h3 class="empty__title">{title}</h3>
	<p class="empty__desc">{description}</p>
	{#if action}
		<div class="empty__action">
			{@render action()}
		</div>
	{/if}
</div>

<style lang="scss">
	.empty {
		@include flex-center;
		flex-direction: column;
		padding: $space-16 $space-4;
		text-align: center;
		border: 2px dashed $neutral-200;
		border-radius: $radius-xl;

		&__icon { color: $neutral-400; margin-bottom: $space-3; }
		&__title { font-size: $text-sm; font-weight: $font-semibold; color: $neutral-900; }
		&__desc { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		&__action { margin-top: $space-4; }
	}
</style>
```

- [ ] **Step 5: Write `TopBar.svelte`**

```svelte
<script lang="ts">
	import { auth, currentUser, isAdmin, userRole } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	interface Props {
		title?: string;
	}

	let { title }: Props = $props();

	function handleLogout() {
		auth.logout();
		goto('/login');
	}

	function switchView() {
		const role = $userRole;
		if (role === 'admin') {
			// Toggle between admin and user view
			const isAdminRoute = window.location.pathname.startsWith('/dashboard') ||
				window.location.pathname.startsWith('/users') ||
				window.location.pathname.startsWith('/token-usage') ||
				window.location.pathname.startsWith('/observability') ||
				window.location.pathname.startsWith('/providers') ||
				window.location.pathname.startsWith('/channels') ||
				window.location.pathname.startsWith('/vm-settings') ||
				window.location.pathname.startsWith('/audit');

			goto(isAdminRoute ? '/home' : '/dashboard');
		}
	}
</script>

<header class="topbar">
	<div class="topbar__left">
		{#if title}
			<h1 class="topbar__title">{title}</h1>
		{/if}
	</div>

	<div class="topbar__right">
		{#if $isAdmin}
			<button class="topbar__switch" onclick={switchView}>
				Switch View
			</button>
		{/if}

		{#if $currentUser}
			<span class="topbar__user">{$currentUser.name || $currentUser.email}</span>
		{/if}

		<button class="topbar__logout" onclick={handleLogout}>
			Logout
		</button>
	</div>
</header>

<style lang="scss">
	.topbar {
		@include flex-between;
		height: $topbar-height;
		padding: 0 $space-6;
		border-bottom: 1px solid $neutral-200;
		background: $neutral-0;

		&__left { display: flex; align-items: center; }

		&__title {
			font-size: $text-lg;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__right {
			display: flex;
			align-items: center;
			gap: $space-4;
		}

		&__switch {
			@include btn;
			background: $primary-50;
			color: $primary-700;
			font-size: $text-xs;
			padding: $space-1 $space-3;

			&:hover { background: $primary-100; }
		}

		&__user {
			font-size: $text-sm;
			color: $neutral-600;
		}

		&__logout {
			@include btn;
			font-size: $text-sm;
			color: $neutral-500;
			background: transparent;

			&:hover { color: $error-600; }
		}
	}
</style>
```

- [ ] **Step 6: Write `Sidebar.svelte`**

```svelte
<script lang="ts">
	import { page } from '$app/state';

	interface NavItem {
		name: string;
		href: string;
		icon: string;
	}

	interface Props {
		items: NavItem[];
		title?: string;
	}

	let { items, title = 'ForgeBox' }: Props = $props();
</script>

<aside class="sidebar">
	<div class="sidebar__logo">
		<svg class="sidebar__icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
		</svg>
		<span class="sidebar__title">{title}</span>
	</div>

	<nav class="sidebar__nav">
		{#each items as item}
			{@const isActive = page.url.pathname === item.href ||
				(item.href !== '/' && page.url.pathname.startsWith(item.href))}
			<a
				href={item.href}
				class="sidebar__link"
				class:sidebar__link--active={isActive}
			>
				<span class="sidebar__link-icon">{item.icon}</span>
				{item.name}
			</a>
		{/each}
	</nav>

	<div class="sidebar__footer">
		<div class="sidebar__status">
			<span class="sidebar__dot"></span>
			<span>Gateway connected</span>
		</div>
		<p class="sidebar__version">ForgeBox v0.1.0</p>
	</div>
</aside>

<style lang="scss">
	.sidebar {
		display: flex;
		flex-direction: column;
		width: $sidebar-width;
		border-right: 1px solid $neutral-200;
		background: $neutral-0;
		height: 100%;

		&__logo {
			display: flex;
			align-items: center;
			gap: $space-2;
			height: $topbar-height;
			padding: 0 $space-6;
			border-bottom: 1px solid $neutral-200;
		}

		&__icon { color: $primary-600; }

		&__title {
			font-size: $text-lg;
			font-weight: $font-bold;
			color: $neutral-900;
		}

		&__nav {
			flex: 1;
			padding: $space-4 $space-3;
			display: flex;
			flex-direction: column;
			gap: $space-1;
			@include scrollbar-thin;
			overflow-y: auto;
		}

		&__link {
			display: flex;
			align-items: center;
			gap: $space-3;
			padding: $space-2 $space-3;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-600;
			border-radius: $radius-lg;
			transition: all $transition-fast;

			&:hover {
				background: $neutral-50;
				color: $neutral-900;
			}

			&--active {
				background: $primary-50;
				color: $primary-700;
			}
		}

		&__link-icon {
			width: 1.25rem;
			text-align: center;
			flex-shrink: 0;
		}

		&__footer {
			border-top: 1px solid $neutral-200;
			padding: $space-4;
		}

		&__status {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__dot {
			width: 8px;
			height: 8px;
			border-radius: 50%;
			background: $success-500;
		}

		&__version {
			margin-top: $space-1;
			font-size: $text-xs;
			color: $neutral-400;
		}
	}
</style>
```

- [ ] **Step 7: Write `TaskStream.svelte`**

```svelte
<script lang="ts">
	import type { TaskEvent } from '$lib/api/types';

	interface Props {
		events: TaskEvent[];
		isRunning: boolean;
	}

	let { events, isRunning }: Props = $props();
	let container: HTMLDivElement | undefined = $state();

	$effect(() => {
		// Scroll to bottom on new events
		if (events.length && container) {
			container.scrollTop = container.scrollHeight;
		}
	});
</script>

{#if events.length > 0 || isRunning}
	<div class="stream">
		<div class="stream__header">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polyline points="4 17 10 11 4 5" /><line x1="12" y1="19" x2="20" y2="19" />
			</svg>
			<span>Output</span>
			{#if isRunning}
				<span class="stream__live">
					<span class="stream__pulse"></span>
					Streaming
				</span>
			{/if}
		</div>

		<div class="stream__body" bind:this={container}>
			{#each events as event}
				{#if event.type === 'text_delta'}
					<span class="stream__text">{event.text}</span>
				{:else if event.type === 'tool_call'}
					<div class="stream__tool">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
						</svg>
						<div>
							<span class="stream__tool-name">{event.tool_call?.name}</span>
							<pre class="stream__tool-input">{event.tool_call?.input}</pre>
						</div>
					</div>
				{:else if event.type === 'tool_result'}
					<div class="stream__result" class:stream__result--error={event.result?.is_error}>
						{event.result?.content}
					</div>
				{:else if event.type === 'error'}
					<div class="stream__error">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
						</svg>
						{event.error}
					</div>
				{:else if event.type === 'done'}
					<div class="stream__done">Task completed</div>
				{/if}
			{/each}
			{#if isRunning}
				<span class="stream__cursor"></span>
			{/if}
		</div>
	</div>
{/if}

<style lang="scss">
	.stream {
		@include card;
		overflow: hidden;

		&__header {
			@include flex-between;
			padding: $space-2 $space-4;
			border-bottom: 1px solid $neutral-200;
			background: $neutral-50;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-700;

			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__live {
			margin-left: auto;
			display: flex;
			align-items: center;
			gap: $space-1;
			font-size: $text-xs;
			color: $info-600;
		}

		&__pulse {
			width: 6px;
			height: 6px;
			border-radius: 50%;
			background: $info-600;
			animation: pulse 1.5s ease-in-out infinite;
		}

		&__body {
			max-height: 500px;
			overflow-y: auto;
			@include scrollbar-thin;
			background: $neutral-900;
			padding: $space-4;
			font-family: $font-mono;
			font-size: $text-sm;
		}

		&__text {
			color: $neutral-100;
			white-space: pre-wrap;
		}

		&__tool {
			display: flex;
			align-items: flex-start;
			gap: $space-2;
			margin: $space-2 0;
			padding: $space-2;
			border: 1px solid $neutral-700;
			border-radius: $radius-md;
			background: $neutral-800;
			color: $primary-300;
		}

		&__tool-name {
			font-weight: $font-semibold;
			color: $primary-300;
		}

		&__tool-input {
			margin-top: $space-1;
			font-size: $text-xs;
			color: $neutral-400;
			overflow-x: auto;
		}

		&__result {
			margin: $space-1 0;
			padding: $space-2;
			border: 1px solid $neutral-700;
			border-radius: $radius-md;
			background: $neutral-800;
			color: $neutral-300;
			font-size: $text-xs;
			white-space: pre-wrap;

			&--error {
				border-color: $error-700;
				background: rgba($error-600, 0.15);
				color: $error-500;
			}
		}

		&__error {
			display: flex;
			align-items: center;
			gap: $space-2;
			margin: $space-2 0;
			color: $error-500;
		}

		&__done {
			margin-top: $space-3;
			padding-top: $space-2;
			border-top: 1px solid $neutral-700;
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__cursor {
			display: inline-block;
			width: 8px;
			height: 16px;
			background: $neutral-400;
			animation: pulse 1s ease-in-out infinite;
		}
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
</style>
```

- [ ] **Step 8: Commit**

```bash
git add web/src/lib/components/
git commit -m "feat(web): add shared Svelte components (Sidebar, TopBar, StatusBadge, etc.)"
```

---

### Task 6: Root Layout, Login Page & Role Routing

Set up the root layout with auth checks, login page, and role-based redirect.

**Files:**
- Create: `web/src/routes/+layout.svelte`
- Create: `web/src/routes/+layout.ts`
- Create: `web/src/routes/+page.svelte`
- Create: `web/src/routes/login/+page.svelte`

- [ ] **Step 1: Write `+layout.ts` (disable SSR for SPA)**

```ts
export const ssr = false;
export const prerender = false;
```

- [ ] **Step 2: Write root `+layout.svelte`**

```svelte
<script lang="ts">
	import '$lib/styles/global.scss';
	import { isAuthenticated } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	$effect(() => {
		if (!$isAuthenticated && page.url.pathname !== '/login') {
			goto('/login');
		}
	});
</script>

{@render children()}
```

- [ ] **Step 3: Write root `+page.svelte` (role redirect)**

```svelte
<script lang="ts">
	import { isAuthenticated, isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	$effect(() => {
		if ($isAuthenticated) {
			goto($isAdmin ? '/dashboard' : '/home');
		} else {
			goto('/login');
		}
	});
</script>
```

- [ ] **Step 4: Write `login/+page.svelte`**

```svelte
<script lang="ts">
	import { auth, isAuthenticated, isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let loading = $state(false);

	$effect(() => {
		if ($isAuthenticated) {
			goto($isAdmin ? '/dashboard' : '/home');
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!email.trim() || !password.trim()) return;

		loading = true;
		error = null;

		try {
			await auth.login(email.trim(), password.trim());
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login">
	<div class="login__card">
		<div class="login__header">
			<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
			</svg>
			<h1>ForgeBox</h1>
			<p>Sign in to your account</p>
		</div>

		{#if error}
			<div class="login__error">{error}</div>
		{/if}

		<form class="login__form" onsubmit={handleSubmit}>
			<label class="login__field">
				<span>Email</span>
				<input
					type="email"
					bind:value={email}
					placeholder="you@example.com"
					disabled={loading}
					required
				/>
			</label>

			<label class="login__field">
				<span>Password</span>
				<input
					type="password"
					bind:value={password}
					placeholder="Enter password"
					disabled={loading}
					required
				/>
			</label>

			<button type="submit" class="btn-primary login__submit" disabled={loading}>
				{loading ? 'Signing in...' : 'Sign in'}
			</button>
		</form>
	</div>
</div>

<style lang="scss">
	.login {
		@include flex-center;
		min-height: 100vh;
		background: $neutral-50;

		&__card {
			@include card;
			width: 100%;
			max-width: 400px;
			padding: $space-8;
		}

		&__header {
			text-align: center;
			margin-bottom: $space-8;
			color: $primary-600;

			h1 {
				margin-top: $space-3;
				font-size: $text-2xl;
				font-weight: $font-bold;
				color: $neutral-900;
			}

			p {
				margin-top: $space-1;
				font-size: $text-sm;
				color: $neutral-500;
			}
		}

		&__error {
			padding: $space-3;
			margin-bottom: $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}

		&__form {
			display: flex;
			flex-direction: column;
			gap: $space-4;
		}

		&__field {
			display: flex;
			flex-direction: column;
			gap: $space-1;

			span {
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-700;
			}

			input {
				@include input-base;
			}
		}

		&__submit {
			width: 100%;
			margin-top: $space-2;
		}
	}
</style>
```

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/
git commit -m "feat(web): add root layout, login page, role-based routing"
```

---

### Task 7: Admin Layout & Sidebar Navigation

Create the admin layout group with sidebar and route guard.

**Files:**
- Create: `web/src/routes/(admin)/+layout.svelte`

- [ ] **Step 1: Write `(admin)/+layout.svelte`**

```svelte
<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import { isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const navItems = [
		{ name: 'Dashboard', href: '/dashboard', icon: '📊' },
		{ name: 'Users & Teams', href: '/users', icon: '👥' },
		{ name: 'Token Usage', href: '/token-usage', icon: '🔢' },
		{ name: 'Observability', href: '/observability', icon: '📈' },
		{ name: 'Providers', href: '/providers', icon: '🔌' },
		{ name: 'Channels', href: '/channels', icon: '📡' },
		{ name: 'VM Settings', href: '/vm-settings', icon: '⚙️' },
		{ name: 'Audit Log', href: '/audit', icon: '🛡️' }
	];

	$effect(() => {
		if (!$isAdmin) {
			goto('/home');
		}
	});
</script>

<div class="layout">
	<Sidebar items={navItems} />
	<div class="layout__main">
		<TopBar />
		<main class="layout__content">
			{@render children()}
		</main>
	</div>
</div>

<style lang="scss">
	.layout {
		display: flex;
		height: 100vh;
		overflow: hidden;

		&__main {
			flex: 1;
			display: flex;
			flex-direction: column;
			overflow: hidden;
		}

		&__content {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-8 $space-6;
			max-width: 80rem;
		}
	}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/routes/\(admin\)/
git commit -m "feat(web): add admin layout with sidebar and role guard"
```

---

### Task 8: User Layout & Sidebar Navigation

Create the user layout group with sidebar.

**Files:**
- Create: `web/src/routes/(user)/+layout.svelte`

- [ ] **Step 1: Write `(user)/+layout.svelte`**

```svelte
<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const navItems = [
		{ name: 'Home', href: '/home', icon: '🏠' },
		{ name: 'Task Runner', href: '/tasks/new', icon: '▶️' },
		{ name: 'My Tasks', href: '/tasks', icon: '📋' },
		{ name: 'Workflows', href: '/workflows', icon: '🔄' },
		{ name: 'Team', href: '/team', icon: '👥' },
		{ name: 'Settings', href: '/settings', icon: '⚙️' }
	];
</script>

<div class="layout">
	<Sidebar items={navItems} />
	<div class="layout__main">
		<TopBar />
		<main class="layout__content">
			{@render children()}
		</main>
	</div>
</div>

<style lang="scss">
	.layout {
		display: flex;
		height: 100vh;
		overflow: hidden;

		&__main {
			flex: 1;
			display: flex;
			flex-direction: column;
			overflow: hidden;
		}

		&__content {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-8 $space-6;
			max-width: 80rem;
		}
	}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add web/src/routes/\(user\)/
git commit -m "feat(web): add user layout with sidebar navigation"
```

---

### Task 9: Admin Pages — Dashboard, Users, Token Usage, Observability

Create the first four admin pages.

**Files:**
- Create: `web/src/routes/(admin)/dashboard/+page.svelte`
- Create: `web/src/routes/(admin)/users/+page.svelte`
- Create: `web/src/routes/(admin)/token-usage/+page.svelte`
- Create: `web/src/routes/(admin)/observability/+page.svelte`

- [ ] **Step 1: Write `dashboard/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { listTasks } from '$lib/api/client';
	import type { Task } from '$lib/api/types';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let tasks = $state<Task[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			tasks = await listTasks();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});

	const running = $derived(tasks.filter((t) => t.status === 'running').length);
	const completed = $derived(tasks.filter((t) => t.status === 'completed').length);
	const failed = $derived(tasks.filter((t) => t.status === 'failed').length);
	const recent = $derived(tasks.slice(0, 5));

	const stats = $derived([
		{ label: 'Total Tasks', value: tasks.length, color: 'primary' },
		{ label: 'Running', value: running, color: 'info' },
		{ label: 'Completed', value: completed, color: 'success' },
		{ label: 'Failed', value: failed, color: 'error' }
	]);
</script>

<div class="page">
	<div class="page__header">
		<h1>Dashboard</h1>
		<p>Overview of your ForgeBox instance</p>
	</div>

	{#if loading}
		<p class="page__loading">Loading...</p>
	{:else if error}
		<p class="page__error">Error: {error}</p>
	{:else}
		<div class="stats">
			{#each stats as stat}
				<div class="stat stat--{stat.color}">
					<p class="stat__label">{stat.label}</p>
					<p class="stat__value">{stat.value}</p>
				</div>
			{/each}
		</div>

		<div class="recent">
			<div class="recent__header">
				<h2>Recent Tasks</h2>
				<a href="/tasks/new" class="btn-primary">Run Task</a>
			</div>

			{#if recent.length === 0}
				<div class="recent__empty">
					No tasks yet. <a href="/tasks/new">Run a task</a> to get started.
				</div>
			{:else}
				<table class="table">
					<thead>
						<tr>
							<th>Status</th>
							<th>Prompt</th>
							<th>Provider</th>
							<th>Cost</th>
							<th>Created</th>
						</tr>
					</thead>
					<tbody>
						{#each recent as task}
							<tr>
								<td><StatusBadge status={task.status} /></td>
								<td class="table__truncate">
									{task.prompt.length > 80 ? `${task.prompt.slice(0, 80)}...` : task.prompt}
								</td>
								<td>{task.provider}</td>
								<td>${task.cost.toFixed(4)}</td>
								<td class="table__muted">{new Date(task.created_at).toLocaleDateString()}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}
		</div>

		<h2 class="section-title">Quick Actions</h2>
		<div class="actions">
			{#each [
				{ label: 'Run a Task', desc: 'Execute an AI task in a secure VM', href: '/tasks/new' },
				{ label: 'View Tasks', desc: 'Browse active and past tasks', href: '/tasks' },
				{ label: 'Configure Providers', desc: 'Manage LLM provider settings', href: '/providers' }
			] as action}
				<a href={action.href} class="action-card">
					<div>
						<p class="action-card__title">{action.label}</p>
						<p class="action-card__desc">{action.desc}</p>
					</div>
					<span class="action-card__arrow">→</span>
				</a>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}

		&__loading { color: $neutral-500; }
		&__error { color: $error-600; }
	}

	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: $space-4;
		margin-bottom: $space-8;
	}

	.stat {
		@include card;
		padding: $space-5;

		&__label { font-size: $text-sm; color: $neutral-500; }
		&__value { font-size: $text-2xl; font-weight: $font-semibold; color: $neutral-900; margin-top: $space-1; }

		&--primary { border-left: 3px solid $primary-500; }
		&--info { border-left: 3px solid $info-500; }
		&--success { border-left: 3px solid $success-500; }
		&--error { border-left: 3px solid $error-500; }
	}

	.recent {
		@include card;
		margin-bottom: $space-8;

		&__header {
			@include flex-between;
			padding: $space-4 $space-5;
			border-bottom: 1px solid $neutral-200;
		}

		&__empty {
			padding: $space-8;
			text-align: center;
			font-size: $text-sm;
			color: $neutral-400;

			a { color: $primary-600; text-decoration: underline; }
		}
	}

	.table {
		text-align: left;
		font-size: $text-sm;

		thead { border-bottom: 1px solid $neutral-100; }

		th {
			padding: $space-3 $space-5;
			font-size: $text-xs;
			font-weight: $font-medium;
			text-transform: uppercase;
			color: $neutral-500;
		}

		td { padding: $space-3 $space-5; }

		tbody tr:hover { background: $neutral-50; }
		tbody tr + tr { border-top: 1px solid $neutral-100; }

		&__truncate { max-width: 300px; @include truncate; color: $neutral-700; }
		&__muted { color: $neutral-400; }
	}

	.section-title { margin-bottom: $space-4; }

	.actions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: $space-4;
	}

	.action-card {
		@include card;
		@include flex-between;
		padding: $space-5;
		transition: border-color $transition-fast;

		&:hover { border-color: $primary-300; }

		&__title { font-weight: $font-medium; color: $neutral-900; }
		&__desc { font-size: $text-sm; color: $neutral-500; }
		&__arrow { color: $neutral-400; font-size: $text-xl; }
	}
</style>
```

- [ ] **Step 2: Write `users/+page.svelte`**

```svelte
<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Users & Teams</h1>
		<p>Manage users, roles, and team assignments</p>
	</div>

	<EmptyState
		title="No users to display"
		description="User management will be available when the auth backend is connected."
	/>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
```

- [ ] **Step 3: Write `token-usage/+page.svelte`**

```svelte
<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Token Usage</h1>
		<p>Usage statistics per user, team, and provider</p>
	</div>

	<EmptyState
		title="No usage data"
		description="Token usage tracking will appear here once tasks have been run."
	/>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
```

- [ ] **Step 4: Write `observability/+page.svelte`**

```svelte
<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Observability</h1>
		<p>Logs, task execution traces, and error rates</p>
	</div>

	<EmptyState
		title="No trace data"
		description="Execution traces and logs will appear here once the observability pipeline is connected."
	/>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
```

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/\(admin\)/
git commit -m "feat(web): add admin pages — dashboard, users, token-usage, observability"
```

---

### Task 10: Admin Pages — Providers, Channels, VM Settings, Audit

Create the remaining four admin pages.

**Files:**
- Create: `web/src/routes/(admin)/providers/+page.svelte`
- Create: `web/src/routes/(admin)/channels/+page.svelte`
- Create: `web/src/routes/(admin)/vm-settings/+page.svelte`
- Create: `web/src/routes/(admin)/audit/+page.svelte`

- [ ] **Step 1: Write `providers/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import EmptyState from '$lib/components/EmptyState.svelte';

	let providers = $state<Provider[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			providers = await listProviders();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>Providers</h1>
			<p>Configure LLM providers</p>
		</div>
		<button class="btn-secondary">Add Provider</button>
	</div>

	{#if loading}
		<p class="text-muted">Loading...</p>
	{:else if error}
		<p class="text-error">Error: {error}</p>
	{:else if providers.length === 0}
		<EmptyState
			title="No providers configured"
			description="Add an LLM provider to get started."
		/>
	{:else}
		<div class="grid">
			{#each providers as provider}
				<div class="provider-card">
					<div>
						<p class="provider-card__name">{provider.name}</p>
						<p class="provider-card__meta">v{provider.version} {provider.builtin ? '(built-in)' : ''}</p>
					</div>
					<span class="provider-card__badge">Active</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			@include flex-between;
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.text-muted { color: $neutral-500; }
	.text-error { color: $error-600; }

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: $space-4;
	}

	.provider-card {
		@include card;
		@include flex-between;
		padding: $space-5;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; }

		&__badge {
			@include badge;
			background: $success-100;
			color: $success-700;
		}
	}
</style>
```

- [ ] **Step 2: Write `channels/+page.svelte`**

```svelte
<script lang="ts">
	const channels = [
		{ name: 'Slack', configured: false },
		{ name: 'Discord', configured: false },
		{ name: 'Webhook', configured: true },
		{ name: 'Email', configured: false }
	];
</script>

<div class="page">
	<div class="page__header">
		<h1>Channels</h1>
		<p>Configure input channels for receiving tasks</p>
	</div>

	<div class="grid">
		{#each channels as channel}
			<div class="channel-card">
				<div>
					<p class="channel-card__name">{channel.name}</p>
					<p class="channel-card__meta">{channel.configured ? 'Configured' : 'Not configured'}</p>
				</div>
				<span class="channel-card__badge" class:channel-card__badge--active={channel.configured}>
					{channel.configured ? 'Active' : 'Inactive'}
				</span>
			</div>
		{/each}
	</div>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: $space-4;
	}

	.channel-card {
		@include card;
		@include flex-between;
		padding: $space-5;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; }

		&__badge {
			@include badge;
			background: $neutral-100;
			color: $neutral-500;

			&--active { background: $success-100; color: $success-700; }
		}
	}
</style>
```

- [ ] **Step 3: Write `vm-settings/+page.svelte`**

```svelte
<script lang="ts">
	const vmDefaults = [
		{ label: 'Memory', value: '512 MB' },
		{ label: 'vCPUs', value: '1' },
		{ label: 'Timeout', value: '5 minutes' },
		{ label: 'Network Access', value: 'Disabled' }
	];
</script>

<div class="page">
	<div class="page__header">
		<h1>VM Settings</h1>
		<p>Firecracker VM defaults, network policies, and resource limits</p>
	</div>

	<h2>VM Defaults</h2>
	<div class="settings-card">
		{#each vmDefaults as item}
			<div class="settings-row">
				<span class="settings-row__label">{item.label}</span>
				<span class="settings-row__value">{item.value}</span>
			</div>
		{/each}
	</div>

	<h2 class="section-title">Storage</h2>
	<div class="settings-card">
		<div class="settings-info">
			Local filesystem storage is active. Configure S3 or GCS backends in the
			ForgeBox configuration file.
		</div>
	</div>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.section-title { margin-top: $space-8; margin-bottom: $space-4; }

	.settings-card {
		@include card;
		margin-top: $space-4;
	}

	.settings-row {
		@include flex-between;
		padding: $space-3 $space-5;
		font-size: $text-sm;

		& + & { border-top: 1px solid $neutral-100; }

		&__label { color: $neutral-600; }
		&__value { font-weight: $font-medium; color: $neutral-900; }
	}

	.settings-info {
		padding: $space-4 $space-5;
		font-size: $text-sm;
		color: $neutral-600;
	}
</style>
```

- [ ] **Step 4: Write `audit/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { format } from 'date-fns';
	import { listAuditEntries } from '$lib/api/client';
	import type { AuditEntry } from '$lib/api/types';
	import EmptyState from '$lib/components/EmptyState.svelte';

	let entries = $state<AuditEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Filters
	let userId = $state('');
	let decision = $state<'all' | 'allow' | 'deny'>('all');
	let dateFrom = $state('');
	let dateTo = $state('');

	const filtered = $derived(
		entries.filter((e) => {
			if (userId && !e.user_id.toLowerCase().includes(userId.toLowerCase())) return false;
			if (decision !== 'all' && e.decision !== decision) return false;
			if (dateFrom && e.timestamp < dateFrom) return false;
			if (dateTo && e.timestamp > dateTo) return false;
			return true;
		})
	);

	onMount(async () => {
		try {
			entries = await listAuditEntries();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});
</script>

<div class="page">
	<div class="page__header">
		<h1>Audit Log</h1>
		<p>Track all tool calls and permission decisions</p>
	</div>

	<div class="filters">
		<label class="filter">
			<span>User ID</span>
			<input type="text" placeholder="Filter by user..." bind:value={userId} />
		</label>
		<label class="filter">
			<span>Decision</span>
			<select bind:value={decision}>
				<option value="all">All</option>
				<option value="allow">Allow</option>
				<option value="deny">Deny</option>
			</select>
		</label>
		<label class="filter">
			<span>From</span>
			<input type="text" placeholder="YYYY-MM-DD" bind:value={dateFrom} />
		</label>
		<label class="filter">
			<span>To</span>
			<input type="text" placeholder="YYYY-MM-DD" bind:value={dateTo} />
		</label>
	</div>

	{#if loading}
		<p class="text-muted">Loading...</p>
	{:else if error}
		<p class="text-error">Error: {error}</p>
	{:else if filtered.length === 0}
		<EmptyState
			title="No audit entries found"
			description="Audit entries will appear here when tools are executed."
		/>
	{:else}
		<div class="table-wrap">
			<table class="table">
				<thead>
					<tr>
						<th>Timestamp</th>
						<th>User</th>
						<th>Action</th>
						<th>Tool</th>
						<th>Decision</th>
						<th>Reason</th>
					</tr>
				</thead>
				<tbody>
					{#each filtered as entry}
						<tr>
							<td class="table__nowrap table__muted">
								{format(new Date(entry.timestamp), 'MMM d, yyyy HH:mm:ss')}
							</td>
							<td class="table__mono">{entry.user_id.slice(0, 8)}</td>
							<td>{entry.action}</td>
							<td class="table__mono">{entry.tool ?? '-'}</td>
							<td>
								<span class="decision" class:decision--allow={entry.decision === 'allow'} class:decision--deny={entry.decision === 'deny'}>
									{entry.decision}
								</span>
							</td>
							<td class="table__truncate table__muted">{entry.reason ?? '-'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.text-muted { color: $neutral-500; }
	.text-error { color: $error-600; }

	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: $space-3;
		align-items: flex-end;
		margin-bottom: $space-6;
	}

	.filter {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		span { font-size: $text-xs; font-weight: $font-medium; color: $neutral-600; }
		input, select { @include input-base; width: auto; }
	}

	.table-wrap {
		@include card;
		overflow: hidden;
	}

	.table {
		text-align: left;
		font-size: $text-sm;

		thead {
			background: $neutral-50;
			border-bottom: 1px solid $neutral-100;
		}

		th {
			padding: $space-3 $space-5;
			font-size: $text-xs;
			font-weight: $font-medium;
			text-transform: uppercase;
			color: $neutral-500;
		}

		td { padding: $space-3 $space-5; }

		tbody tr:hover { background: $neutral-50; }
		tbody tr + tr { border-top: 1px solid $neutral-100; }

		&__nowrap { white-space: nowrap; }
		&__mono { font-family: $font-mono; font-size: $text-xs; color: $neutral-700; }
		&__muted { color: $neutral-400; }
		&__truncate { max-width: 200px; @include truncate; }
	}

	.decision {
		@include badge;

		&--allow { background: $success-100; color: $success-700; }
		&--deny { background: $error-100; color: $error-700; }
	}
</style>
```

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/\(admin\)/
git commit -m "feat(web): add admin pages — providers, channels, vm-settings, audit"
```

---

### Task 11: User Pages — Home, Tasks List, Task Runner

Create the user-facing pages for home, task history, and task runner.

**Files:**
- Create: `web/src/routes/(user)/home/+page.svelte`
- Create: `web/src/routes/(user)/tasks/+page.svelte`
- Create: `web/src/routes/(user)/tasks/new/+page.svelte`

- [ ] **Step 1: Write `home/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { listTasks } from '$lib/api/client';
	import type { Task } from '$lib/api/types';
	import { currentUser } from '$lib/stores/auth';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let tasks = $state<Task[]>([]);
	let loading = $state(true);

	const recent = $derived(tasks.slice(0, 5));
	const running = $derived(tasks.filter((t) => t.status === 'running').length);

	onMount(async () => {
		try {
			tasks = await listTasks();
		} catch {
			// silently fail on home page
		} finally {
			loading = false;
		}
	});
</script>

<div class="page">
	<div class="page__header">
		<h1>Welcome{$currentUser ? `, ${$currentUser.name || $currentUser.email}` : ''}</h1>
		<p>Your personal dashboard</p>
	</div>

	<div class="quick-stats">
		<div class="quick-stat">
			<p class="quick-stat__value">{tasks.length}</p>
			<p class="quick-stat__label">Total Tasks</p>
		</div>
		<div class="quick-stat">
			<p class="quick-stat__value">{running}</p>
			<p class="quick-stat__label">Running</p>
		</div>
	</div>

	<div class="actions">
		<a href="/tasks/new" class="action-card action-card--primary">
			<span class="action-card__icon">▶</span>
			<div>
				<p class="action-card__title">Run a Task</p>
				<p class="action-card__desc">Execute an AI task in a secure VM</p>
			</div>
		</a>
		<a href="/tasks" class="action-card">
			<span class="action-card__icon">📋</span>
			<div>
				<p class="action-card__title">My Tasks</p>
				<p class="action-card__desc">View task history and results</p>
			</div>
		</a>
		<a href="/workflows" class="action-card">
			<span class="action-card__icon">🔄</span>
			<div>
				<p class="action-card__title">Workflows</p>
				<p class="action-card__desc">Create reusable automations</p>
			</div>
		</a>
	</div>

	{#if !loading && recent.length > 0}
		<h2 class="section-title">Recent Tasks</h2>
		<div class="recent-list">
			{#each recent as task}
				<a href="/tasks/{task.id}" class="recent-item">
					<div class="recent-item__content">
						<p class="recent-item__prompt">{task.prompt}</p>
						<p class="recent-item__meta">{task.provider} &middot; {new Date(task.created_at).toLocaleDateString()}</p>
					</div>
					<StatusBadge status={task.status} />
				</a>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.quick-stats {
		display: flex;
		gap: $space-4;
		margin-bottom: $space-8;
	}

	.quick-stat {
		@include card;
		padding: $space-5;
		min-width: 150px;

		&__value { font-size: $text-3xl; font-weight: $font-bold; color: $neutral-900; }
		&__label { font-size: $text-sm; color: $neutral-500; }
	}

	.actions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: $space-4;
		margin-bottom: $space-8;
	}

	.action-card {
		@include card;
		display: flex;
		align-items: center;
		gap: $space-4;
		padding: $space-5;
		transition: border-color $transition-fast, box-shadow $transition-fast;

		&:hover { border-color: $primary-300; box-shadow: $shadow-md; }

		&--primary { border-left: 3px solid $primary-500; }

		&__icon { font-size: $text-2xl; }
		&__title { font-weight: $font-medium; color: $neutral-900; }
		&__desc { font-size: $text-sm; color: $neutral-500; }
	}

	.section-title { margin-bottom: $space-4; }

	.recent-list {
		display: flex;
		flex-direction: column;
		gap: $space-2;
	}

	.recent-item {
		@include card;
		@include flex-between;
		padding: $space-4;
		transition: border-color $transition-fast;

		&:hover { border-color: $neutral-300; }

		&__prompt { @include truncate; max-width: 500px; font-size: $text-sm; font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; margin-top: $space-1; }
	}
</style>
```

- [ ] **Step 2: Write `tasks/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { formatDistanceToNow } from 'date-fns';
	import { listTasks } from '$lib/api/client';
	import type { Task } from '$lib/api/types';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import ProviderBadge from '$lib/components/ProviderBadge.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';

	let tasks = $state<Task[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			tasks = await listTasks();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>My Tasks</h1>
			<p>History of your tasks, status, and results</p>
		</div>
		<a href="/tasks/new" class="btn-primary">New Task</a>
	</div>

	{#if loading}
		<p class="text-muted">Loading...</p>
	{:else if error}
		<p class="text-error">Error: {error}</p>
	{:else if tasks.length === 0}
		<EmptyState
			title="No tasks yet"
			description="Run your first task to get started."
		>
			{#snippet action()}
				<a href="/tasks/new" class="btn-primary">Run a Task</a>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="task-list">
			{#each tasks as task}
				<a href="/tasks/{task.id}" class="task-row">
					<div class="task-row__main">
						<p class="task-row__prompt">{task.prompt}</p>
						<div class="task-row__meta">
							<ProviderBadge name={task.provider} />
							{#if task.model}
								<span class="task-row__model">{task.model}</span>
							{/if}
							{#if task.cost > 0}
								<span>${task.cost.toFixed(4)}</span>
							{/if}
							<span>{formatDistanceToNow(new Date(task.created_at), { addSuffix: true })}</span>
						</div>
					</div>
					<StatusBadge status={task.status} />
				</a>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			@include flex-between;
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.text-muted { color: $neutral-500; }
	.text-error { color: $error-600; }

	.task-list {
		display: flex;
		flex-direction: column;
		gap: $space-2;
	}

	.task-row {
		@include card;
		@include flex-between;
		padding: $space-4 $space-5;
		transition: box-shadow $transition-fast;

		&:hover { box-shadow: $shadow-md; }

		&__main { min-width: 0; flex: 1; }

		&__prompt {
			@include truncate;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-900;
		}

		&__meta {
			display: flex;
			flex-wrap: wrap;
			align-items: center;
			gap: $space-3;
			margin-top: $space-2;
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__model {
			font-family: $font-mono;
			font-size: $text-xs;
		}
	}
</style>
```

- [ ] **Step 3: Write `tasks/new/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders, createTask, streamTask, cancelTask } from '$lib/api/client';
	import type { Provider, TaskEvent } from '$lib/api/types';
	import TaskStream from '$lib/components/TaskStream.svelte';

	let prompt = $state('');
	let provider = $state('');
	let model = $state('');
	let memoryMb = $state(512);
	let networkAccess = $state(false);
	let providers = $state<Provider[]>([]);
	let events = $state<TaskEvent[]>([]);
	let isRunning = $state(false);
	let taskId = $state<string | null>(null);
	let cost = $state<number | null>(null);
	let duration = $state<string | null>(null);
	let error = $state<string | null>(null);
	let stopFn: (() => void) | null = null;

	onMount(async () => {
		try {
			providers = await listProviders();
		} catch {
			// ignore
		}
	});

	async function handleRun() {
		if (!prompt.trim() || isRunning) return;
		events = [];
		cost = null;
		duration = null;
		error = null;
		isRunning = true;

		try {
			const start = Date.now();
			const res = await createTask({
				prompt: prompt.trim(),
				provider: provider || undefined,
				model: model || undefined,
				memory_mb: memoryMb,
				network_access: networkAccess
			});
			taskId = res.task_id;

			stopFn = streamTask(
				res.task_id,
				(event) => {
					events = [...events, event];
					if (event.type === 'done') {
						isRunning = false;
						duration = `${((Date.now() - start) / 1000).toFixed(1)}s`;
					}
				},
				(err) => {
					error = err.message;
					isRunning = false;
				}
			);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create task';
			isRunning = false;
		}
	}

	async function handleCancel() {
		if (!taskId) return;
		stopFn?.();
		try {
			await cancelTask(taskId);
		} catch {
			// ignore
		}
		isRunning = false;
	}
</script>

<div class="page">
	<div class="page__header">
		<h1>Run Task</h1>
		<p>Describe what you want to do and ForgeBox will execute it in a secure VM.</p>
	</div>

	<textarea
		class="prompt-input"
		rows="4"
		placeholder="Describe what you want to do in plain English..."
		bind:value={prompt}
		disabled={isRunning}
	></textarea>

	<div class="settings">
		<select class="settings__select" bind:value={provider} disabled={isRunning}>
			<option value="">Provider (auto)</option>
			{#each providers as p}
				<option value={p.name}>{p.name}</option>
			{/each}
		</select>

		<input
			class="settings__input"
			placeholder="Model (default)"
			bind:value={model}
			disabled={isRunning}
		/>

		<select class="settings__select" bind:value={memoryMb} disabled={isRunning}>
			{#each [256, 512, 1024, 2048] as mb}
				<option value={mb}>{mb} MB</option>
			{/each}
		</select>

		<label class="settings__checkbox">
			<input type="checkbox" bind:checked={networkAccess} disabled={isRunning} />
			Network Access
		</label>
	</div>

	<div class="actions">
		{#if isRunning}
			<button class="btn-danger" onclick={handleCancel}>Cancel</button>
		{:else}
			<button class="btn-primary" onclick={handleRun} disabled={!prompt.trim()}>Run</button>
		{/if}
	</div>

	{#if error}
		<p class="error-msg">Error: {error}</p>
	{/if}

	<TaskStream {events} {isRunning} />

	{#if cost !== null && duration}
		<div class="result-summary">
			<span>Cost: <strong>${cost.toFixed(4)}</strong></span>
			<span>Duration: <strong>{duration}</strong></span>
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.prompt-input {
		@include input-base;
		margin-bottom: $space-4;
		resize: vertical;
	}

	.settings {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: $space-3;
		margin-bottom: $space-4;

		&__select, &__input {
			@include input-base;
			width: auto;
			min-width: 150px;
		}

		&__checkbox {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-sm;
			color: $neutral-600;
			cursor: pointer;

			input[type='checkbox'] {
				width: 1rem;
				height: 1rem;
				accent-color: $primary-600;
			}
		}
	}

	.actions {
		display: flex;
		align-items: center;
		gap: $space-3;
		margin-bottom: $space-6;
	}

	.error-msg {
		font-size: $text-sm;
		color: $error-600;
		margin-bottom: $space-4;
	}

	.result-summary {
		display: flex;
		gap: $space-6;
		margin-top: $space-4;
		font-size: $text-sm;
		color: $neutral-500;

		strong { color: $neutral-700; }
	}
</style>
```

- [ ] **Step 4: Commit**

```bash
git add web/src/routes/\(user\)/
git commit -m "feat(web): add user pages — home, tasks list, task runner"
```

---

### Task 12: User Pages — Task Detail, Workflows, Team, Settings

Create the remaining user pages.

**Files:**
- Create: `web/src/routes/(user)/tasks/[id]/+page.svelte`
- Create: `web/src/routes/(user)/workflows/+page.svelte`
- Create: `web/src/routes/(user)/team/+page.svelte`
- Create: `web/src/routes/(user)/settings/+page.svelte`

- [ ] **Step 1: Write `tasks/[id]/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { format } from 'date-fns';
	import { getTask, streamTask, cancelTask } from '$lib/api/client';
	import type { Task, TaskEvent } from '$lib/api/types';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import ProviderBadge from '$lib/components/ProviderBadge.svelte';
	import TaskStream from '$lib/components/TaskStream.svelte';

	let task = $state<Task | null>(null);
	let events = $state<TaskEvent[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let isStreaming = $state(false);
	let stopFn: (() => void) | null = null;

	const taskId = $derived(page.params.id);

	onMount(async () => {
		try {
			task = await getTask(taskId);

			if (task.status === 'running') {
				isStreaming = true;
				stopFn = streamTask(
					taskId,
					(event) => {
						events = [...events, event];
						if (event.type === 'done' || event.type === 'error') {
							isStreaming = false;
						}
						if (event.status) {
							task = task ? { ...task, status: event.status } : task;
						}
					},
					() => { isStreaming = false; }
				);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load task';
		} finally {
			loading = false;
		}
	});

	onDestroy(() => {
		stopFn?.();
	});

	async function handleCancel() {
		stopFn?.();
		try {
			await cancelTask(taskId);
			if (task) task = { ...task, status: 'cancelled' };
		} catch {
			// ignore
		}
		isStreaming = false;
	}
</script>

<div class="page">
	{#if loading}
		<p class="text-muted">Loading...</p>
	{:else if error}
		<p class="text-error">Error: {error}</p>
	{:else if task}
		<div class="page__header">
			<div>
				<div class="page__title-row">
					<h1>Task Detail</h1>
					<StatusBadge status={task.status} />
				</div>
				<p class="page__id">{task.id}</p>
			</div>
			{#if task.status === 'running'}
				<button class="btn-danger" onclick={handleCancel}>Cancel</button>
			{/if}
		</div>

		<div class="detail-card">
			<div class="detail-row">
				<span class="detail-row__label">Prompt</span>
				<span class="detail-row__value">{task.prompt}</span>
			</div>
			<div class="detail-row">
				<span class="detail-row__label">Provider</span>
				<span class="detail-row__value"><ProviderBadge name={task.provider} /></span>
			</div>
			<div class="detail-row">
				<span class="detail-row__label">Model</span>
				<span class="detail-row__value detail-row__mono">{task.model || '-'}</span>
			</div>
			<div class="detail-row">
				<span class="detail-row__label">Cost</span>
				<span class="detail-row__value">${task.cost.toFixed(4)}</span>
			</div>
			<div class="detail-row">
				<span class="detail-row__label">Tokens</span>
				<span class="detail-row__value">{task.tokens_in} in / {task.tokens_out} out</span>
			</div>
			<div class="detail-row">
				<span class="detail-row__label">Created</span>
				<span class="detail-row__value">{format(new Date(task.created_at), 'MMM d, yyyy HH:mm:ss')}</span>
			</div>
			{#if task.error}
				<div class="detail-row detail-row--error">
					<span class="detail-row__label">Error</span>
					<span class="detail-row__value">{task.error}</span>
				</div>
			{/if}
		</div>

		{#if task.result}
			<h2 class="section-title">Result</h2>
			<div class="result-card">
				<pre>{task.result}</pre>
			</div>
		{/if}

		<TaskStream {events} isRunning={isStreaming} />
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			@include flex-between;
			margin-bottom: $space-8;
		}

		&__title-row {
			display: flex;
			align-items: center;
			gap: $space-3;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
		}

		&__id {
			font-family: $font-mono;
			font-size: $text-xs;
			color: $neutral-400;
			margin-top: $space-1;
		}
	}

	.text-muted { color: $neutral-500; }
	.text-error { color: $error-600; }

	.detail-card {
		@include card;
		margin-bottom: $space-6;
	}

	.detail-row {
		@include flex-between;
		padding: $space-3 $space-5;
		font-size: $text-sm;

		& + & { border-top: 1px solid $neutral-100; }

		&__label { color: $neutral-500; font-weight: $font-medium; }
		&__value { color: $neutral-900; }
		&__mono { font-family: $font-mono; font-size: $text-xs; }

		&--error &__value { color: $error-600; }
	}

	.section-title { margin-bottom: $space-4; }

	.result-card {
		@include card;
		padding: $space-4;
		margin-bottom: $space-6;

		pre {
			font-family: $font-mono;
			font-size: $text-sm;
			color: $neutral-800;
			white-space: pre-wrap;
			word-break: break-word;
		}
	}
</style>
```

- [ ] **Step 2: Write `workflows/+page.svelte`**

```svelte
<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>Workflows</h1>
			<p>Create and manage reusable automation workflows</p>
		</div>
		<button class="btn-primary">New Workflow</button>
	</div>

	<EmptyState
		title="No workflows yet"
		description="Create a workflow to automate repetitive tasks with reusable templates."
	>
		{#snippet action()}
			<button class="btn-primary">Create Workflow</button>
		{/snippet}
	</EmptyState>
</div>

<style lang="scss">
	.page {
		&__header {
			@include flex-between;
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
```

- [ ] **Step 3: Write `team/+page.svelte`**

```svelte
<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Team</h1>
		<p>View team members, shared tasks, and workflows</p>
	</div>

	<EmptyState
		title="No team assigned"
		description="You'll see team members and shared resources here once you're assigned to a team."
	/>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
```

- [ ] **Step 4: Write `settings/+page.svelte`**

```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import { isTauri } from '$lib/platform';

	let activeTab = $state<'general' | 'providers' | 'connection'>('general');
	let providers = $state<Provider[]>([]);
	let loading = $state(true);
	let backendUrl = $state('');
	let showTauriSettings = $state(false);

	onMount(async () => {
		showTauriSettings = isTauri();
		if (showTauriSettings) {
			backendUrl = localStorage.getItem('forgebox_api_url') || 'http://localhost:8420/api/v1';
		}
		try {
			providers = await listProviders();
		} catch {
			// ignore
		} finally {
			loading = false;
		}
	});

	function saveBackendUrl() {
		localStorage.setItem('forgebox_api_url', backendUrl);
	}

	const tabs = $derived([
		{ key: 'general' as const, label: 'General' },
		{ key: 'providers' as const, label: 'Providers' },
		...(showTauriSettings ? [{ key: 'connection' as const, label: 'Connection' }] : [])
	]);
</script>

<div class="page">
	<div class="page__header">
		<h1>Settings</h1>
	</div>

	<div class="tabs">
		{#each tabs as tab}
			<button
				class="tabs__tab"
				class:tabs__tab--active={activeTab === tab.key}
				onclick={() => activeTab = tab.key}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	{#if activeTab === 'general'}
		<div class="section">
			<h2>Preferences</h2>
			<div class="settings-card">
				<div class="settings-row">
					<span class="settings-row__label">Theme</span>
					<span class="settings-row__value">System default</span>
				</div>
				<div class="settings-row">
					<span class="settings-row__label">Notifications</span>
					<span class="settings-row__value">Enabled</span>
				</div>
			</div>
		</div>
	{:else if activeTab === 'providers'}
		<div class="section">
			<h2>Available Providers</h2>
			{#if loading}
				<p class="text-muted">Loading...</p>
			{:else if providers.length === 0}
				<p class="text-muted">No providers configured.</p>
			{:else}
				<div class="provider-grid">
					{#each providers as p}
						<div class="provider-item">
							<p class="provider-item__name">{p.name}</p>
							<p class="provider-item__meta">v{p.version} {p.builtin ? '(built-in)' : ''}</p>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{:else if activeTab === 'connection'}
		<div class="section">
			<h2>Backend Connection</h2>
			<div class="settings-card">
				<div class="connection-form">
					<label class="connection-field">
						<span>Backend URL</span>
						<input type="url" bind:value={backendUrl} placeholder="http://localhost:8420/api/v1" />
					</label>
					<button class="btn-primary" onclick={saveBackendUrl}>Save</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
		}
	}

	.tabs {
		display: flex;
		gap: $space-1;
		padding: $space-1;
		background: $neutral-100;
		border-radius: $radius-lg;
		margin-bottom: $space-6;

		&__tab {
			@include btn;
			color: $neutral-500;
			background: transparent;

			&--active {
				background: $neutral-0;
				color: $neutral-900;
				box-shadow: $shadow-sm;
			}

			&:hover:not(&--active) { color: $neutral-700; }
		}
	}

	.text-muted { color: $neutral-500; font-size: $text-sm; }

	.section {
		h2 { margin-bottom: $space-4; }
	}

	.settings-card { @include card; }

	.settings-row {
		@include flex-between;
		padding: $space-3 $space-5;
		font-size: $text-sm;

		& + & { border-top: 1px solid $neutral-100; }

		&__label { color: $neutral-600; }
		&__value { font-weight: $font-medium; color: $neutral-900; }
	}

	.provider-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
		gap: $space-3;
	}

	.provider-item {
		@include card;
		padding: $space-4;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; margin-top: $space-1; }
	}

	.connection-form {
		padding: $space-5;
		display: flex;
		align-items: flex-end;
		gap: $space-3;
	}

	.connection-field {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: $space-1;

		span { font-size: $text-sm; font-weight: $font-medium; color: $neutral-700; }
		input { @include input-base; }
	}
</style>
```

- [ ] **Step 5: Commit**

```bash
git add web/src/routes/\(user\)/
git commit -m "feat(web): add user pages — task detail, workflows, team, settings"
```

---

### Task 13: Tauri v2 Configuration

Set up Tauri v2 config files for desktop builds.

**Files:**
- Create: `web/src-tauri/tauri.conf.json`
- Create: `web/src-tauri/Cargo.toml`
- Create: `web/src-tauri/src/main.rs`

- [ ] **Step 1: Write `tauri.conf.json`**

```json
{
	"$schema": "https://raw.githubusercontent.com/niclas/tauri-settings-schema/refs/heads/v2/schemas/tauri.schema.json",
	"productName": "ForgeBox",
	"version": "0.1.0",
	"identifier": "com.forgebox.dashboard",
	"build": {
		"frontendDist": "../build",
		"devUrl": "http://localhost:3000",
		"beforeDevCommand": "npm run dev",
		"beforeBuildCommand": "npm run build"
	},
	"app": {
		"title": "ForgeBox",
		"windows": [
			{
				"title": "ForgeBox",
				"width": 1280,
				"height": 800,
				"resizable": true,
				"fullscreen": false
			}
		],
		"security": {
			"csp": null
		}
	},
	"bundle": {
		"active": true,
		"targets": "all",
		"icon": [
			"icons/32x32.png",
			"icons/128x128.png",
			"icons/128x128@2x.png",
			"icons/icon.icns",
			"icons/icon.ico"
		]
	}
}
```

- [ ] **Step 2: Write `Cargo.toml`**

```toml
[package]
name = "forgebox-dashboard"
version = "0.1.0"
description = "ForgeBox Dashboard"
edition = "2021"

[lib]
name = "forgebox_dashboard_lib"
crate-type = ["lib", "cdylib", "staticlib"]

[build-dependencies]
tauri-build = { version = "2", features = [] }

[dependencies]
tauri = { version = "2", features = [] }
tauri-plugin-store = "2"
tauri-plugin-shell = "2"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
```

- [ ] **Step 3: Write `src/main.rs`**

```rust
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::new().build())
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

- [ ] **Step 4: Create `src-tauri/build.rs`**

```rust
fn main() {
    tauri_build::build()
}
```

- [ ] **Step 5: Commit**

```bash
git add web/src-tauri/
git commit -m "feat(web): add Tauri v2 configuration for desktop builds"
```

---

### Task 14: Clean Up & Verify Build

Remove leftover React files, delete old `src/lib/api.ts`, verify the SvelteKit build works.

**Files:**
- Delete: `web/src/lib/api.ts` (replaced by `web/src/lib/api/client.ts`)
- Delete: `web/src/lib/types.ts` (replaced by `web/src/lib/api/types.ts`)

- [ ] **Step 1: Remove leftover files**

```bash
cd web
rm -f src/lib/api.ts src/lib/types.ts
```

- [ ] **Step 2: Create `static/favicon.png` placeholder**

```bash
cd web && mkdir -p static
# Create a minimal 1x1 PNG placeholder
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82' > static/favicon.png
```

- [ ] **Step 3: Run `npm install` and `npm run build`**

```bash
cd web && npm install && npm run build
```

Expected: Clean build to `build/` directory with no errors.

- [ ] **Step 4: Run typecheck**

```bash
cd web && npm run typecheck
```

Expected: No type errors.

- [ ] **Step 5: Commit**

```bash
git add -A web/
git commit -m "feat(web): clean up leftover React files, verify SvelteKit build"
```
