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
