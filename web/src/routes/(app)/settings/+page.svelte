<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { listProviders, deleteProvider } from '$lib/api/client';
	import type { Provider } from '$lib/api/types';
	import { providerLabel } from '$lib/utils/providerLabels';
	import { refreshProviders } from '$lib/stores/providers.svelte';
	import { pushToast } from '$lib/stores/toasts.svelte';
	import { isTauri } from '$lib/platform';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Sparkline from '$lib/components/Sparkline.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
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
	let busyProviderId = $state<string | null>(null);
	let pendingDelete = $state<Provider | null>(null);

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

	// TODO(provider-usage): replace with a real fetch of last-7-day request
	// counts once the metrics pipeline ships. Tracked in specs/3.0.0-providers.md
	// §3.5.2 — the Sparkline contract (points: number[]) stays the same, only
	// the data source changes. Deterministic per-provider seed so the mock
	// chart is stable across re-renders until then.
	function mockUsage(seedKey: string): number[] {
		let h = 2166136261;
		for (let i = 0; i < seedKey.length; i++) {
			h ^= seedKey.charCodeAt(i);
			h = Math.imul(h, 16777619);
		}
		const out: number[] = [];
		for (let i = 0; i < 7; i++) {
			h ^= h << 13;
			h ^= h >>> 17;
			h ^= h << 5;
			const v = ((h >>> 0) % 1000) / 1000;
			out.push(Math.round(40 + v * 460));
		}
		return out;
	}

	const dayLabels = (() => {
		const fmt = new Intl.DateTimeFormat(undefined, { weekday: 'short' });
		const out: string[] = [];
		const today = new Date();
		for (let i = 6; i >= 0; i--) {
			const d = new Date(today);
			d.setDate(today.getDate() - i);
			out.push(fmt.format(d));
		}
		return out;
	})();

	async function refreshList() {
		try {
			providers = await listProviders();
			void refreshProviders();
		} catch {
			// ignore
		}
	}

	function requestDelete(p: Provider) {
		if (!p.id || p.builtin) return;
		pendingDelete = p;
	}

	async function confirmDelete() {
		const p = pendingDelete;
		if (!p?.id) return;
		busyProviderId = p.id;
		try {
			await deleteProvider(p.id);
			pushToast('Provider deleted', 'success');
			pendingDelete = null;
			await refreshList();
		} catch (err) {
			pushToast(err instanceof Error ? err.message : 'Failed to delete', 'error', 5000);
		} finally {
			busyProviderId = null;
		}
	}

	function cancelDelete() {
		if (busyProviderId) return;
		pendingDelete = null;
	}

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
					{#each providers as p (p.id ?? p.name)}
						{@const usage = mockUsage(p.id ?? p.name)}
						{@const total = usage.reduce((a, b) => a + b, 0)}
						<div class="provider-item">
							<div class="provider-item__head">
								<div class="provider-item__head-text">
									<p class="provider-item__name">{providerLabel(p)}</p>
									<p class="provider-item__meta">v{p.version} {p.builtin ? '(built-in)' : ''}</p>
								</div>
								{#if !p.builtin && p.id}
									<button
										type="button"
										class="provider-item__delete"
										aria-label={`Delete ${providerLabel(p)}`}
										title="Delete provider"
										disabled={busyProviderId === p.id}
										onclick={() => requestDelete(p)}
									>
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
											<polyline points="3 6 5 6 21 6" />
											<path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
											<path d="M10 11v6M14 11v6" />
											<path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
										</svg>
									</button>
								{/if}
							</div>

							<div class="provider-item__chart">
								<div class="provider-item__chart-head">
									<span class="provider-item__chart-title">Usage (7d)</span>
									<span class="provider-item__chart-total">{total.toLocaleString()} reqs</span>
								</div>
								<Sparkline
									points={usage}
									labels={dayLabels}
									ariaLabel={`${providerLabel(p)} usage over the last 7 days`}
								/>
							</div>
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

<ConfirmModal
	open={pendingDelete !== null}
	title="Delete provider"
	message={pendingDelete
		? `Permanently delete ${providerLabel(pendingDelete)}? Agents and tasks using it will fail until another provider of this type is configured. This cannot be undone.`
		: ''}
	confirmLabel="Delete"
	variant="danger"
	busy={busyProviderId !== null}
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
/>

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
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: $space-3;
	}

	.provider-item {
		@include card;
		padding: $space-4;
		display: flex;
		flex-direction: column;
		gap: $space-3;

		&__head {
			display: flex;
			align-items: flex-start;
			justify-content: space-between;
			gap: $space-2;
		}

		&__head-text {
			display: flex;
			flex-direction: column;
			gap: 2px;
			min-width: 0;
		}

		&__name { font-weight: $font-medium; color: $neutral-900; }
		&__meta { font-size: $text-xs; color: $neutral-500; }

		&__delete {
			@include flex-center;
			width: 24px;
			height: 24px;
			padding: 0;
			flex-shrink: 0;
			border: none;
			background: transparent;
			color: $neutral-400;
			border-radius: $radius-md;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover:not(:disabled) {
				background: $error-50;
				color: $error-600;
			}

			&:disabled { opacity: 0.4; cursor: not-allowed; }
		}

		&__chart {
			display: flex;
			flex-direction: column;
			gap: $space-1;
		}

		&__chart-head {
			display: flex;
			align-items: baseline;
			justify-content: space-between;
		}

		&__chart-title {
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__chart-total {
			font-size: $text-xs;
			font-variant-numeric: tabular-nums;
			color: $neutral-700;
			font-weight: $font-medium;
		}
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
