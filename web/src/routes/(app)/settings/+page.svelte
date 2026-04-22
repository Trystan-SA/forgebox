<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import { isTauri } from '$lib/platform';

	let activeTab = $state<'general' | 'connection'>('general');
	let providers = $state<Provider[]>([]);
	let loading = $state(true);
	let backendUrl = $state('');
	let showTauriSettings = $state(false);

	onMount(async () => {
		showTauriSettings = isTauri();
		if (showTauriSettings) {
			backendUrl = localStorage.getItem('forgebox_api_url') || 'http://localhost:8420/api/v1';
		}
		try {
			providers = await listProviders();
		} catch {
			// ignore
		} finally {
			loading = false;
		}
	});

	function saveBackendUrl() {
		localStorage.setItem('forgebox_api_url', backendUrl);
	}

	const tabs = $derived([
		{ key: 'general' as const, label: 'General' },
		...(showTauriSettings ? [{ key: 'connection' as const, label: 'Connection' }] : [])
	]);
</script>

<div class="page">
	<div class="page__header">
		<h1>Settings</h1>
	</div>

	<div class="tabs">
		{#each tabs as tab}
			<button
				class="tabs__tab"
				class:tabs__tab--active={activeTab === tab.key}
				onclick={() => activeTab = tab.key}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	{#if activeTab === 'general'}
		<div class="section">
			<h2>Preferences</h2>
			<div class="settings-card">
				<div class="settings-row">
					<span class="settings-row__label">Theme</span>
					<span class="settings-row__value">System default</span>
				</div>
				<div class="settings-row">
					<span class="settings-row__label">Notifications</span>
					<span class="settings-row__value">Enabled</span>
				</div>
			</div>
		</div>

		<div class="section" class:section--alert={!loading && providers.length === 0}>
			<div class="section__head">
				<h2>Providers</h2>
				<a href="/providers/new" class="btn-secondary">Add Provider</a>
			</div>
			{#if loading}
				<p class="text-muted">Loading...</p>
			{:else if providers.length === 0}
				<p class="section__alert">Must add at least one provider</p>
			{:else}
				<div class="provider-grid">
					{#each providers as p}
						<div class="provider-item">
							<p class="provider-item__name">{p.name}</p>
							<p class="provider-item__meta">v{p.version} {p.builtin ? '(built-in)' : ''}</p>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{:else if activeTab === 'connection'}
		<div class="section">
			<h2>Backend Connection</h2>
			<div class="settings-card">
				<div class="connection-form">
					<label class="connection-field">
						<span>Backend URL</span>
						<input type="url" bind:value={backendUrl} placeholder="http://localhost:8420/api/v1" />
					</label>
					<button class="btn-primary" onclick={saveBackendUrl}>Save</button>
				</div>
			</div>
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
		}
	}

	.tabs {
		display: flex;
		gap: $space-1;
		padding: $space-1;
		background: $neutral-100;
		border-radius: $radius-lg;
		margin-bottom: $space-6;

		&__tab {
			@include btn;
			color: $neutral-500;
			background: transparent;

			&--active {
				background: $neutral-0;
				color: $neutral-900;
				box-shadow: $shadow-sm;
			}

			&:hover:not(&--active) { color: $neutral-700; }
		}
	}

	.text-muted { color: $neutral-500; font-size: $text-sm; }

	.section {
		h2 { margin-bottom: $space-4; }

		& + & { margin-top: $space-8; }

		&__head {
			@include flex-between;
			margin-bottom: $space-4;

			h2 { margin-bottom: 0; }
		}

		&--alert {
			padding: $space-5;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;

			h2 { color: $error-700; }
		}

		&__alert {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $error-700;
		}
	}

	.settings-card { @include card; }

	.settings-row {
		@include flex-between;
		padding: $space-3 $space-5;
		font-size: $text-sm;

		& + & { border-top: 1px solid $neutral-100; }

		&__label { color: $neutral-600; }
		&__value { font-weight: $font-medium; color: $neutral-900; }
	}

	.provider-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
		gap: $space-3;
	}

	.provider-item {
		@include card;
		padding: $space-4;

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; margin-top: $space-1; }
	}

	.connection-form {
		padding: $space-5;
		display: flex;
		align-items: flex-end;
		gap: $space-3;
	}

	.connection-field {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: $space-1;

		span { font-size: $text-sm; font-weight: $font-medium; color: $neutral-700; }
		input { @include input-base; }
	}
</style>
