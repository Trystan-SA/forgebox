<script lang="ts">
	import type { TaskEvent } from '$lib/api/types';

	interface Props {
		events: TaskEvent[];
		isRunning: boolean;
	}

	let { events, isRunning }: Props = $props();
	let container: HTMLDivElement | undefined = $state();

	$effect(() => {
		// Scroll to bottom on new events
		if (events.length && container) {
			container.scrollTop = container.scrollHeight;
		}
	});
</script>

{#if events.length > 0 || isRunning}
	<div class="stream">
		<div class="stream__header">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<polyline points="4 17 10 11 4 5" /><line x1="12" y1="19" x2="20" y2="19" />
			</svg>
			<span>Output</span>
			{#if isRunning}
				<span class="stream__live">
					<span class="stream__pulse"></span>
					Streaming
				</span>
			{/if}
		</div>

		<div class="stream__body" bind:this={container}>
			{#each events as event}
				{#if event.type === 'text_delta'}
					<span class="stream__text">{event.text}</span>
				{:else if event.type === 'tool_call'}
					<div class="stream__tool">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
						</svg>
						<div>
							<span class="stream__tool-name">{event.tool_call?.name}</span>
							<pre class="stream__tool-input">{event.tool_call?.input}</pre>
						</div>
					</div>
				{:else if event.type === 'tool_result'}
					<div class="stream__result" class:stream__result--error={event.result?.is_error}>
						{event.result?.content}
					</div>
				{:else if event.type === 'error'}
					<div class="stream__error">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
						</svg>
						{event.error}
					</div>
				{:else if event.type === 'done'}
					<div class="stream__done">Task completed</div>
				{/if}
			{/each}
			{#if isRunning}
				<span class="stream__cursor"></span>
			{/if}
		</div>
	</div>
{/if}

<style lang="scss">
	.stream {
		@include card;
		overflow: hidden;

		&__header {
			@include flex-between;
			padding: $space-2 $space-4;
			border-bottom: 1px solid $neutral-200;
			background: $neutral-50;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-700;

			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__live {
			margin-left: auto;
			display: flex;
			align-items: center;
			gap: $space-1;
			font-size: $text-xs;
			color: $info-600;
		}

		&__pulse {
			width: 6px;
			height: 6px;
			border-radius: 50%;
			background: $info-600;
			animation: pulse 1.5s ease-in-out infinite;
		}

		&__body {
			max-height: 500px;
			overflow-y: auto;
			@include scrollbar-thin;
			background: $neutral-900;
			padding: $space-4;
			font-family: $font-mono;
			font-size: $text-sm;
		}

		&__text {
			color: $neutral-100;
			white-space: pre-wrap;
		}

		&__tool {
			display: flex;
			align-items: flex-start;
			gap: $space-2;
			margin: $space-2 0;
			padding: $space-2;
			border: 1px solid $neutral-700;
			border-radius: $radius-md;
			background: $neutral-800;
			color: $primary-300;
		}

		&__tool-name {
			font-weight: $font-semibold;
			color: $primary-300;
		}

		&__tool-input {
			margin-top: $space-1;
			font-size: $text-xs;
			color: $neutral-400;
			overflow-x: auto;
		}

		&__result {
			margin: $space-1 0;
			padding: $space-2;
			border: 1px solid $neutral-700;
			border-radius: $radius-md;
			background: $neutral-800;
			color: $neutral-300;
			font-size: $text-xs;
			white-space: pre-wrap;

			&--error {
				border-color: $error-700;
				background: rgba($error-600, 0.15);
				color: $error-500;
			}
		}

		&__error {
			display: flex;
			align-items: center;
			gap: $space-2;
			margin: $space-2 0;
			color: $error-500;
		}

		&__done {
			margin-top: $space-3;
			padding-top: $space-2;
			border-top: 1px solid $neutral-700;
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__cursor {
			display: inline-block;
			width: 8px;
			height: 16px;
			background: $neutral-400;
			animation: pulse 1s ease-in-out infinite;
		}
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.4; }
	}
</style>
