<script lang="ts">
	import { onMount } from 'svelte';
	import { listTasks } from '$lib/api/client';
	import type { Task } from '$lib/api/types';
	import { currentUser } from '$lib/stores/auth';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let tasks = $state<Task[]>([]);
	let loading = $state(true);

	const recent = $derived(tasks.slice(0, 5));
	const running = $derived(tasks.filter((t) => t.status === 'running').length);

	onMount(async () => {
		try {
			tasks = await listTasks();
		} catch {
			// silently fail on home page
		} finally {
			loading = false;
		}
	});
</script>

<div class="page">
	<div class="page__header">
		<h1>Welcome{$currentUser ? `, ${$currentUser.name || $currentUser.email}` : ''}</h1>
		<p>Your personal dashboard</p>
	</div>

	<div class="quick-stats">
		<div class="quick-stat">
			<p class="quick-stat__value">{tasks.length}</p>
			<p class="quick-stat__label">Total Tasks</p>
		</div>
		<div class="quick-stat">
			<p class="quick-stat__value">{running}</p>
			<p class="quick-stat__label">Running</p>
		</div>
	</div>

	<div class="actions">
		<a href="/tasks/new" class="action-card action-card--primary">
			<span class="action-card__icon">▶</span>
			<div>
				<p class="action-card__title">Run a Task</p>
				<p class="action-card__desc">Execute an AI task in a secure VM</p>
			</div>
		</a>
		<a href="/tasks" class="action-card">
			<span class="action-card__icon">📋</span>
			<div>
				<p class="action-card__title">My Tasks</p>
				<p class="action-card__desc">View task history and results</p>
			</div>
		</a>
		<a href="/workflows" class="action-card">
			<span class="action-card__icon">🔄</span>
			<div>
				<p class="action-card__title">Workflows</p>
				<p class="action-card__desc">Create reusable automations</p>
			</div>
		</a>
	</div>

	{#if !loading && recent.length > 0}
		<h2 class="section-title">Recent Tasks</h2>
		<div class="recent-list">
			{#each recent as task}
				<a href="/tasks/{task.id}" class="recent-item">
					<div class="recent-item__content">
						<p class="recent-item__prompt">{task.prompt}</p>
						<p class="recent-item__meta">{task.provider} &middot; {new Date(task.created_at).toLocaleDateString()}</p>
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
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.quick-stats {
		display: flex;
		gap: $space-4;
		margin-bottom: $space-8;
	}

	.quick-stat {
		@include card;
		padding: $space-5;
		min-width: 150px;

		&__value { font-size: $text-3xl; font-weight: $font-bold; color: $neutral-900; }
		&__label { font-size: $text-sm; color: $neutral-500; }
	}

	.actions {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: $space-4;
		margin-bottom: $space-8;
	}

	.action-card {
		@include card;
		display: flex;
		align-items: center;
		gap: $space-4;
		padding: $space-5;
		transition: border-color $transition-fast, box-shadow $transition-fast;

		&:hover { border-color: $primary-300; box-shadow: $shadow-md; }

		&--primary { border-left: 3px solid $primary-500; }

		&__icon { font-size: $text-2xl; }
		&__title { font-weight: $font-medium; color: $neutral-900; }
		&__desc { font-size: $text-sm; color: $neutral-500; }
	}

	.section-title { margin-bottom: $space-4; }

	.recent-list {
		display: flex;
		flex-direction: column;
		gap: $space-2;
	}

	.recent-item {
		@include card;
		@include flex-between;
		padding: $space-4;
		transition: border-color $transition-fast;

		&:hover { border-color: $neutral-300; }

		&__prompt { @include truncate; max-width: 500px; font-size: $text-sm; font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; margin-top: $space-1; }
	}
</style>
