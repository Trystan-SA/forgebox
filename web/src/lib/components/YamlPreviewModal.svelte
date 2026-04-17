<script lang="ts">
	import { tokenizeYaml } from '$lib/utils/yamlHighlight';

	interface Props {
		open: boolean;
		title: string;
		yaml: string;
		loading?: boolean;
		error?: string | null;
		footer?: string;
		onclose: () => void;
	}

	let { open, title, yaml, loading = false, error = null, footer, onclose }: Props = $props();

	let copied = $state(false);
	let copyError = $state<string | null>(null);

	const lines = $derived(tokenizeYaml(yaml));

	async function copy() {
		try {
			await navigator.clipboard.writeText(yaml);
			copied = true;
			copyError = null;
			setTimeout(() => { copied = false; }, 1500);
		} catch {
			copyError = 'Clipboard unavailable';
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (open && e.key === 'Escape') onclose();
	}

	const displayError = $derived(error ?? copyError);
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<button class="overlay" onclick={onclose} aria-label="Close preview"></button>
	<div class="modal" role="dialog" aria-label="{title} YAML preview">
		<div class="modal__head">
			<div class="modal__title">
				<span class="modal__tag">yaml</span>
				<span class="modal__name">{title}</span>
			</div>
			<div class="modal__actions">
				<button class="btn-ghost modal__copy" onclick={copy} disabled={loading || !yaml}>
					{copied ? 'Copied' : 'Copy'}
				</button>
				<button class="modal__close" onclick={onclose} aria-label="Close">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
				</button>
			</div>
		</div>
		<div class="modal__body">
			{#if loading}
				<div class="modal__state">Loading…</div>
			{:else if displayError}
				<div class="modal__state modal__state--error">{displayError}</div>
			{:else}
				<pre class="modal__pre"><code>{#each lines as line, i}{#each line as tok}<span class="yml-{tok.kind}">{tok.text}</span>{/each}{i < lines.length - 1 ? '\n' : ''}{/each}</code></pre>
			{/if}
		</div>
		{#if footer}
			<div class="modal__foot">
				<span>{footer}</span>
			</div>
		{/if}
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
		width: min(720px, 92vw);
		max-height: 82vh;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-2xl;
		box-shadow: $shadow-lg;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		animation: modal-in 0.18s cubic-bezier(0.16, 1, 0.3, 1);

		&__head {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: $space-3;
			padding: $space-3 $space-4;
			border-bottom: 1px solid $neutral-100;
		}

		&__title {
			display: flex;
			align-items: center;
			gap: $space-2;
			min-width: 0;
		}

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			color: $primary-700;
			background: $primary-50;
			padding: 2px 6px;
			border-radius: $radius-sm;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__name {
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-800;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		&__actions {
			display: flex;
			align-items: center;
			gap: $space-2;
			flex-shrink: 0;
		}

		&__copy {
			font-family: $font-mono;
			font-size: $text-xs;
			padding: $space-1 $space-3;
			text-transform: uppercase;
			letter-spacing: 0.04em;

			&:disabled { opacity: 0.4; cursor: not-allowed; }
		}

		&__close {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 28px;
			border: none;
			background: none;
			border-radius: $radius-lg;
			color: $neutral-400;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__body {
			flex: 1;
			overflow: auto;
			@include scrollbar-thin;
			background: $neutral-50;
		}

		&__pre {
			margin: 0;
			padding: $space-4;
			font-family: $font-mono;
			font-size: $text-sm;
			line-height: $leading-relaxed;
			color: $neutral-700;
			white-space: pre;

			code {
				font-family: inherit;
				font-size: inherit;
				background: none;
				padding: 0;
			}

			:global(.yml-key)     { color: $primary-700; font-weight: $font-medium; }
			:global(.yml-string)  { color: $success-700; }
			:global(.yml-number)  { color: $warning-700; }
			:global(.yml-bool)    { color: $info-600; font-weight: $font-medium; }
			:global(.yml-null)    { color: $error-500; font-style: italic; }
			:global(.yml-comment) { color: $neutral-400; font-style: italic; }
			:global(.yml-marker)  { color: $neutral-500; font-weight: $font-bold; }
			:global(.yml-punc)    { color: $neutral-400; }
			:global(.yml-plain)   { color: $neutral-800; }
			:global(.yml-indent)  { color: inherit; }
		}

		&__state {
			padding: $space-6;
			text-align: center;
			font-size: $text-sm;
			color: $neutral-500;

			&--error { color: $error-600; }
		}

		&__foot {
			padding: $space-2 $space-4;
			border-top: 1px solid $neutral-100;
			text-align: center;

			span {
				font-size: $text-xs;
				color: $neutral-400;
			}
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