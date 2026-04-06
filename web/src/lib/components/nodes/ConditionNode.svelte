<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';

	interface Props {
		data: {
			label?: string;
			condition?: string;
		};
	}

	let { data }: Props = $props();
</script>

<div class="nd">
	<Handle type="target" position={Position.Left} />
	<div class="nd__bar"></div>
	<div class="nd__content">
		<div class="nd__head">
			<span class="nd__tag">Condition</span>
		</div>
		<p class="nd__label">{data.label || 'If / Else'}</p>
		{#if data.condition}
			<p class="nd__condition">{data.condition}</p>
		{/if}
		<div class="nd__branches">
			<span class="nd__branch nd__branch--true">True</span>
			<span class="nd__branch nd__branch--false">False</span>
		</div>
	</div>
	<Handle type="source" position={Position.Right} id="true" style="top: 35%;" />
	<Handle type="source" position={Position.Right} id="false" style="top: 65%;" />
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
			border-color: $info-600;
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

		&__condition {
			font-family: $font-mono;
			font-size: 10px;
			color: $info-600;
			background: $info-100;
			padding: $space-2;
			border-radius: $radius-md;
		}

		&__branches {
			display: flex;
			gap: $space-2;
		}

		&__branch {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-medium;
			padding: 2px 6px;
			border-radius: $radius-sm;

			&--true {
				color: $success-700;
				background: $success-100;
			}

			&--false {
				color: $error-700;
				background: $error-100;
			}
		}
	}
</style>
