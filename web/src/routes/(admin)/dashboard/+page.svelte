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
