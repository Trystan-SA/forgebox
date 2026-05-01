<script lang="ts">
	import { onMount } from 'svelte';
	import { listProviders, createTask, streamTask, cancelTask } from '$lib/api/client';
	import type { Provider, TaskEvent } from '$lib/api/types';
	import TaskStream from '$lib/components/TaskStream.svelte';
	import ModelSelector from '$lib/components/ModelSelector.svelte';

	let prompt = $state('');
	let provider = $state('');
	let model = $state('');
	let memoryMb = $state(512);
	let networkAccess = $state(false);
	let providers = $state<Provider[]>([]);
	let events = $state<TaskEvent[]>([]);
	let isRunning = $state(false);
	let taskId = $state<string | null>(null);
	let cost = $state<number | null>(null);
	let duration = $state<string | null>(null);
	let error = $state<string | null>(null);
	let stopFn: (() => void) | null = null;

	onMount(async () => {
		try {
			providers = await listProviders();
		} catch {
			// ignore
		}
	});

	async function handleRun() {
		if (!prompt.trim() || isRunning) return;
		events = [];
		cost = null;
		duration = null;
		error = null;
		isRunning = true;

		try {
			const start = Date.now();
			const res = await createTask({
				prompt: prompt.trim(),
				provider: provider || undefined,
				model: model || undefined,
				memory_mb: memoryMb,
				network_access: networkAccess
			});
			taskId = res.task_id;

			stopFn = streamTask(
				res.task_id,
				(event) => {
					events = [...events, event];
					if (event.type === 'done') {
						isRunning = false;
						duration = `${((Date.now() - start) / 1000).toFixed(1)}s`;
					}
				},
				(err) => {
					error = err.message;
					isRunning = false;
				}
			);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create task';
			isRunning = false;
		}
	}

	async function handleCancel() {
		if (!taskId) return;
		stopFn?.();
		try {
			await cancelTask(taskId);
		} catch {
			// ignore
		}
		isRunning = false;
	}
</script>

<div class="page">
	<div class="page__header">
		<h1>Run Task</h1>
		<p>Describe what you want to do and ForgeBox will execute it in a secure VM.</p>
	</div>

	<textarea
		class="prompt-input"
		rows="4"
		placeholder="Describe what you want to do in plain English..."
		bind:value={prompt}
		disabled={isRunning}
	></textarea>

	<div class="settings">
		<ModelSelector
			{providers}
			bind:provider
			bind:model
			disabled={isRunning}
			allowAuto
			compact
		/>

		<select class="settings__select" bind:value={memoryMb} disabled={isRunning}>
			{#each [256, 512, 1024, 2048] as mb}
				<option value={mb}>{mb} MB</option>
			{/each}
		</select>

		<label class="settings__checkbox">
			<input type="checkbox" bind:checked={networkAccess} disabled={isRunning} />
			Network Access
		</label>
	</div>

	<div class="actions">
		{#if isRunning}
			<button class="btn-danger" onclick={handleCancel}>Cancel</button>
		{:else}
			<button class="btn-primary" onclick={handleRun} disabled={!prompt.trim()}>Run</button>
		{/if}
	</div>

	{#if error}
		<p class="error-msg">Error: {error}</p>
	{/if}

	<TaskStream {events} {isRunning} />

	{#if cost !== null && duration}
		<div class="result-summary">
			<span>Cost: <strong>${cost.toFixed(4)}</strong></span>
			<span>Duration: <strong>{duration}</strong></span>
		</div>
	{/if}
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}

	.prompt-input {
		@include input-base;
		margin-bottom: $space-4;
		resize: vertical;
	}

	.settings {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: $space-3;
		margin-bottom: $space-4;

		&__select, &__input {
			@include input-base;
			width: auto;
			min-width: 150px;
		}

		&__checkbox {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-sm;
			color: $neutral-600;
			cursor: pointer;

			input[type='checkbox'] {
				width: 1rem;
				height: 1rem;
				accent-color: $primary-600;
			}
		}
	}

	.actions {
		display: flex;
		align-items: center;
		gap: $space-3;
		margin-bottom: $space-6;
	}

	.error-msg {
		font-size: $text-sm;
		color: $error-600;
		margin-bottom: $space-4;
	}

	.result-summary {
		display: flex;
		gap: $space-6;
		margin-top: $space-4;
		font-size: $text-sm;
		color: $neutral-500;

		strong { color: $neutral-700; }
	}
</style>
