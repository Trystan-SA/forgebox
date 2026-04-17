<script lang="ts">
	interface Props {
		open: boolean;
		name: string;
		description: string;
		onsave: (patch: { name: string; description: string }) => void | Promise<void>;
		onclose: () => void;
	}

	let { open, name, description, onsave, onclose }: Props = $props();

	let draftName = $state(name);
	let draftDesc = $state(description);
	let saving = $state(false);
	let nameInput: HTMLInputElement | undefined = $state();

	$effect(() => {
		if (open) {
			draftName = name;
			draftDesc = description;
			saving = false;
			setTimeout(() => nameInput?.focus(), 0);
		}
	});

	const canSave = $derived(draftName.trim().length > 0 && !saving);

	async function handleSave() {
		if (!canSave) return;
		saving = true;
		try {
			await onsave({ name: draftName.trim(), description: draftDesc.trim() });
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'Escape') onclose();
		if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleSave();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<button class="overlay" onclick={onclose} aria-label="Close editor"></button>
	<div class="modal" role="dialog" aria-label="Edit automation details">
		<div class="modal__head">
			<span class="modal__tag">edit</span>
			<button class="modal__close" onclick={onclose} aria-label="Close">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
			</button>
		</div>
		<div class="modal__body">
			<label class="field">
				<span class="field__label">Name</span>
				<input
					bind:this={nameInput}
					class="field__input"
					type="text"
					bind:value={draftName}
					placeholder="Automation name"
					maxlength="120"
				/>
			</label>
			<label class="field">
				<span class="field__label">Description</span>
				<textarea
					class="field__textarea"
					bind:value={draftDesc}
					placeholder="What does this automation do?"
					rows="4"
				></textarea>
			</label>
		</div>
		<div class="modal__foot">
			<button class="btn-secondary" onclick={onclose} disabled={saving}>Cancel</button>
			<button class="btn-primary" onclick={handleSave} disabled={!canSave}>
				{saving ? 'Saving…' : 'Save'}
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
		animation: meta-fade 0.15s ease-out;
	}

	.modal {
		position: fixed;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		z-index: 51;
		width: min(520px, 92vw);
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-2xl;
		box-shadow: $shadow-lg;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		animation: meta-in 0.18s cubic-bezier(0.16, 1, 0.3, 1);

		&__head {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: $space-3 $space-4;
			border-bottom: 1px solid $neutral-100;
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
			padding: $space-4;
			display: flex;
			flex-direction: column;
			gap: $space-4;
		}

		&__foot {
			display: flex;
			justify-content: flex-end;
			gap: $space-2;
			padding: $space-3 $space-4;
			border-top: 1px solid $neutral-100;
			background: $neutral-50;
		}
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		&__label {
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
			text-transform: uppercase;
			letter-spacing: 0.04em;
		}

		&__input {
			@include input-base;
			font-size: $text-sm;
			padding: $space-2 $space-3;
		}

		&__textarea {
			@include input-base;
			font-size: $text-sm;
			padding: $space-2 $space-3;
			resize: vertical;
			min-height: 80px;
			font-family: inherit;
		}
	}

	@keyframes meta-fade {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes meta-in {
		from { opacity: 0; transform: translate(-50%, -50%) scale(0.95); }
		to { opacity: 1; transform: translate(-50%, -50%) scale(1); }
	}
</style>