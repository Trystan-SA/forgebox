import type {
	Task,
	Session,
	Provider,
	ToolSchema,
	AuditEntry,
	CreateTaskRequest,
	TaskEvent,
	LoginRequest,
	LoginResponse,
	SetupStatusResponse,
	SetupRequest,
	SetupResponse,
	Automation,
	CreateAutomationRequest,
	UpdateAutomationRequest
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

// --- Setup ---
export async function checkSetupStatus(): Promise<SetupStatusResponse> {
	return request('/setup/status');
}

export async function setupAccount(req: SetupRequest): Promise<SetupResponse> {
	return request('/setup', {
		method: 'POST',
		body: JSON.stringify(req)
	});
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

// --- Automations ---
export async function listAutomations(): Promise<Automation[]> {
	return (await request<Automation[] | null>('/automations')) ?? [];
}

export async function createAutomation(req: CreateAutomationRequest): Promise<Automation> {
	return request('/automations', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function getAutomation(id: string): Promise<Automation> {
	return request(`/automations/${id}`);
}

export async function updateAutomation(id: string, req: UpdateAutomationRequest): Promise<Automation> {
	return request(`/automations/${id}`, {
		method: 'PUT',
		body: JSON.stringify(req)
	});
}

export async function deleteAutomation(id: string): Promise<{ status: string }> {
	return request(`/automations/${id}`, { method: 'DELETE' });
}

// --- Audit ---
export async function listAuditEntries(): Promise<AuditEntry[]> {
	return (await request<AuditEntry[] | null>('/audit')) ?? [];
}
