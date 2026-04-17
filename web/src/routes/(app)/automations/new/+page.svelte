<script lang="ts">
	import { createAutomation } from '$lib/api/client';
	import { goto } from '$app/navigation';

	let name = $state('');
	let description = $state('');
	let sharing = $state<'personal' | 'team' | 'org'>('personal');
	let loading = $state(false);
	let error = $state<string | null>(null);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;

		loading = true;
		error = null;

		try {
			const automation = await createAutomation({
				name: name.trim(),
				description: description.trim(),
				sharing
			});
			goto(`/automations/${automation.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create';
		} finally {
			loading = false;
		}
	}
</script>

<div class="page">
	<div class="page__header">
		<a href="/automations" class="page__back">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
			Back
		</a>
		<h1>New Workflow</h1>
		<p>Set up the basics, then build your workflow visually.</p>
	</div>

	{#if error}
		<div class="page__error-box">{error}</div>
	{/if}

	<form class="form" onsubmit={handleSubmit}>
		<div class="form__card">
			<label class="field">
				<span>Name</span>
				<input type="text" bind:value={name} placeholder="e.g. Deploy status checker" required disabled={loading} />
			</label>

			<label class="field">
				<span>Description</span>
				<textarea bind:value={description} placeholder="What does this workflow do?" rows="3" disabled={loading}></textarea>
			</label>

			<fieldset class="sharing">
				<legend>Sharing</legend>
				<label class="sharing__option">
					<input type="radio" bind:group={sharing} value="personal" />
					<div>
						<strong>Personal</strong>
						<span>Only you can see and use this</span>
					</div>
				</label>
				<label class="sharing__option">
					<input type="radio" bind:group={sharing} value="team" />
					<div>
						<strong>Team</strong>
						<span>Share with your team members</span>
					</div>
				</label>
				<label class="sharing__option">
					<input type="radio" bind:group={sharing} value="org" />
					<div>
						<strong>Organization</strong>
						<span>Available to everyone in the org</span>
					</div>
				</label>
			</fieldset>
		</div>

		<button type="submit" class="btn-primary form__submit" disabled={loading || !name.trim()}>
			{loading ? 'Creating...' : 'Create & Open Editor'}
		</button>
	</form>
</div>

<style lang="scss">
	.page {
		max-width: 560px;

		&__header {
			margin-bottom: $space-6;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}

		&__back {
			display: inline-flex;
			align-items: center;
			gap: $space-1;
			font-size: $text-sm;
			color: $neutral-500;
			margin-bottom: $space-3;
			transition: color $transition-fast;

			&:hover { color: $neutral-700; }
		}

		&__error-box {
			padding: $space-3;
			margin-bottom: $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}
	}

	.form {
		&__card {
			@include card;
			padding: $space-6;
			display: flex;
			flex-direction: column;
			gap: $space-5;
		}

		&__submit {
			margin-top: $space-4;
			width: 100%;
		}
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		span {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-700;
		}

		input, textarea {
			@include input-base;
		}

		textarea { resize: vertical; }
	}

	.sharing {
		border: none;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: $space-3;

		legend {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-700;
			margin-bottom: $space-2;
		}

		&__option {
			display: flex;
			align-items: flex-start;
			gap: $space-3;
			padding: $space-3;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			cursor: pointer;
			transition: border-color $transition-fast;

			&:has(input:checked) {
				border-color: $primary-500;
				background: $primary-50;
			}

			input { margin-top: 3px; accent-color: $primary-600; }

			strong {
				display: block;
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-800;
			}

			span {
				font-size: $text-xs;
				color: $neutral-500;
			}
		}
	}
</style>
