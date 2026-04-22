import type { BrainFile, BrainGraph, DreamProposal, BrainFileWithMeta } from '$lib/api/types';
import * as api from '$lib/api/brain';

interface BrainState {
	files: BrainFile[];
	graph: BrainGraph | null;
	selectedFileId: string | null;
	selectedFile: BrainFile | null;
	searchResults: BrainFileWithMeta[];
	dreamProposals: DreamProposal[];
	loading: boolean;
}

export const state = $state<BrainState>({
	files: [],
	graph: null,
	selectedFileId: null,
	selectedFile: null,
	searchResults: [],
	dreamProposals: [],
	loading: false
});

let currentAgentId = '';

export async function loadBrain(agentId: string) {
	currentAgentId = agentId;
	state.loading = true;
	try {
		const [fileList, graphData, dreams] = await Promise.all([
			api.listBrainFiles(agentId),
			api.getBrainGraph(agentId),
			api.listDreamProposals(agentId)
		]);
		state.files = fileList;
		state.graph = graphData;
		state.dreamProposals = dreams;
	} finally {
		state.loading = false;
	}
}

export async function selectFile(fileId: string) {
	state.selectedFileId = fileId;
	state.selectedFile = await api.getBrainFile(currentAgentId, fileId);
}

export function clearSelection() {
	state.selectedFileId = null;
	state.selectedFile = null;
}

export async function createFile(title: string, content: string) {
	const file = await api.createBrainFile(currentAgentId, title, content);
	state.files = [...state.files, file];
	state.selectedFileId = file.id;
	state.selectedFile = file;
	return file;
}

export async function updateFile(fileId: string, title: string, content: string) {
	const updated = await api.updateBrainFile(currentAgentId, fileId, { title, content });
	state.files = state.files.map((f) => (f.id === fileId ? updated : f));
	if (state.selectedFileId === fileId) {
		state.selectedFile = updated;
	}
	return updated;
}

export async function deleteFile(fileId: string) {
	await api.deleteBrainFile(currentAgentId, fileId);
	state.files = state.files.filter((f) => f.id !== fileId);
	if (state.selectedFileId === fileId) {
		clearSelection();
	}
}

export async function search(query: string) {
	if (!query.trim()) {
		state.searchResults = [];
		return;
	}
	state.searchResults = await api.searchBrainFiles(currentAgentId, query);
}

export async function approveDream(dreamId: string) {
	await api.approveDream(currentAgentId, dreamId);
	state.dreamProposals = state.dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'approved' as const } : d
	);
	await loadBrain(currentAgentId);
}

export async function rejectDream(dreamId: string) {
	await api.rejectDream(currentAgentId, dreamId);
	state.dreamProposals = state.dreamProposals.map((d) =>
		d.id === dreamId ? { ...d, status: 'rejected' as const } : d
	);
}
