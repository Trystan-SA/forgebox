import type { BrainFile, BrainGraph, DreamProposal, BrainFileWithMeta } from '$lib/api/types';
import * as api from '$lib/api/brain';

export let files = $state<BrainFile[]>([]);
export let graph = $state<BrainGraph | null>(null);
export let selectedFileId = $state<string | null>(null);
export let selectedFile = $state<BrainFile | null>(null);
export let searchResults = $state<BrainFileWithMeta[]>([]);
export let dreamProposals = $state<DreamProposal[]>([]);
export let loading = $state(false);

let currentAgentId = '';

export async function loadBrain(agentId: string) {
	currentAgentId = agentId;
	loading = true;
	try {
		const [fileList, graphData, dreams] = await Promise.all([
			api.listBrainFiles(agentId),
			api.getBrainGraph(agentId),
			api.listDreamProposals(agentId)
		]);
		files = fileList;
		graph = graphData;
		dreamProposals = dreams;
	} finally {
		loading = false;
	}
}

export async function selectFile(fileId: string) {
	selectedFileId = fileId;
	selectedFile = await api.getBrainFile(currentAgentId, fileId);
}

export function clearSelection() {
	selectedFileId = null;
	selectedFile = null;
}

export async function createFile(title: string, content: string) {
	const file = await api.createBrainFile(currentAgentId, title, content);
	files = [...files, file];
	selectedFileId = file.id;
	selectedFile = file;
	return file;
}

export async function updateFile(fileId: string, title: string, content: string) {
	const updated = await api.updateBrainFile(currentAgentId, fileId, { title, content });
	files = files.map((f) => (f.id === fileId ? updated : f));
	if (selectedFileId === fileId) {
		selectedFile = updated;
	}
	return updated;
}

export async function deleteFile(fileId: string) {
	await api.deleteBrainFile(currentAgentId, fileId);
	files = files.filter((f) => f.id !== fileId);
	if (selectedFileId === fileId) {
		clearSelection();
	}
}

export async function search(query: string) {
	if (!query.trim()) {
		searchResults = [];
		return;
	}
	searchResults = await api.searchBrainFiles(currentAgentId, query);
}

export async function approveDream(dreamId: string) {
	await api.approveDream(currentAgentId, dreamId);
	dreamProposals = dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'approved' as const } : d
	);
	await loadBrain(currentAgentId);
}

export async function rejectDream(dreamId: string) {
	await api.rejectDream(currentAgentId, dreamId);
	dreamProposals = dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'rejected' as const } : d
	);
}
