<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { listProviders } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import { isTauri } from '$lib/platform';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import AuditLog from '$lib/components/AuditLog.svelte';
	import Team from '$lib/components/Team.svelte';
	import Channels from '$lib/components/Channels.svelte';

	type TabKey = 'general' | 'team' | 'channels' | 'providers' | 'vm' | 'observability' | 'token-usage' | 'audit' | 'connection';
	const validTabs: TabKey[] = ['general', 'team', 'channels', 'providers', 'vm', 'observability', 'token-usage', 'audit', 'connection'];

	function tabFromQuery(value: string | null): TabKey {
		return validTabs.includes(value as TabKey) ? (value as TabKey) : 'general';
	}

	let activeTab = $state<TabKey>(tabFromQuery(page.url.searchParams.get('tab')));



	const vmDefaults = [
		{ label: 'Memory', value: '512 MB' },
		{ label: 'vCPUs', value: '1' },
		{ label: 'Timeout', value: '5 minutes' },
		{ label: 'Network Access', value: 'Disabled' }
	];
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
		{ key: 'team' as const, label: 'Team' },
		{ key: 'channels' as const, label: 'Channels' },
		{ key: 'providers' as const, label: 'Providers' },
		{ key: 'vm' as const, label: 'VM' },
		{ key: 'observability' as const, label: 'Observability' },
		{ key: 'token-usage' as const, label: 'Token Usage' },
		{ key: 'audit' as const, label: 'Audit Log' },
		...(showTauriSettings ? [{ key: 'connection' as const, label: 'Connection' }] : [])
	]);

	function selectTab(key: TabKey) {
		activeTab = key;
		const url = new URL(window.location.href);
		if (key === 'general') {
			url.searchParams.delete('tab');
		} else {
			url.searchParams.set('tab', key);
		}
		window.history.replaceState(null, '', url);
	}
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
				onclick={() => selectTab(tab.key)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	{#if activeTab === 'general'}
		<div class="section">
			<h2>Preferences</h2>
			<p class="section__hint">Display, theme, and notification defaults.</p>
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
	{:else if activeTab === 'team'}
		<Team />
	{:else if activeTab === 'channels'}
		<Channels />
	{:else if activeTab === 'providers'}
		<div class="section">
			<h2>Providers</h2>
			<p class="section__hint">LLM backends available to agents and tasks.</p>
			<div class="toolbar">
				<a href="/providers/new" class="btn-primary">Add Provider</a>
			</div>
			{#if loading}
				<p class="text-muted">Loading...</p>
			{:else if providers.length === 0}
				<EmptyState
					title="No providers configured"
					description="Add at least one provider so agents have an LLM to call."
				/>
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
	{:else if activeTab === 'vm'}
		<div class="section">
			<h2>VM Defaults</h2>
			<p class="section__hint">Firecracker VM defaults, network policies, and resource limits</p>
			<div class="settings-card">
				{#each vmDefaults as item}
					<div class="settings-row">
						<span class="settings-row__label">{item.label}</span>
						<span class="settings-row__value">{item.value}</span>
					</div>
				{/each}
			</div>
		</div>

		<div class="section">
			<h2>Storage</h2>
			<div class="settings-card">
				<div class="settings-info">
					Local filesystem storage is active. Configure S3 or GCS backends in the
					ForgeBox configuration file.
				</div>
			</div>
		</div>
	{:else if activeTab === 'observability'}
		<div class="section">
			<h2>Observability</h2>
			<p class="section__hint">Logs, task execution traces, and error rates</p>
			<EmptyState
				title="No trace data"
				description="Execution traces and logs will appear here once the observability pipeline is connected."
			/>
		</div>
	{:else if activeTab === 'token-usage'}
		<div class="section">
			<h2>Token Usage</h2>
			<p class="section__hint">Usage statistics per user, team, and provider</p>
			<EmptyState
				title="No usage data"
				description="Token usage tracking will appear here once tasks have been run."
			/>
		</div>
	{:else if activeTab === 'audit'}
		<AuditLog />
	{:else if activeTab === 'connection'}
		<div class="section">
			<h2>Backend Connection</h2>
			<p class="section__hint">Override the gateway URL used by this client.</p>
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

	.toolbar {
		display: flex;
		justify-content: flex-end;
		margin-bottom: $space-4;
	}

	.section {
		h2 { margin-bottom: $space-4; }

		& + & { margin-top: $space-8; }

		&__hint {
			margin: -$space-2 0 $space-4;
			font-size: $text-sm;
			color: $neutral-500;
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

	.settings-info {
		padding: $space-4 $space-5;
		font-size: $text-sm;
		color: $neutral-600;
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
