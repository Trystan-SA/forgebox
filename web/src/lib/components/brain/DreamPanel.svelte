<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { DreamProposal, DreamChange } from '$lib/api/types';

	interface Props {
		proposals: DreamProposal[];
		open?: boolean;
	}

	let { proposals, open = false }: Props = $props();

	const dispatch = createEventDispatcher<{
		approve: { id: string };
		reject: { id: string };
		close: Record<string, never>;
	}>();

	let expandedId = $state<string | null>(null);

	function toggle(id: string) {
		expandedId = expandedId === id ? null : id;
	}

	function parseChanges(changesStr: string): DreamChange[] {
		try {
			const parsed = JSON.parse(changesStr);
			return Array.isArray(parsed) ? parsed : [];
		} catch {
			return [];
		}
	}

	function relativeTime(dateStr: string): string {
		const now = Date.now();
		const then = new Date(dateStr).getTime();
		const diffMs = now - then;
		const diffMin = Math.floor(diffMs / 60000);
		const diffHr = Math.floor(diffMin / 60);
		const diffDay = Math.floor(diffHr / 24);
		const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
		if (diffDay >= 1) return rtf.format(-diffDay, 'day');
		if (diffHr >= 1) return rtf.format(-diffHr, 'hour');
		if (diffMin >= 1) return rtf.format(-diffMin, 'minute');
		return 'just now';
	}

	function statusClass(status: string): string {
		if (status === 'approved') return 'dream-panel__status--success';
		if (status === 'rejected') return 'dream-panel__status--error';
		return 'dream-panel__status--warning';
	}

	function actionLabel(action: string): string {
		if (action === 'create') return 'Create';
		if (action === 'edit') return 'Edit';
		if (action === 'delete') return 'Delete';
		return action;
	}
</script>

<div class="dream-panel" class:dream-panel--open={open}>
	<div class="dream-panel__header">
		<div class="dream-panel__header-left">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2z" />
				<path d="M8 14s1.5 2 4 2 4-2 4-2" />
				<line x1="9" y1="9" x2="9.01" y2="9" />
				<line x1="15" y1="9" x2="15.01" y2="9" />
			</svg>
			<h2 class="dream-panel__title">Dream Proposals</h2>
			{#if proposals.filter(p => p.status === 'pending').length > 0}
				<span class="dream-panel__count">
					{proposals.filter(p => p.status === 'pending').length}
				</span>
			{/if}
		</div>
		<button type="button" class="dream-panel__close" onclick={() => dispatch('close', {})} aria-label="Close">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<div class="dream-panel__list">
		{#if proposals.length === 0}
			<div class="dream-panel__empty">
				<p>No dream proposals yet.</p>
				<span>When the agent generates knowledge suggestions, they'll appear here for review.</span>
			</div>
		{:else}
			{#each proposals as proposal}
				{@const changes = parseChanges(proposal.changes)}
				{@const isExpanded = expandedId === proposal.id}
				{@const isPending = proposal.status === 'pending'}

				<div class="dream-panel__item" class:dream-panel__item--expanded={isExpanded}>
					<button
						type="button"
						class="dream-panel__item-header"
						onclick={() => isPending && toggle(proposal.id)}
						disabled={!isPending}
					>
						<span class="dream-panel__status {statusClass(proposal.status)}">
							{proposal.status}
						</span>
						<span class="dream-panel__summary">{proposal.summary}</span>
						<span class="dream-panel__time">{relativeTime(proposal.created_at)}</span>
						{#if isPending}
							<svg
								width="12"
								height="12"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2.5"
								class="dream-panel__chevron"
								class:dream-panel__chevron--open={isExpanded}
							>
								<polyline points="6 9 12 15 18 9" />
							</svg>
						{/if}
					</button>

					{#if isExpanded && isPending}
						<div class="dream-panel__body">
							{#if changes.length > 0}
								<div class="dream-panel__changes">
									<div class="dream-panel__changes-label">Proposed changes</div>
									{#each changes as change}
										<div class="dream-panel__change">
											<span
												class="dream-panel__change-action"
												class:dream-panel__change-action--create={change.action === 'create'}
												class:dream-panel__change-action--edit={change.action === 'edit'}
												class:dream-panel__change-action--delete={change.action === 'delete'}
											>
												{actionLabel(change.action)}
											</span>
											<span class="dream-panel__change-file">
												{change.new_title ?? change.file_id ?? 'unknown'}
											</span>
											{#if change.reason}
												<p class="dream-panel__change-reason">{change.reason}</p>
											{/if}
										</div>
									{/each}
								</div>
							{/if}

							<div class="dream-panel__actions">
								<button
									type="button"
									class="dream-panel__approve"
									onclick={() => dispatch('approve', { id: proposal.id })}
								>
									Approve
								</button>
								<button
									type="button"
									class="dream-panel__reject"
									onclick={() => dispatch('reject', { id: proposal.id })}
								>
									Reject
								</button>
							</div>
						</div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>
</div>

<style lang="scss">
	.dream-panel {
		position: fixed;
		top: $topbar-height;
		right: 0;
		bottom: 0;
		width: 360px;
		background: $neutral-0;
		border-left: 1px solid $neutral-200;
		box-shadow: $shadow-lg;
		z-index: 40;
		display: flex;
		flex-direction: column;
		transform: translateX(100%);
		transition: transform $transition-slow;

		&--open {
			transform: translateX(0);
		}

		&__header {
			@include flex-between;
			padding: $space-4 $space-5;
			border-bottom: 1px solid $neutral-200;
			flex-shrink: 0;
		}

		&__header-left {
			display: flex;
			align-items: center;
			gap: $space-2;
			color: $neutral-500;
		}

		&__title {
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-800;
		}

		&__count {
			@include badge;
			background: $warning-100;
			color: $warning-700;
			font-size: 10px;
			padding: 1px 6px;
		}

		&__close {
			@include flex-center;
			width: 28px;
			height: 28px;
			border-radius: $radius-md;
			border: none;
			background: transparent;
			color: $neutral-400;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__list {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
		}

		&__empty {
			display: flex;
			flex-direction: column;
			gap: $space-2;
			padding: $space-8 $space-6;
			text-align: center;
			color: $neutral-400;

			p {
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-600;
				margin: 0;
			}

			span {
				font-size: $text-xs;
				line-height: $leading-relaxed;
			}
		}

		&__item {
			border-bottom: 1px solid $neutral-100;

			&--expanded {
				background: $neutral-50;
			}
		}

		&__item-header {
			display: flex;
			align-items: flex-start;
			gap: $space-2;
			width: 100%;
			text-align: left;
			padding: $space-3 $space-4;
			background: transparent;
			border: none;
			cursor: pointer;
			transition: background $transition-fast;

			&:hover:not(:disabled) { background: $neutral-50; }
			&:disabled { cursor: default; }
		}

		&__status {
			@include badge;
			font-size: 10px;
			flex-shrink: 0;
			text-transform: capitalize;

			&--warning { background: $warning-100; color: $warning-700; }
			&--success { background: $success-100; color: $success-700; }
			&--error { background: $error-100; color: $error-700; }
		}

		&__summary {
			flex: 1;
			font-size: $text-xs;
			color: $neutral-700;
			line-height: $leading-relaxed;
		}

		&__time {
			font-size: 10px;
			color: $neutral-400;
			white-space: nowrap;
			flex-shrink: 0;
		}

		&__chevron {
			color: $neutral-400;
			flex-shrink: 0;
			margin-top: 2px;
			transition: transform $transition-fast;

			&--open { transform: rotate(180deg); }
		}

		&__body {
			padding: $space-3 $space-4 $space-4;
			border-top: 1px solid $neutral-100;
		}

		&__changes {
			margin-bottom: $space-4;
		}

		&__changes-label {
			font-size: 10px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.06em;
			color: $neutral-400;
			margin-bottom: $space-2;
		}

		&__change {
			padding: $space-2 $space-3;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-md;
			margin-bottom: $space-2;
		}

		&__change-action {
			@include badge;
			font-size: 10px;
			margin-right: $space-2;
			text-transform: capitalize;

			&--create { background: $success-50; color: $success-700; }
			&--edit { background: $info-100; color: $info-600; }
			&--delete { background: $error-50; color: $error-700; }
		}

		&__change-file {
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-800;
		}

		&__change-reason {
			font-size: $text-xs;
			color: $neutral-500;
			line-height: $leading-relaxed;
			margin: $space-1 0 0;
		}

		&__actions {
			display: flex;
			gap: $space-2;
		}

		&__approve {
			@include btn;
			flex: 1;
			padding: $space-2 $space-3;
			font-size: $text-xs;
			font-weight: $font-semibold;
			background: $success-500;
			color: $neutral-0;
			border-radius: $radius-md;

			&:hover { background: $success-600; }
		}

		&__reject {
			@include btn;
			flex: 1;
			padding: $space-2 $space-3;
			font-size: $text-xs;
			font-weight: $font-semibold;
			background: $neutral-100;
			color: $neutral-700;
			border-radius: $radius-md;
			border: 1px solid $neutral-200;

			&:hover { background: $error-50; color: $error-700; border-color: $error-200; }
		}
	}

	@media (max-width: 640px) {
		.dream-panel {
			width: 100%;
		}
	}
</style>
