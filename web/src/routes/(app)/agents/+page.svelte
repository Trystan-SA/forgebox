<script lang="ts">
	import { onMount } from 'svelte';
	import type { Agent } from '$lib/api/types';
	import { listAgents, deleteAgent } from '$lib/api/client';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { formatDistanceToNow } from 'date-fns';

	let agents = $state<Agent[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			agents = await listAgents();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load agents';
		} finally {
			loading = false;
		}
	});

	async function handleDelete(id: string) {
		if (!confirm('Delete this agent?')) return;
		try {
			await deleteAgent(id);
			agents = agents.filter((a) => a.id !== id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete agent';
		}
	}

	function toolCount(toolsJSON: string): number {
		try {
			const parsed = JSON.parse(toolsJSON || '[]');
			return Array.isArray(parsed) ? parsed.length : 0;
		} catch {
			return 0;
		}
	}

	function sharingLabel(s: string) {
		if (s === 'org') return 'Organization';
		if (s === 'team') return 'Team';
		return 'Personal';
	}
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>Agents</h1>
			<p>Create and manage AI agents for your team</p>
		</div>
		<a href="/agents/new" class="btn-primary">New Agent</a>
	</div>

	{#if loading}
		<p class="page__loading">Loading...</p>
	{:else if error}
		<p class="page__error">{error}</p>
	{:else if agents.length === 0}
		<EmptyState
			title="No agents yet"
			description="Agents are autonomous AI assistants configured with custom instructions, tools, and permissions."
		>
			{#snippet action()}
				<a href="/agents/new" class="btn-primary">Create Agent</a>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="grid">
			{#each agents as agent}
				{@const tc = toolCount(agent.tools)}
				<a href="/agents/{agent.id}" class="card">
					<div class="card__top">
						<div class="card__title-row">
							<h3 class="card__name">{agent.name}</h3>
							<span class="card__badge card__badge--{agent.sharing}">{sharingLabel(agent.sharing)}</span>
						</div>
						{#if agent.description}
							<p class="card__desc">{agent.description}</p>
						{/if}
						<div class="card__meta-row">
							{#if agent.provider}
								<span class="card__chip">{agent.provider}</span>
							{/if}
							{#if agent.model}
								<span class="card__chip">{agent.model}</span>
							{/if}
							{#if tc > 0}
								<span class="card__chip">{tc} tool{tc > 1 ? 's' : ''}</span>
							{/if}
						</div>
					</div>
					<div class="card__bottom">
						<span class="card__meta">
							Updated {formatDistanceToNow(new Date(agent.updated_at))} ago
						</span>
						<button
							class="card__delete"
							onclick={(e) => { e.preventDefault(); handleDelete(agent.id); }}
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
		&__error { color: $error-600; }
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
