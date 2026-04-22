import type {
	Brain,
	BrainFile,
	BrainFileWithMeta,
	BrainGraph,
	DreamProposal
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

	const res = await fetch(`${base}${path}`, { headers, ...init });

	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error || `HTTP ${res.status}`);
	}

	return res.json();
}

// --- Brain ---
export async function getBrain(agentId: string): Promise<Brain> {
	return request(`/agents/${agentId}/brain`);
}

// --- Brain Files ---
export async function listBrainFiles(agentId: string): Promise<BrainFile[]> {
	return (await request<BrainFile[] | null>(`/agents/${agentId}/brain/files`)) ?? [];
}

export async function createBrainFile(
	agentId: string,
	title: string,
	content: string
): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files`, {
		method: 'POST',
		body: JSON.stringify({ title, content })
	});
}

export async function getBrainFile(agentId: string, fileId: string): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files/${fileId}`);
}

export async function updateBrainFile(
	agentId: string,
	fileId: string,
	updates: { title?: string; content?: string }
): Promise<BrainFile> {
	return request(`/agents/${agentId}/brain/files/${fileId}`, {
		method: 'PUT',
		body: JSON.stringify(updates)
	});
}

export async function deleteBrainFile(
	agentId: string,
	fileId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/files/${fileId}`, { method: 'DELETE' });
}

// --- Graph ---
export async function getBrainGraph(agentId: string): Promise<BrainGraph> {
	return request(`/agents/${agentId}/brain/graph`);
}

// --- Search ---
export async function searchBrainFiles(
	agentId: string,
	query: string,
	limit = 10
): Promise<BrainFileWithMeta[]> {
	return (
		(await request<BrainFileWithMeta[] | null>(`/agents/${agentId}/brain/search`, {
			method: 'POST',
			body: JSON.stringify({ query, limit })
		})) ?? []
	);
}

// --- Dreams ---
export async function listDreamProposals(agentId: string): Promise<DreamProposal[]> {
	return (await request<DreamProposal[] | null>(`/agents/${agentId}/brain/dreams`)) ?? [];
}

export async function getDreamProposal(
	agentId: string,
	dreamId: string
): Promise<DreamProposal> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}`);
}

export async function approveDream(
	agentId: string,
	dreamId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}/approve`, { method: 'POST' });
}

export async function rejectDream(
	agentId: string,
	dreamId: string
): Promise<{ status: string }> {
	return request(`/agents/${agentId}/brain/dreams/${dreamId}/reject`, { method: 'POST' });
}
