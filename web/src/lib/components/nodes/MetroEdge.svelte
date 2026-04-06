<script lang="ts">
	import { BaseEdge, type EdgeProps } from '@xyflow/svelte';

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

	const pad = 20;
	const loopGap = 60;
	const r = 15; // diagonal radius at corners

	function clamp(val: number, max: number) {
		return Math.min(val, max);
	}

	let path = $derived.by(() => {
		const absDy = Math.abs(targetY - sourceY);
		const diagEnd = sourceX + pad + absDy;

		// Normal forward: target is far enough to the right
		if (diagEnd < targetX - pad) {
			return `M ${sourceX},${sourceY} L ${sourceX + pad},${sourceY} L ${diagEnd},${targetY} L ${targetX},${targetY}`;
		}

		// Route around — go through open space
		const routeAbove = targetY > sourceY;
		const sign = routeAbove ? -1 : 1; // -1 = up, +1 = down
		const clearY = routeAbove
			? Math.min(sourceY, targetY) - loopGap
			: Math.max(sourceY, targetY) + loopGap;

		const sx = sourceX + pad;  // after right pad
		const tx = targetX - pad;  // before left pad into target
		const vertSource = Math.abs(clearY - sourceY);
		const vertTarget = Math.abs(clearY - targetY);

		// Diagonal radius capped so it doesn't exceed half the available segment
		const r1 = clamp(r, vertSource / 2);
		const r2 = clamp(r, vertTarget / 2);
		const rh = clamp(r, Math.abs(tx - sx) / 2);
		const ra = Math.min(r1, rh);
		const rb = Math.min(r2, rh);

		return [
			`M ${sourceX},${sourceY}`,
			`L ${sx},${sourceY}`,
			// corner: horizontal to vertical
			`L ${sx + ra},${sourceY + sign * ra}`,
			// vertical
			`L ${sx + ra},${clearY - sign * ra}`,
			// corner: vertical to horizontal
			`L ${sx},${clearY}`,
			// horizontal
			`L ${tx},${clearY}`,
			// corner: horizontal to vertical
			`L ${tx - rb},${clearY + sign * rb}`,
			// vertical
			`L ${tx - rb},${targetY - sign * rb}`,
			// corner: vertical to horizontal
			`L ${tx},${targetY}`,
			`L ${targetX},${targetY}`
		].join(' ');
	});

	let labelX = $derived((sourceX + targetX) / 2);
	let labelY = $derived((sourceY + targetY) / 2);
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
