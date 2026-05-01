<!--
	Generic destructive-action confirmation modal. Replaces native window.confirm()
	calls so destructive UX matches the rest of the app's design system.

	The caller controls visibility via `open`, supplies the copy, and gets
	`onConfirm`/`onCancel` callbacks. Pressing Escape or clicking the
	scrim cancels.
-->
<script lang="ts">
	interface Props {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		variant?: 'danger' | 'primary';
		busy?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let {
		open,
		title,
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		variant = 'danger',
		busy = false,
		onConfirm,
		onCancel
	}: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') onCancel();
		else if (e.key === 'Enter') onConfirm();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<button class="overlay" onclick={onCancel} aria-label="Close dialog"></button>
	<div class="modal" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
		<div class="modal__head" class:modal__head--danger={variant === 'danger'}>
			{#if variant === 'danger'}
				<span class="modal__icon" aria-hidden="true">
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
						<line x1="12" y1="9" x2="12" y2="13" />
						<line x1="12" y1="17" x2="12.01" y2="17" />
					</svg>
				</span>
			{/if}
			<h3 id="confirm-title" class="modal__title">{title}</h3>
		</div>
		<p class="modal__body">{message}</p>
		<div class="modal__actions">
			<button type="button" class="btn-secondary" onclick={onCancel} disabled={busy}>
				{cancelLabel}
			</button>
			<button
				type="button"
				class={variant === 'danger' ? 'btn-danger' : 'btn-primary'}
				onclick={onConfirm}
				disabled={busy}
			>
				{busy ? 'Working…' : confirmLabel}
			</button>
		</div>
	</div>
{/if}

<style lang="scss">
	.overlay {
		position: fixed;
		inset: 0;
		z-index: 50;
		background: rgba($neutral-900, 0.4);
		border: none;
		cursor: default;
		animation: overlay-in 0.15s ease-out;
	}

	.modal {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		z-index: 51;
		width: min(440px, 92vw);
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-2xl;
		box-shadow: $shadow-lg;
		padding: $space-5;
		display: flex;
		flex-direction: column;
		gap: $space-3;
		animation: modal-in 0.18s cubic-bezier(0.16, 1, 0.3, 1);

		&__head {
			display: flex;
			align-items: center;
			gap: $space-3;

			&--danger .modal__title { color: $error-700; }
		}

		&__icon {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			width: 36px;
			height: 36px;
			border-radius: $radius-full;
			background: $error-50;
			color: $error-600;
			flex-shrink: 0;
		}

		&__title {
			margin: 0;
			font-size: $text-lg;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__body {
			margin: 0;
			font-size: $text-sm;
			color: $neutral-600;
			line-height: $leading-relaxed;
		}

		&__actions {
			display: flex;
			justify-content: flex-end;
			gap: $space-2;
			margin-top: $space-2;
		}
	}

	@keyframes overlay-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes modal-in {
		from { opacity: 0; transform: translate(-50%, -50%) scale(0.95); }
		to { opacity: 1; transform: translate(-50%, -50%) scale(1); }
	}
</style>
