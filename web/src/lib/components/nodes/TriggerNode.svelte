<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';

	interface Props {
		data: {
			label?: string;
			triggerType?: string;
		};
	}

	let { data }: Props = $props();
</script>

<div class="nd">
	<div class="nd__bar"></div>
	<div class="nd__content">
		<div class="nd__head">
			<span class="nd__tag">Trigger</span>
		</div>
		<p class="nd__label">{data.label || 'Manual trigger'}</p>
		<div class="nd__foot">
			<span class="nd__chip">{data.triggerType || 'manual'}</span>
		</div>
	</div>
	<Handle type="source" position={Position.Right} />
</div>

<style lang="scss">
	.nd {
		display: flex;
		min-width: 200px;
		max-width: 260px;
		border-radius: $radius-xl;
		background: $success-50;
		border: 2px solid $success-500;
		box-shadow: $shadow-md;
		font-family: $font-sans;
		overflow: visible;
		transition: box-shadow $transition-fast, border-color $transition-fast;

		&:hover {
			box-shadow: $shadow-lg;
			border-color: $success-600;
		}

		&__bar {
			width: 4px;
			flex-shrink: 0;
			background: $success-500;
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
			color: $success-700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
		}

		&__label {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-900;
			line-height: $leading-tight;
		}

		&__foot {
			display: flex;
			gap: $space-2;
		}

		&__chip {
			font-family: $font-mono;
			font-size: 10px;
			color: $success-700;
			background: $success-100;
			padding: 2px 6px;
			border-radius: $radius-sm;
		}
	}
</style>
