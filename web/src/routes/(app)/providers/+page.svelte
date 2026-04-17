<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import EmptyState from '$lib/components/EmptyState.svelte';

	let providers = $state<Provider[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			providers = await listProviders();
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
			<h1>Providers</h1>
			<p>Configure LLM providers</p>
		</div>
		<a href="/providers/new" class="btn-secondary">Add Provider</a>
	</div>

	{#if loading}
		<p class="text-muted">Loading...</p>
	{:else if error}
		<p class="text-error">Error: {error}</p>
	{:else if providers.length === 0}
		<EmptyState
			title="No providers configured"
			description="Add an LLM provider to get started."
		>
			{#snippet action()}
				<a href="/providers/new" class="btn-primary">Add Provider</a>
			{/snippet}
		</EmptyState>
	{:else}
		<div class="grid">
			{#each providers as provider}
				<div class="provider-card">
					<div>
						<p class="provider-card__name">{provider.name}</p>
						<p class="provider-card__meta">v{provider.version} {provider.builtin ? '(built-in)' : ''}</p>
					</div>
					<span class="provider-card__badge">Active</span>
				</div>
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

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: $space-4;
	}

	.provider-card {
		@include card;
		@include flex-between;
		padding: $space-5;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; }

		&__badge {
			@include badge;
			background: $success-100;
			color: $success-700;
		}
	}
</style>
