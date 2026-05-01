<!--
	Tiny SVG line chart for short timeseries (currently last-7-days
	provider usage). Pure presentational — the parent supplies the values
	and labels. Renders an inline <svg> sized via CSS so it stretches to
	the card width.
-->
<script lang="ts">
	interface Props {
		points: number[];
		labels?: string[];
		ariaLabel?: string;
	}

	let { points, labels = [], ariaLabel = 'Usage trend' }: Props = $props();

	const VIEW_W = 200;
	const VIEW_H = 60;
	const PADDING = 4;

	const max = $derived(Math.max(1, ...points));

	const polyline = $derived.by(() => {
		if (points.length === 0) return '';
		const stepX = (VIEW_W - PADDING * 2) / Math.max(1, points.length - 1);
		const usableH = VIEW_H - PADDING * 2;
		return points
			.map((v, i) => {
				const x = PADDING + i * stepX;
				const y = PADDING + usableH * (1 - v / max);
				return `${x.toFixed(1)},${y.toFixed(1)}`;
			})
			.join(' ');
	});

	const areaPath = $derived.by(() => {
		if (!polyline) return '';
		const stepX = (VIEW_W - PADDING * 2) / Math.max(1, points.length - 1);
		const baseline = VIEW_H - PADDING;
		const start = `${PADDING.toFixed(1)},${baseline.toFixed(1)}`;
		const end = `${(PADDING + (points.length - 1) * stepX).toFixed(1)},${baseline.toFixed(1)}`;
		return `M${start} L${polyline.split(' ').join(' L')} L${end} Z`;
	});
</script>

<div class="sparkline" aria-label={ariaLabel}>
	<svg viewBox={`0 0 ${VIEW_W} ${VIEW_H}`} preserveAspectRatio="none" role="img">
		<path d={areaPath} class="sparkline__area" />
		<polyline points={polyline} class="sparkline__line" />
	</svg>
	{#if labels.length > 0}
		<div class="sparkline__labels">
			{#each labels as l}
				<span>{l}</span>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.sparkline {
		display: flex;
		flex-direction: column;
		gap: $space-1;
		width: 100%;

		svg {
			width: 100%;
			height: 56px;
			display: block;
		}

		&__line {
			fill: none;
			stroke: $primary-600;
			stroke-width: 1.5;
			stroke-linecap: round;
			stroke-linejoin: round;
			vector-effect: non-scaling-stroke;
		}

		&__area {
			fill: $primary-100;
			opacity: 0.6;
		}

		&__labels {
			display: flex;
			justify-content: space-between;
			font-size: 10px;
			color: $neutral-400;
			font-variant-numeric: tabular-nums;
		}
	}
</style>
