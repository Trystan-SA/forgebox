<script lang="ts">
	import { goto } from '$app/navigation';
	import { pushToast } from '$lib/stores/toasts.svelte';

	type ProviderType = 'anthropic' | 'openai' | 'google' | 'ollama' | 'custom';

	const providerOptions: { value: ProviderType; label: string; placeholder: string }[] = [
		{ value: 'anthropic', label: 'Anthropic', placeholder: 'sk-ant-...' },
		{ value: 'openai', label: 'OpenAI', placeholder: 'sk-...' },
		{ value: 'google', label: 'Google', placeholder: 'AIza...' },
		{ value: 'ollama', label: 'Ollama', placeholder: 'No key required' },
		{ value: 'custom', label: 'Custom (OpenAI-compatible)', placeholder: 'sk-...' }
	];

	let type = $state<ProviderType>('anthropic');
	let name = $state('');
	let apiKey = $state('');
	let baseUrl = $state('');
	let saving = $state(false);

	const selected = $derived(providerOptions.find((p) => p.value === type)!);
	const canSave = $derived(name.trim().length > 0 && (type === 'ollama' || apiKey.trim().length > 0) && !saving);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!canSave) return;
		saving = true;
		try {
			const existing = JSON.parse(localStorage.getItem('forgebox_providers') ?? '[]');
			existing.push({
				id: crypto.randomUUID().slice(0, 8),
				type,
				name: name.trim(),
				api_key: apiKey.trim(),
				base_url: baseUrl.trim() || undefined,
				created_at: new Date().toISOString()
			});
			localStorage.setItem('forgebox_providers', JSON.stringify(existing));
			pushToast('Provider added', 'success');
			goto('/providers');
		} catch (err) {
			pushToast(err instanceof Error ? err.message : 'Failed to save', 'error', 5000);
			saving = false;
		}
	}
</script>

<div class="cr">
	<div class="cr__top">
		<a href="/providers" class="cr__back" aria-label="Back to providers">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
		</a>
		<div>
			<h1 class="cr__title">Add Provider</h1>
			<p class="cr__sub">Configure credentials for an LLM provider.</p>
		</div>
	</div>

	<form class="cr__form" onsubmit={handleSubmit}>
		<section class="sec">
			<span class="sec__label">Provider</span>
			<div class="types">
				{#each providerOptions as opt}
					<button
						type="button"
						class="types__btn"
						class:types__btn--on={type === opt.value}
						onclick={() => { type = opt.value; }}
						disabled={saving}
					>
						<strong>{opt.label}</strong>
					</button>
				{/each}
			</div>
		</section>

		<section class="sec">
			<label class="fld">
				<span class="fld__lbl">Display Name</span>
				<input
					class="fld__input"
					type="text"
					bind:value={name}
					placeholder="e.g. Production Anthropic"
					required
					disabled={saving}
				/>
			</label>

			{#if type !== 'ollama'}
				<label class="fld">
					<span class="fld__lbl">API Key</span>
					<input
						class="fld__input"
						type="password"
						bind:value={apiKey}
						placeholder={selected.placeholder}
						required
						disabled={saving}
						autocomplete="off"
					/>
				</label>
			{/if}

			{#if type === 'ollama' || type === 'custom'}
				<label class="fld">
					<span class="fld__lbl">Base URL {type === 'custom' ? '' : '(optional)'}</span>
					<input
						class="fld__input"
						type="url"
						bind:value={baseUrl}
						placeholder={type === 'ollama' ? 'http://localhost:11434' : 'https://api.example.com/v1'}
						required={type === 'custom'}
						disabled={saving}
					/>
				</label>
			{/if}
		</section>

		<div class="cr__actions">
			<a href="/providers" class="btn-secondary">Cancel</a>
			<button type="submit" class="cr__submit" disabled={!canSave}>
				{saving ? 'Saving…' : 'Add Provider'}
			</button>
		</div>
	</form>
</div>

<style lang="scss">
	.cr {
		max-width: 640px;
		margin: 0 auto;

		&__top {
			display: flex;
			align-items: flex-start;
			gap: $space-3;
			margin-bottom: $space-6;
		}

		&__back {
			@include flex-center;
			width: 36px;
			height: 36px;
			flex-shrink: 0;
			border-radius: $radius-lg;
			color: $neutral-400;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__title {
			font-size: $text-2xl;
			font-weight: $font-bold;
			color: $neutral-900;
			line-height: 1.2;
		}

		&__sub {
			font-size: $text-sm;
			color: $neutral-400;
			margin-top: 2px;
		}

		&__form {
			display: flex;
			flex-direction: column;
			gap: $space-5;
		}

		&__actions {
			display: flex;
			justify-content: flex-end;
			gap: $space-2;
			margin-top: $space-2;
		}

		&__submit {
			@include btn;
			padding: $space-3 $space-5;
			font-weight: $font-semibold;
			color: $neutral-0;
			background: $primary-600;
			border-radius: $radius-xl;

			&:hover:not(:disabled) { background: $primary-700; }
			&:disabled { opacity: 0.5; cursor: not-allowed; }
		}
	}

	.sec {
		@include card;
		padding: $space-4 $space-5;
		display: flex;
		flex-direction: column;
		gap: $space-3;

		&__label {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: $neutral-400;
		}
	}

	.types {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
		gap: $space-2;

		&__btn {
			padding: $space-3;
			text-align: left;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			cursor: pointer;
			transition: all $transition-fast;

			strong {
				display: block;
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-800;
			}

			&:hover:not(:disabled) { border-color: $primary-300; }

			&--on {
				border-color: $primary-600;
				background: $primary-50;

				strong { color: $primary-700; }
			}
		}
	}

	.fld {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		&__lbl {
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
		}

		&__input {
			@include input-base;
			font-size: $text-sm;
			padding: 7px $space-3;
			border-radius: $radius-md;
		}
	}
</style>
