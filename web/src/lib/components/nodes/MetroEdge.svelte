<script lang="ts">
	import { BaseEdge, getBezierPath, Position, type EdgeProps } from '@xyflow/svelte';

	let {
		sourceX,
		sourceY,
		targetX,
		targetY,
		label,
		labelStyle,
		markerStart,
		markerEnd,
		interactionWidth,
		style
	}: EdgeProps = $props();

	let [path, labelX, labelY] = $derived(getBezierPath({
		sourceX,
		sourceY,
		sourcePosition: Position.Right,
		targetX,
		targetY,
		targetPosition: Position.Left
	}));
</script>

<BaseEdge
	{path}
	{labelX}
	{labelY}
	{label}
	{labelStyle}
	{markerStart}
	{markerEnd}
	{interactionWidth}
	{style}
/>
