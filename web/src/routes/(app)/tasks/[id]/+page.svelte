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

	const taskId = $derived(page.params.id!);

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
