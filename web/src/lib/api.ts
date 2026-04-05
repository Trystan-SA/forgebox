// API client for the ForgeBox gateway.

import type {
  Task,
  Session,
  Message,
  Provider,
  ToolSchema,
  AuditEntry,
  CreateTaskRequest,
  TaskEvent,
} from "./types";

const BASE = "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

// --- Tasks ---

export async function createTask(
  req: CreateTaskRequest,
): Promise<{ task_id: string; status: string }> {
  return request("/tasks", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function getTask(id: string): Promise<Task> {
  return request(`/tasks/${id}`);
}

export async function listTasks(): Promise<Task[]> {
  return request("/tasks");
}

export async function cancelTask(
  id: string,
): Promise<{ task_id: string; status: string }> {
  return request(`/tasks/${id}`, { method: "DELETE" });
}

// Subscribe to real-time task events via SSE.
export function streamTask(
  id: string,
  onEvent: (event: TaskEvent) => void,
  onError?: (error: Error) => void,
): () => void {
  const source = new EventSource(`${BASE}/tasks/${id}/stream`);

  source.onmessage = (e) => {
    try {
      const event: TaskEvent = JSON.parse(e.data);
      onEvent(event);
    } catch {
      // Ignore parse errors for heartbeat/keepalive messages.
    }
  };

  source.onerror = () => {
    onError?.(new Error("SSE connection lost"));
    source.close();
  };

  return () => source.close();
}

// --- Sessions ---

export async function listSessions(): Promise<Session[]> {
  return request("/sessions");
}

export async function getSession(id: string): Promise<Session> {
  return request(`/sessions/${id}`);
}

export async function sendMessage(
  sessionId: string,
  text: string,
): Promise<{ status: string }> {
  return request(`/sessions/${sessionId}/message`, {
    method: "POST",
    body: JSON.stringify({ text }),
  });
}

// --- Discovery ---

export async function listProviders(): Promise<Provider[]> {
  return request("/providers");
}

export async function listTools(): Promise<ToolSchema[]> {
  return request("/tools");
}

// --- Audit (future endpoint) ---

export async function listAuditEntries(): Promise<AuditEntry[]> {
  // TODO: Wire up when backend audit endpoint is added.
  return [];
}
