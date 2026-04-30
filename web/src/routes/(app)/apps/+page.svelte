<script lang="ts">
	import { onMount } from 'svelte';
	import type { App } from '$lib/api/types';
	import { listApps, deleteApp } from '$lib/api/client';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { formatDistanceToNow } from 'date-fns';

	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			apps = await listApps();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load apps';
		} finally {
			loading = false;
		}
	});

	async function handleDelete(id: string) {
		if (!confirm('Delete this app?')) return;
		try {
			await deleteApp(id);
			apps = apps.filter((a) => a.id !== id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete app';
		}
	}

	function sharingLabel(s: string) {
		if (s === 'org') return 'Organization';
		if (s === 'team') return 'Team';
		return 'Personal';
	}

	function statusColor(s: string) {
		if (s === 'running') return 'running';
		if (s === 'deploying') return 'deploying';
		if (s === 'error') return 'error';
		if (s === 'stopped') return 'stopped';
		return 'draft';
	}

	function statusLabel(s: string) {
		if (s === 'running') return 'Running';
		if (s === 'deploying') return 'Deploying';
		if (s === 'error') return 'Error';
		if (s === 'stopped') return 'Stopped';
		return 'Draft';
	}

	function parseTools(tools: string): string[] {
		try { return JSON.parse(tools); }
		catch { return []; }
	}

	function toolLabel(t: string) {
		if (t === 'database') return 'Database';
		if (t === 'api') return 'API';
		if (t === 'ai') return 'AI';
		return t;
	}
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>Apps</h1>
			<p>Build internal tools powered by AI, running in isolated VMs</p>
		</div>
		<a href="/apps/new" class="btn-primary">New App</a>
	</div>

	{#if error}
		<div class="page__error">{error}</div>
	{/if}

	{#if loading}
		<p class="page__loading">Loading...</p>
	{:else if apps.length === 0}
		<EmptyState
			title="No apps yet"
			description="Apps are internal tools built with AI. They run in isolated VMs with access to databases, APIs, and AI models."
		>
			{#snippet action()}
				<a href="/apps/new" class="btn-primary">Create App</a>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="grid">
			{#each apps as app}
				<a href="/apps/{app.id}" class="card">
					<div class="card__top">
						<div class="card__title-row">
							<h3 class="card__name">{app.name}</h3>
							<span class="card__status card__status--{statusColor(app.status)}">{statusLabel(app.status)}</span>
						</div>
						{#if app.description}
							<p class="card__desc">{app.description}</p>
						{/if}
						<div class="card__meta-row">
							<span class="card__badge card__badge--{app.sharing}">{sharingLabel(app.sharing)}</span>
							{#each parseTools(app.tools) as tool}
								<span class="card__chip">{toolLabel(tool)}</span>
							{/each}
						</div>
					</div>
					<div class="card__bottom">
						<span class="card__meta">
							Updated {formatDistanceToNow(new Date(app.updated_at))} ago
						</span>
						<button
							class="card__delete"
							onclick={(e) => { e.preventDefault(); handleDelete(app.id); }}
						>
							Delete
						</button>
					</div>
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

		&__loading { color: $neutral-500; }

		&__error {
			padding: $space-3;
			margin-bottom: $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: $space-4;
	}

	.card {
		@include card;
		padding: $space-5;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: $space-4;
		text-decoration: none;
		transition: border-color $transition-fast;

		&:hover { border-color: $primary-300; }

		&__top { display: flex; flex-direction: column; gap: $space-2; }

		&__title-row {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: $space-2;
		}

		&__name {
			font-size: $text-base;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__desc {
			font-size: $text-sm;
			color: $neutral-500;
			line-height: $leading-relaxed;
		}

		&__meta-row {
			display: flex;
			gap: $space-2;
			flex-wrap: wrap;
		}

		&__chip {
			font-family: $font-mono;
			font-size: 10px;
			color: $neutral-600;
			background: $neutral-100;
			padding: 2px 6px;
			border-radius: $radius-sm;
		}

		&__badge {
			@include badge;
			&--personal { background: $neutral-100; color: $neutral-600; }
			&--team { background: $info-50; color: $info-600; }
			&--org { background: $primary-50; color: $primary-700; }
		}

		&__status {
			@include badge;
			&--running { background: $success-50; color: $success-700; }
			&--deploying { background: $warning-50; color: $warning-700; }
			&--error { background: $error-50; color: $error-700; }
			&--stopped { background: $neutral-100; color: $neutral-500; }
			&--draft { background: $neutral-100; color: $neutral-400; }
		}

		&__bottom {
			display: flex;
			align-items: center;
			gap: $space-3;
			border-top: 1px solid $neutral-100;
			padding-top: $space-3;
		}

		&__meta {
			font-size: $text-xs;
			color: $neutral-400;
		}

		&__delete {
			margin-left: auto;
			font-size: $text-xs;
			color: $neutral-400;
			background: none;
			border: none;
			cursor: pointer;
			padding: $space-1 $space-2;
			border-radius: $radius-md;
			transition: all $transition-fast;

			&:hover { color: $error-600; background: $error-50; }
		}
	}
</style>
