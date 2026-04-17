<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';

	interface Props {
		data: {
			label?: string;
			field?: string;
			cases?: string[];
		};
	}

	let { data }: Props = $props();

	const cases = $derived(data.cases?.length ? data.cases : ['Case 1', 'Case 2']);
	const allOutputs = $derived([...cases, 'default']);
</script>

<div class="nd">
	<Handle type="target" position={Position.Left} />
	<div class="nd__bar"></div>
	<div class="nd__content">
		<div class="nd__head">
			<span class="nd__tag">Switch</span>
		</div>
		<p class="nd__label">{data.label || 'Switch'}</p>
		{#if data.field}
			<p class="nd__field">{data.field}</p>
		{/if}
		<div class="nd__cases">
			{#each allOutputs as caseName, i}
				<span class="nd__case" class:nd__case--default={caseName === 'default'}>{caseName}</span>
			{/each}
		</div>
	</div>
	{#each allOutputs as caseName, i}
		<Handle
			type="source"
			position={Position.Right}
			id={caseName}
			style="top: {((i + 1) / (allOutputs.length + 1)) * 100}%;"
		/>
	{/each}
</div>

<style lang="scss">
	.nd {
		display: flex;
		min-width: 200px;
		max-width: 260px;
		border-radius: $radius-xl;
		background: $info-50;
		border: 2px solid $info-500;
		box-shadow: $shadow-md;
		font-family: $font-sans;
		overflow: visible;
		transition: box-shadow $transition-fast, border-color $transition-fast;

		&:hover {
			box-shadow: $shadow-lg;
			border-color: $info-500;
		}

		&__bar {
			width: 4px;
			flex-shrink: 0;
			background: $info-500;
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
		}

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			color: $info-600;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__label {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-900;
			line-height: $leading-tight;
		}

		&__field {
			font-family: $font-mono;
			font-size: 10px;
			color: $info-600;
			background: $info-100;
			padding: $space-1 $space-2;
			border-radius: $radius-sm;
		}

		&__cases {
			display: flex;
			flex-wrap: wrap;
			gap: $space-1;
		}

		&__case {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-medium;
			padding: 2px 6px;
			border-radius: $radius-sm;
			color: $info-600;
			background: $info-100;

			&--default {
				color: $neutral-500;
				background: $neutral-100;
			}
		}
	}
</style>
