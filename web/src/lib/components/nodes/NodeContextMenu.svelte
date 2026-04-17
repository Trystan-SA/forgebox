<script lang="ts">
	import type { Node } from '@xyflow/svelte';

	interface Props {
		open: boolean;
		x: number;
		y: number;
		node: Node | null;
		ontoggleDisabled: () => void;
		ondelete: () => void;
		onclose: () => void;
	}

	let { open, x, y, node, ontoggleDisabled, ondelete, onclose }: Props = $props();

	const isDisabled = $derived(Boolean(node?.data?.disabled));
</script>

{#if open && node}
	<button class="overlay" onclick={onclose} aria-label="Close node menu"></button>
	<div class="menu" style="left: {x}px; top: {y}px;" role="menu">
		<div class="menu__head">
			<span class="menu__type">{node.type}</span>
			<span class="menu__label">{node.data?.label ?? node.id}</span>
		</div>
		<button class="menu__item" onclick={ontoggleDisabled}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				{#if isDisabled}
					<polygon points="5 3 19 12 5 21 5 3" />
				{:else}
					<circle cx="12" cy="12" r="10" /><line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
				{/if}
			</svg>
			<span>{isDisabled ? 'Enable' : 'Disable'}</span>
			<span class="menu__hint">pass-through</span>
		</button>
		<button class="menu__item menu__item--danger" onclick={ondelete}>
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polyline points="3 6 5 6 21 6" /><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6" /><path d="M10 11v6" /><path d="M14 11v6" />
			</svg>
			<span>Delete</span>
		</button>
	</div>
{/if}

<style lang="scss">
	.overlay {
		position: absolute;
		inset: 0;
		z-index: 10;
		background: rgba($neutral-900, 0.08);
		border: none;
		cursor: default;
		animation: node-ctx-fade 0.12s ease-out;
	}

	.menu {
		position: absolute;
		z-index: 12;
		min-width: 200px;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-xl;
		box-shadow: $shadow-lg;
		padding: $space-1;
		animation: node-ctx-in 0.12s cubic-bezier(0.16, 1, 0.3, 1);
		transform-origin: top left;

		&__head {
			padding: $space-2 $space-3 $space-1;
			display: flex;
			flex-direction: column;
			gap: 2px;
			border-bottom: 1px solid $neutral-100;
			margin-bottom: $space-1;
		}

		&__type {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-bold;
			color: $neutral-400;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__label {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-800;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
			max-width: 240px;
		}

		&__item {
			display: flex;
			align-items: center;
			gap: $space-2;
			width: 100%;
			padding: $space-2 $space-3;
			background: none;
			border: none;
			border-radius: $radius-lg;
			cursor: pointer;
			text-align: left;
			font-size: $text-sm;
			color: $neutral-800;
			transition: all $transition-fast;

			&:hover { background: $neutral-50; }

			svg { color: $neutral-500; flex-shrink: 0; }

			&--danger {
				color: $error-600;
				svg { color: $error-500; }

				&:hover { background: $error-50; }
			}
		}

		&__hint {
			margin-left: auto;
			font-family: $font-mono;
			font-size: 10px;
			color: $neutral-400;
		}
	}

	@keyframes node-ctx-fade {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes node-ctx-in {
		from { opacity: 0; transform: scale(0.95); }
		to { opacity: 1; transform: scale(1); }
	}
</style>