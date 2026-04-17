<script lang="ts">
	import { toasts, dismissToast } from '$lib/stores/toasts.svelte';
</script>

<div class="toasts" role="region" aria-label="Notifications" aria-live="polite">
	{#each toasts as t (t.id)}
		<div class="toast toast--{t.kind}">
			<span class="toast__icon" aria-hidden="true">
				{#if t.kind === 'success'}
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12" /></svg>
				{:else if t.kind === 'error'}
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" /></svg>
				{:else}
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>
				{/if}
			</span>
			<span class="toast__msg">{t.message}</span>
			<button class="toast__close" onclick={() => dismissToast(t.id)} aria-label="Dismiss">
				<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
			</button>
		</div>
	{/each}
</div>

<style lang="scss">
	.toasts {
		position: fixed;
		bottom: $space-6;
		right: $space-6;
		z-index: 100;
		display: flex;
		flex-direction: column;
		gap: $space-2;
		pointer-events: none;
	}

	.toast {
		pointer-events: auto;
		display: flex;
		align-items: center;
		gap: $space-2;
		min-width: 260px;
		max-width: 420px;
		padding: $space-2 $space-3;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-xl;
		box-shadow: $shadow-lg;
		font-size: $text-sm;
		color: $neutral-800;
		animation: toast-in 0.18s cubic-bezier(0.16, 1, 0.3, 1);

		&__icon {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 24px;
			height: 24px;
			border-radius: $radius-md;
			flex-shrink: 0;
		}

		&__msg {
			flex: 1;
			line-height: $leading-tight;
		}

		&__close {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 22px;
			height: 22px;
			background: none;
			border: none;
			border-radius: $radius-md;
			color: $neutral-400;
			cursor: pointer;
			transition: all $transition-fast;
			flex-shrink: 0;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&--success {
			.toast__icon { background: $success-50; color: $success-600; }
		}

		&--error {
			.toast__icon { background: $error-50; color: $error-600; }
		}

		&--info {
			.toast__icon { background: $info-50; color: $info-600; }
		}
	}

	@keyframes toast-in {
		from { opacity: 0; transform: translateY(8px); }
		to { opacity: 1; transform: translateY(0); }
	}
</style>
