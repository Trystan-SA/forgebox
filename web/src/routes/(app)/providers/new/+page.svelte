<script lang="ts">
	import { goto } from '$app/navigation';
	import { pushToast } from '$lib/stores/toasts.svelte';
	import { createProvider } from '$lib/api/client';
	import { refreshProviders } from '$lib/stores/providers.svelte';
	import type { ProviderType } from '$lib/api/types';

	type ProviderOption = {
		value: ProviderType;
		label: string;
		placeholder: string;
		credentialLabel: string;
		credentialField: 'api_key' | 'token' | null;
		hint?: string;
		showBaseUrl: boolean;
	};

	// Labels here mirror the canonical labels in
	// internal/plugins/labels.go — the gateway derives the display name
	// from the type, so what you pick on the left is what you'll see in
	// the model selector.
	const providerOptions: ProviderOption[] = [
		{
			value: 'anthropic-api',
			label: 'Anthropic (API)',
			placeholder: 'sk-ant-api-...',
			credentialLabel: 'API Key',
			credentialField: 'api_key',
			showBaseUrl: false
		},
		{
			value: 'anthropic-subscription',
			label: 'Anthropic (Subscription)',
			placeholder: 'sk-ant-oat01-...',
			credentialLabel: 'OAuth Setup Token',
			credentialField: 'token',
			hint: 'Generate a token by running `claude setup-token` with the official Claude CLI while logged into your Claude Max account.',
			showBaseUrl: false
		},
		{
			value: 'openai',
			label: 'OpenAI',
			placeholder: 'sk-...',
			credentialLabel: 'API Key',
			credentialField: 'api_key',
			showBaseUrl: false
		},
		{
			value: 'ollama',
			label: 'Ollama',
			placeholder: 'No key required',
			credentialLabel: 'API Key',
			credentialField: null,
			showBaseUrl: true
		}
	];

	let type = $state<ProviderType>('anthropic-api');
	let apiKey = $state('');
	let baseUrl = $state('');
	let saving = $state(false);

	const selected = $derived(providerOptions.find((p) => p.value === type)!);
	const canSave = $derived(
		(selected.credentialField === null || apiKey.trim().length > 0) && !saving
	);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!canSave) return;
		saving = true;
		try {
			const config: Record<string, unknown> = {};
			if (selected.credentialField && apiKey.trim()) {
				config[selected.credentialField] = apiKey.trim();
			}
			if (baseUrl.trim()) {
				config.base_url = baseUrl.trim();
			}
			await createProvider({ type, config });
			await refreshProviders();
			pushToast('Provider added', 'success');
			goto('/settings?tab=providers');
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
			{#if selected.credentialField}
				<label class="fld">
					<span class="fld__lbl">{selected.credentialLabel}</span>
					<input
						class="fld__input"
						type="password"
						bind:value={apiKey}
						placeholder={selected.placeholder}
						required
						disabled={saving}
						autocomplete="off"
					/>
					{#if selected.hint}
						<span class="fld__hint">{selected.hint}</span>
					{/if}
				</label>
			{/if}

			{#if selected.showBaseUrl}
				<label class="fld">
					<span class="fld__lbl">Base URL (optional)</span>
					<input
						class="fld__input"
						type="url"
						bind:value={baseUrl}
						placeholder="http://localhost:11434"
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

		&__hint {
			font-size: 11px;
			color: $neutral-500;
			line-height: 1.4;
		}
	}
</style>
