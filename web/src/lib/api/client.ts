import type {
	Task,
	Session,
	Provider,
	CreateProviderRequest,
	ToolSchema,
	AuditEntry,
	CreateTaskRequest,
	TaskEvent,
	ToolCall,
	LoginRequest,
	LoginResponse,
	SetupStatusResponse,
	SetupRequest,
	SetupResponse,
	Automation,
	CreateAutomationRequest,
	UpdateAutomationRequest,
	Agent,
	CreateAgentRequest,
	UpdateAgentRequest,
	App,
	CreateAppRequest,
	UpdateAppRequest
} from './types';
import { getBaseUrl } from '$lib/platform';
import { subscribe as subscribeSocket } from '$lib/stores/socket.svelte';

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
	_onError?: (error: Error) => void
): () => void {
	type TokenPayload = { task_id: string; delta: string };
	type StatusPayload = { task_id: string; status?: TaskEvent['status']; error?: string };

	const unsubToken = subscribeSocket('task.token', (raw) => {
		const p = raw as TokenPayload;
		if (p?.task_id !== id) return;
		onEvent({ type: 'text_delta', text: p.delta });
	});

	const unsubStatus = subscribeSocket('task.updated', (raw) => {
		const p = raw as StatusPayload;
		if (p?.task_id !== id) return;
		if (p.error) {
			onEvent({ type: 'error', error: p.error });
		} else if (p.status) {
			onEvent({ type: 'status_update', status: p.status });
		}
	});

	type ApprovalPendingPayload = {
		task_id: string;
		approval_id: string;
		tool_call: ToolCall | null;
	};
	type ApprovalResolvedPayload = {
		task_id: string;
		approval_id: string;
		approved: boolean;
	};

	const unsubPending = subscribeSocket('task.tool_pending_approval', (raw) => {
		const p = raw as ApprovalPendingPayload;
		if (p?.task_id !== id) return;
		onEvent({
			type: 'tool_pending_approval',
			approval_id: p.approval_id,
			tool_call: p.tool_call ?? undefined
		});
	});

	const unsubResolved = subscribeSocket('task.tool_approval_resolved', (raw) => {
		const p = raw as ApprovalResolvedPayload;
		if (p?.task_id !== id) return;
		onEvent({
			type: 'tool_approval_resolved',
			approval_id: p.approval_id,
			approved: p.approved
		});
	});

	return () => {
		unsubToken();
		unsubStatus();
		unsubPending();
		unsubResolved();
	};
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

// --- Providers ---
export async function listProviders(): Promise<Provider[]> {
	return (await request<Provider[] | null>('/providers')) ?? [];
}

export async function createProvider(req: CreateProviderRequest): Promise<Provider> {
	return request('/providers', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function deleteProvider(id: string): Promise<{ id: string; status: string }> {
	return request(`/providers/${id}`, { method: 'DELETE' });
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

export async function getAutomationYaml(id: string): Promise<string> {
	const base = getBaseUrl();
	const token = getToken();
	const res = await fetch(`${base}/automations/${id}/yaml`, {
		headers: {
			...(token ? { Authorization: `Bearer ${token}` } : {})
		}
	});
	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error || `HTTP ${res.status}`);
	}
	return res.text();
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

// --- Agents ---
export async function listAgents(): Promise<Agent[]> {
	return (await request<Agent[] | null>('/agents')) ?? [];
}

export async function createAgent(req: CreateAgentRequest): Promise<Agent> {
	return request('/agents', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function getAgent(id: string): Promise<Agent> {
	return request(`/agents/${id}`);
}

export async function updateAgent(id: string, req: UpdateAgentRequest): Promise<Agent> {
	return request(`/agents/${id}`, {
		method: 'PUT',
		body: JSON.stringify(req)
	});
}

export async function deleteAgent(id: string): Promise<{ status: string }> {
	return request(`/agents/${id}`, { method: 'DELETE' });
}

// --- Apps ---
export async function listApps(): Promise<App[]> {
	return (await request<App[] | null>('/apps')) ?? [];
}

export async function createApp(req: CreateAppRequest): Promise<App> {
	return request('/apps', {
		method: 'POST',
		body: JSON.stringify(req)
	});
}

export async function getApp(id: string): Promise<App> {
	return request(`/apps/${id}`);
}

export async function updateApp(id: string, req: UpdateAppRequest): Promise<App> {
	return request(`/apps/${id}`, {
		method: 'PUT',
		body: JSON.stringify(req)
	});
}

export async function deleteApp(id: string): Promise<{ status: string }> {
	return request(`/apps/${id}`, { method: 'DELETE' });
}

// --- Audit ---
export async function listAuditEntries(): Promise<AuditEntry[]> {
	return (await request<AuditEntry[] | null>('/audit')) ?? [];
}
