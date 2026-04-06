<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';

	interface Props {
		data: {
			label?: string;
			provider?: string;
			model?: string;
			prompt?: string;
		};
	}

	let { data }: Props = $props();
</script>

<div class="nd">
	<Handle type="target" position={Position.Left} />
	<div class="nd__bar"></div>
	<div class="nd__content">
		<div class="nd__head">
			<span class="nd__tag">AI</span>
			{#if data.provider}
				<span class="nd__provider">{data.provider}</span>
			{/if}
		</div>
		<p class="nd__label">{data.label || 'AI Completion'}</p>
		{#if data.prompt}
			<p class="nd__prompt">{data.prompt.length > 50 ? data.prompt.slice(0, 50) + '…' : data.prompt}</p>
		{/if}
		<div class="nd__foot">
			{#if data.model}
				<span class="nd__chip">{data.model}</span>
			{/if}
		</div>
	</div>
	<Handle type="source" position={Position.Right} />
</div>

<style lang="scss">
	.nd {
		display: flex;
		min-width: 210px;
		max-width: 280px;
		border-radius: $radius-xl;
		background: $primary-50;
		border: 2px solid $primary-500;
		box-shadow: $shadow-md;
		font-family: $font-sans;
		overflow: visible;
		transition: box-shadow $transition-fast, border-color $transition-fast;

		&:hover {
			box-shadow: $shadow-lg;
			border-color: $primary-600;
		}

		&__bar {
			width: 4px;
			flex-shrink: 0;
			background: $primary-500;
		}

		&__content {
			flex: 1;
			padding: $space-3;
			display: flex;
			flex-direction: column;
			gap: $space-2;
		}

		&__head {
			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			color: $primary-700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__provider {
			margin-left: auto;
			font-family: $font-mono;
			font-size: 9px;
			color: $primary-600;
			background: $primary-100;
			padding: 1px 5px;
			border-radius: $radius-sm;
		}

		&__label {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-900;
			line-height: $leading-tight;
		}

		&__prompt {
			font-family: $font-mono;
			font-size: 10px;
			color: $primary-700;
			background: $primary-100;
			padding: $space-2;
			border-radius: $radius-md;
			line-height: $leading-relaxed;
		}

		&__foot {
			display: flex;
			gap: $space-2;
		}

		&__chip {
			font-family: $font-mono;
			font-size: 10px;
			color: $primary-700;
			background: $primary-100;
			padding: 2px 6px;
			border-radius: $radius-sm;
		}
	}
</style>
