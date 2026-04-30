<script lang="ts">
	import { onMount } from 'svelte';
	import { format } from 'date-fns';
	import { listAuditEntries } from '$lib/api/client';
	import type { AuditEntry } from '$lib/api/types';
	import EmptyState from '$lib/components/EmptyState.svelte';

	let entries = $state<AuditEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let userId = $state('');
	let decision = $state<'all' | 'allow' | 'deny'>('all');
	let dateFrom = $state('');
	let dateTo = $state('');

	const filtered = $derived(
		entries.filter((e) => {
			if (userId && !e.user_id.toLowerCase().includes(userId.toLowerCase())) return false;
			if (decision !== 'all' && e.decision !== decision) return false;
			if (dateFrom && e.timestamp < dateFrom) return false;
			if (dateTo && e.timestamp > dateTo) return false;
			return true;
		})
	);

	onMount(async () => {
		try {
			entries = await listAuditEntries();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load';
		} finally {
			loading = false;
		}
	});
</script>

<section class="section">
	<h2>Audit Log</h2>
	<p class="section__hint">Track all tool calls and permission decisions</p>
	<div class="filters">
	<label class="filter">
		<span>User ID</span>
		<input type="text" placeholder="Filter by user..." bind:value={userId} />
	</label>
	<label class="filter">
		<span>Decision</span>
		<select bind:value={decision}>
			<option value="all">All</option>
			<option value="allow">Allow</option>
			<option value="deny">Deny</option>
		</select>
	</label>
	<label class="filter">
		<span>From</span>
		<input type="text" placeholder="YYYY-MM-DD" bind:value={dateFrom} />
	</label>
	<label class="filter">
		<span>To</span>
		<input type="text" placeholder="YYYY-MM-DD" bind:value={dateTo} />
	</label>
</div>

{#if loading}
	<p class="text-muted">Loading...</p>
{:else if error}
	<p class="text-error">Error: {error}</p>
{:else if filtered.length === 0}
	<EmptyState
		title="No audit entries found"
		description="Audit entries will appear here when tools are executed."
	/>
{:else}
	<div class="table-wrap">
		<table class="table">
			<thead>
				<tr>
					<th>Timestamp</th>
					<th>User</th>
					<th>Action</th>
					<th>Tool</th>
					<th>Decision</th>
					<th>Reason</th>
				</tr>
			</thead>
			<tbody>
				{#each filtered as entry}
					<tr>
						<td class="table__nowrap table__muted">
							{format(new Date(entry.timestamp), 'MMM d, yyyy HH:mm:ss')}
						</td>
						<td class="table__mono">{entry.user_id.slice(0, 8)}</td>
						<td>{entry.action}</td>
						<td class="table__mono">{entry.tool ?? '-'}</td>
						<td>
							<span class="decision" class:decision--allow={entry.decision === 'allow'} class:decision--deny={entry.decision === 'deny'}>
								{entry.decision}
							</span>
						</td>
						<td class="table__truncate table__muted">{entry.reason ?? '-'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
</section>

<style lang="scss">
	.section {
		h2 { margin-bottom: $space-4; }

		&__hint {
			margin: -$space-2 0 $space-4;
			font-size: $text-sm;
			color: $neutral-500;
		}
	}

	.text-muted { color: $neutral-500; }
	.text-error { color: $error-600; }

	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: $space-3;
		align-items: flex-end;
		margin-bottom: $space-6;
	}

	.filter {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		span { font-size: $text-xs; font-weight: $font-medium; color: $neutral-600; }
		input, select { @include input-base; width: auto; }
	}

	.table-wrap {
		@include card;
		overflow: hidden;
	}

	.table {
		text-align: left;
		font-size: $text-sm;

		thead {
			background: $neutral-50;
			border-bottom: 1px solid $neutral-100;
		}

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

		&__nowrap { white-space: nowrap; }
		&__mono { font-family: $font-mono; font-size: $text-xs; color: $neutral-700; }
		&__muted { color: $neutral-400; }
		&__truncate { max-width: 200px; @include truncate; }
	}

	.decision {
		@include badge;

		&--allow { background: $success-100; color: $success-700; }
		&--deny { background: $error-100; color: $error-700; }
	}
</style>
