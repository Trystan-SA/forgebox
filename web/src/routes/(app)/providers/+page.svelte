<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders, deleteProvider } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import { pushToast } from '$lib/stores/toasts.svelte';

	let providers = $state<Provider[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let deleting = $state<string | null>(null);

	onMount(async () => {
		try {
			providers = await listProviders();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});

	async function handleDelete(p: Provider) {
		if (!p.id) return;
		if (!confirm(`Remove provider "${p.name}"? Anything using it will stop working.`)) return;
		deleting = p.id;
		try {
			await deleteProvider(p.id);
			providers = providers.filter((x) => x.id !== p.id);
			pushToast('Provider removed', 'success');
		} catch (err) {
			pushToast(err instanceof Error ? err.message : 'Failed to remove', 'error', 5000);
		} finally {
			deleting = null;
		}
	}
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
					<div class="provider-card__info">
						<p class="provider-card__name">{provider.name}</p>
						<p class="provider-card__meta">
							{provider.provider_type ?? provider.name}
							· v{provider.version}
							{provider.builtin ? '· built-in' : ''}
						</p>
					</div>
					<div class="provider-card__actions">
						<span class="provider-card__badge">Active</span>
						{#if !provider.builtin && provider.id}
							<button
								type="button"
								class="provider-card__remove"
								onclick={() => handleDelete(provider)}
								disabled={deleting === provider.id}
								aria-label="Remove provider"
							>
								{deleting === provider.id ? '…' : 'Remove'}
							</button>
						{/if}
					</div>
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
		gap: $space-3;

		&__info { min-width: 0; }
		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta {
			font-size: $text-xs;
			color: $neutral-500;
			margin-top: 2px;
		}

		&__actions {
			display: flex;
			align-items: center;
			gap: $space-2;
			flex-shrink: 0;
		}

		&__badge {
			@include badge;
			background: $success-100;
			color: $success-700;
		}

		&__remove {
			padding: 4px $space-2;
			font-size: 11px;
			font-weight: $font-medium;
			color: $error-600;
			background: transparent;
			border: 1px solid transparent;
			border-radius: $radius-md;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover:not(:disabled) {
				background: $error-50;
				border-color: $error-100;
			}

			&:disabled { opacity: 0.5; cursor: not-allowed; }
		}
	}
</style>
