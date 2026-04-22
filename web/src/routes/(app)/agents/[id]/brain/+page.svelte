<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Spinner from '$lib/components/Spinner.svelte';
	import BrainGraph from '$lib/components/brain/BrainGraph.svelte';
	import BrainEditor from '$lib/components/brain/BrainEditor.svelte';
	import BrainFileMeta from '$lib/components/brain/BrainFileMeta.svelte';
	import BrainSearch from '$lib/components/brain/BrainSearch.svelte';
	import DreamPanel from '$lib/components/brain/DreamPanel.svelte';
	import * as brain from '$lib/stores/brain.svelte';
	import type { BrainFile, BrainFileWithMeta } from '$lib/api/types';

	let agentId = $derived(page.params.id ?? '');

	let searchHighlights = $state<string[]>([]);
	let dreamPanelOpen = $state(false);
	let showNewFileModal = $state(false);
	let newFileTitle = $state('');
	let newFileError = $state('');
	let creatingFile = $state(false);
	let loadError = $state<string | null>(null);

	onMount(async () => {
		try {
			await brain.loadBrain(agentId);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load brain';
		}
	});

	async function handleSave(e: CustomEvent<{ title: string; content: string }>) {
		if (!brain.state.selectedFileId) return;
		try {
			await brain.updateFile(brain.state.selectedFileId, e.detail.title, e.detail.content);
		} catch (err) {
			console.error('Save failed:', err);
		}
	}

	async function handleDelete() {
		if (!brain.state.selectedFileId) return;
		try {
			await brain.deleteFile(brain.state.selectedFileId);
		} catch (err) {
			console.error('Delete failed:', err);
		}
	}

	async function handleTitleChange(e: CustomEvent<string>) {
		if (!brain.state.selectedFileId || !brain.state.selectedFile) return;
		try {
			await brain.updateFile(brain.state.selectedFileId, e.detail, brain.state.selectedFile.content ?? '');
		} catch (err) {
			console.error('Title update failed:', err);
		}
	}

	function handleGraphSelect(e: CustomEvent<{ file_id: string }>) {
		brain.selectFile(e.detail.file_id);
	}

	function handleSearchHighlight(e: CustomEvent<{ fileIds: string[] }>) {
		searchHighlights = e.detail.fileIds;
	}

	async function handleApproveDream(e: CustomEvent<{ id: string }>) {
		try {
			await brain.approveDream(e.detail.id);
		} catch (err) {
			console.error('Approve dream failed:', err);
		}
	}

	async function handleRejectDream(e: CustomEvent<{ id: string }>) {
		try {
			await brain.rejectDream(e.detail.id);
		} catch (err) {
			console.error('Reject dream failed:', err);
		}
	}

	async function submitNewFile() {
		const title = newFileTitle.trim();
		if (!title) { newFileError = 'Title is required'; return; }
		creatingFile = true;
		newFileError = '';
		try {
			await brain.createFile(title, '');
			showNewFileModal = false;
			newFileTitle = '';
		} catch (err) {
			newFileError = err instanceof Error ? err.message : 'Failed to create file';
		} finally {
			creatingFile = false;
		}
	}

	function openNewFile() {
		newFileTitle = '';
		newFileError = '';
		showNewFileModal = true;
	}

	function closeNewFile() {
		showNewFileModal = false;
	}

	function handleModalKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') closeNewFile();
		if (e.key === 'Enter') submitNewFile();
	}

	const pendingDreamCount = $derived(
		brain.state.dreamProposals.filter((d) => d.status === 'pending').length
	);

	const allHashtags = $derived(() => {
		const set = new Set<string>();
		(brain.state.files as BrainFileWithMeta[]).forEach((f) => {
			if ('hashtags' in f && Array.isArray((f as BrainFileWithMeta).hashtags)) {
				(f as BrainFileWithMeta).hashtags.forEach((h) => set.add(h));
			}
		});
		return Array.from(set);
	});

	const selectedFileHashtags = $derived(() => {
		if (!brain.state.selectedFile) return [];
		const f = brain.state.files.find((x) => x.id === brain.state.selectedFileId);
		if (!f) return [];
		if ('hashtags' in f && Array.isArray((f as BrainFileWithMeta).hashtags)) {
			return (f as BrainFileWithMeta).hashtags;
		}
		return [];
	});
</script>

<div class="bp">
	<div class="bp__topbar">
		<div class="bp__topbar-left">
			<a href="/agents/{agentId}" class="bp__back">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<polyline points="15 18 9 12 15 6" />
				</svg>
			</a>
			<h1 class="bp__title">Brain</h1>
		</div>

		<div class="bp__topbar-center">
			<BrainSearch on:highlight={handleSearchHighlight} />
		</div>

		<div class="bp__topbar-right">
			<button type="button" class="bp__btn-new" onclick={openNewFile}>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
					<line x1="12" y1="5" x2="12" y2="19" />
					<line x1="5" y1="12" x2="19" y2="12" />
				</svg>
				New File
			</button>

			<button
				type="button"
				class="bp__btn-dreams"
				class:bp__btn-dreams--active={dreamPanelOpen}
				onclick={() => { dreamPanelOpen = !dreamPanelOpen; }}
			>
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z" />
					<path d="M8 14s1.5 2 4 2 4-2 4-2" />
					<line x1="9" y1="9" x2="9.01" y2="9" />
					<line x1="15" y1="9" x2="15.01" y2="9" />
				</svg>
				Dreams
				{#if pendingDreamCount > 0}
					<span class="bp__dream-count">{pendingDreamCount}</span>
				{/if}
			</button>
		</div>
	</div>

	{#if brain.state.loading}
		<div class="bp__loading">
			<Spinner size="md" />
			<span>Loading brain...</span>
		</div>
	{:else if loadError}
		<div class="bp__error">
			<p>{loadError}</p>
			<button type="button" class="bp__btn-retry" onclick={() => brain.loadBrain(agentId)}>Retry</button>
		</div>
	{:else if brain.state.files.length === 0}
		<div class="bp__empty">
			<EmptyState
				title="No brain files yet"
				description="Create your first brain file to start building the agent's knowledge graph."
			>
				{#snippet action()}
					<button type="button" class="bp__btn-new bp__btn-new--lg" onclick={openNewFile}>
						Create your first brain file
					</button>
				{/snippet}
			</EmptyState>
		</div>
	{:else}
		<div class="bp__main">
			<div class="bp__graph-pane">
				<BrainGraph
					graph={brain.state.graph}
					selectedFileId={brain.state.selectedFileId}
					{searchHighlights}
					on:select={handleGraphSelect}
				/>
			</div>

			<div class="bp__editor-pane">
				{#if brain.state.selectedFile}
					<div class="bp__editor-wrap">
						<BrainEditor
							file={brain.state.selectedFile}
							allFiles={brain.state.files}
							allHashtags={allHashtags()}
							on:save={handleSave}
							on:delete={handleDelete}
						/>
						<BrainFileMeta
							file={brain.state.selectedFile}
							hashtags={selectedFileHashtags()}
							on:titleChange={handleTitleChange}
						/>
					</div>
				{:else}
					<div class="bp__select-hint">
						<EmptyState
							title="Select a file"
							description="Click a node in the graph to open a file for editing."
						/>
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<DreamPanel
		proposals={brain.state.dreamProposals}
		open={dreamPanelOpen}
		on:approve={handleApproveDream}
		on:reject={handleRejectDream}
		on:close={() => { dreamPanelOpen = false; }}
	/>

	{#if showNewFileModal}
		<div class="bp__modal-backdrop" onclick={closeNewFile} role="presentation"></div>
		<div class="bp__modal" role="dialog" aria-modal="true" aria-label="New brain file">
			<h3 class="bp__modal-title">New Brain File</h3>
			<label class="bp__modal-label" for="new-file-title">Title</label>
			<input
				id="new-file-title"
				class="bp__modal-input"
				type="text"
				placeholder="e.g. Project Overview"
				bind:value={newFileTitle}
				onkeydown={handleModalKeydown}
				autofocus
			/>
			{#if newFileError}
				<p class="bp__modal-error">{newFileError}</p>
			{/if}
			<div class="bp__modal-actions">
				<button type="button" class="btn-secondary" onclick={closeNewFile} disabled={creatingFile}>
					Cancel
				</button>
				<button type="button" class="btn-primary" onclick={submitNewFile} disabled={creatingFile || !newFileTitle.trim()}>
					{creatingFile ? 'Creating…' : 'Create'}
				</button>
			</div>
		</div>
	{/if}
</div>

<style lang="scss">
	.bp {
		display: flex;
		flex-direction: column;
		height: calc(100vh - #{$topbar-height});
		overflow: hidden;

		&__topbar {
			@include flex-between;
			gap: $space-4;
			padding: $space-3 $space-5;
			background: $neutral-0;
			border-bottom: 1px solid $neutral-200;
			flex-shrink: 0;
		}

		&__topbar-left {
			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__topbar-center {
			flex: 1;
			max-width: 400px;
		}

		&__topbar-right {
			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__back {
			@include flex-center;
			width: 32px;
			height: 32px;
			border-radius: $radius-lg;
			color: $neutral-400;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__title {
			font-size: $text-lg;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__btn-new {
			@include btn;
			gap: $space-1;
			padding: $space-2 $space-3;
			font-size: $text-sm;
			font-weight: $font-medium;
			background: $primary-600;
			color: $neutral-0;
			border-radius: $radius-lg;

			&:hover { background: $primary-700; }

			&--lg {
				padding: $space-3 $space-5;
				font-size: $text-sm;
			}
		}

		&__btn-dreams {
			@include btn;
			gap: $space-1;
			padding: $space-2 $space-3;
			font-size: $text-sm;
			font-weight: $font-medium;
			background: $neutral-100;
			color: $neutral-600;
			border-radius: $radius-lg;
			border: 1px solid $neutral-200;

			&:hover { background: $neutral-200; }

			&--active {
				background: $warning-50;
				color: $warning-700;
				border-color: $warning-100;
			}
		}

		&__dream-count {
			@include badge;
			font-size: 10px;
			background: $warning-500;
			color: $neutral-0;
			padding: 0 5px;
		}

		&__loading {
			@include flex-center;
			gap: $space-3;
			flex: 1;
			color: $neutral-400;
			font-size: $text-sm;
		}

		&__error {
			@include flex-center;
			flex-direction: column;
			gap: $space-3;
			flex: 1;
			color: $error-700;
			font-size: $text-sm;

			p { margin: 0; }
		}

		&__btn-retry {
			@include btn;
			padding: $space-2 $space-4;
			font-size: $text-sm;
			background: $neutral-100;
			color: $neutral-700;
			border-radius: $radius-lg;

			&:hover { background: $neutral-200; }
		}

		&__empty {
			@include flex-center;
			flex: 1;
		}

		&__main {
			flex: 1;
			display: grid;
			grid-template-columns: 60% 40%;
			overflow: hidden;
		}

		&__graph-pane {
			padding: $space-4;
			overflow: hidden;
			border-right: 1px solid $neutral-200;
		}

		&__editor-pane {
			display: flex;
			flex-direction: column;
			overflow: hidden;
			padding: $space-4;
		}

		&__editor-wrap {
			display: flex;
			flex-direction: column;
			height: 100%;
			overflow: hidden;
		}

		&__select-hint {
			@include flex-center;
			height: 100%;
		}

		&__modal-backdrop {
			position: fixed;
			inset: 0;
			background: rgba(0, 0, 0, 0.4);
			z-index: 50;
		}

		&__modal {
			position: fixed;
			top: 50%;
			left: 50%;
			transform: translate(-50%, -50%);
			z-index: 51;
			background: $neutral-0;
			border-radius: $radius-2xl;
			border: 1px solid $neutral-200;
			box-shadow: $shadow-lg;
			padding: $space-6;
			width: 400px;
			max-width: calc(100vw - $space-8);
		}

		&__modal-title {
			font-size: $text-lg;
			font-weight: $font-semibold;
			color: $neutral-900;
			margin-bottom: $space-4;
		}

		&__modal-label {
			display: block;
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
			margin-bottom: $space-1;
		}

		&__modal-input {
			@include input-base;
			margin-bottom: $space-4;
		}

		&__modal-error {
			font-size: $text-xs;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-md;
			padding: $space-2 $space-3;
			margin-bottom: $space-3;
		}

		&__modal-actions {
			display: flex;
			gap: $space-2;
			justify-content: flex-end;
		}
	}

	@media (max-width: 768px) {
		.bp {
			&__main {
				grid-template-columns: 1fr;
				grid-template-rows: 1fr 1fr;
			}

			&__graph-pane {
				border-right: none;
				border-bottom: 1px solid $neutral-200;
			}

			&__topbar-center {
				display: none;
			}
		}
	}
</style>
