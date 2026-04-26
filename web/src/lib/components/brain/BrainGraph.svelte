<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
	import {
		forceSimulation,
		forceLink,
		forceManyBody,
		forceCenter,
		forceCollide,
		type Simulation,
		type SimulationNodeDatum,
		type SimulationLinkDatum
	} from 'd3-force';
	import type { BrainGraph, GraphNode, BrainLink, GraphCluster } from '$lib/api/types';

	interface Props {
		graph: BrainGraph | null;
		selectedFileId: string | null;
		searchHighlights: string[];
	}

	let { graph, selectedFileId, searchHighlights }: Props = $props();

	const dispatch = createEventDispatcher<{ select: { file_id: string } }>();

	let svgEl: SVGSVGElement;
	let containerEl: HTMLDivElement;
	let width = $state(600);
	let height = $state(400);

	interface SimNode extends SimulationNodeDatum {
		id: string;
		title: string;
		cluster_id: number;
		hashtags: string[];
		x: number;
		y: number;
	}

	interface SimLink extends SimulationLinkDatum<SimNode> {
		source: SimNode | string;
		target: SimNode | string;
	}

	let simNodes = $state<SimNode[]>([]);
	let simLinks = $state<SimLink[]>([]);
	let simulation: Simulation<SimNode, SimLink> | null = null;
	let tick = $state(0);


	function clusterColor(clusterId: number): string {
		if (!graph) return '#6366f1';
		const cluster = graph.clusters.find((c: GraphCluster) => c.id === clusterId);
		return cluster?.color ?? '#6366f1';
	}

	function buildSimulation(g: BrainGraph) {
		if (simulation) {
			simulation.stop();
			simulation = null;
		}

		const nodes: SimNode[] = g.nodes.map((n: GraphNode) => ({
			id: n.file_id,
			title: n.title,
			cluster_id: n.cluster_id,
			hashtags: n.hashtags ?? [],
			x: n.x ?? width / 2,
			y: n.y ?? height / 2
		}));

		const nodeById = new Map(nodes.map((n) => [n.id, n]));

		const links: SimLink[] = (g.links ?? [])
			.map((l: BrainLink) => ({
				source: l.source_file_id,
				target: l.target_file_id
			}))
			.filter((l) => nodeById.has(l.source as string) && nodeById.has(l.target as string));

		simNodes = nodes;
		simLinks = links;

		simulation = forceSimulation<SimNode, SimLink>(nodes)
			.force(
				'link',
				forceLink<SimNode, SimLink>(links)
					.id((d) => d.id)
					.strength(1)
			)
			.force('charge', forceManyBody<SimNode>().strength(-120))
			.force('center', forceCenter<SimNode>(width / 2, height / 2))
			.force('collide', forceCollide<SimNode>(18))
			.on('tick', () => {
				simNodes = [...nodes];
				tick++;
			})
			.on('end', () => {
				simNodes = [...nodes];
			});
	}

	function resizeObserver() {
		if (!containerEl) return;
		const rect = containerEl.getBoundingClientRect();
		width = rect.width || 600;
		height = rect.height || 400;
	}

	let ro: ResizeObserver | null = null;

	onMount(() => {
		resizeObserver();
		ro = new ResizeObserver(resizeObserver);
		ro.observe(containerEl);
	});

	onDestroy(() => {
		if (simulation) simulation.stop();
		if (ro) ro.disconnect();
		window.removeEventListener('pointermove', handleWindowPointerMove);
		window.removeEventListener('pointerup', handleWindowPointerUp);
		window.removeEventListener('pointercancel', handleWindowPointerUp);
	});

	$effect(() => {
		if (graph && graph.nodes.length > 0 && width > 0 && height > 0) {
			buildSimulation(graph);
		} else {
			simNodes = [];
			simLinks = [];
		}
	});

	function handleNodeClick(node: SimNode) {
		dispatch('select', { file_id: node.id });
	}

	const DRAG_THRESHOLD = 4;
	let dragState: {
		node: SimNode;
		pointerId: number;
		startX: number;
		startY: number;
		moved: boolean;
	} | null = null;

	function svgPoint(clientX: number, clientY: number): { x: number; y: number } {
		const rect = svgEl.getBoundingClientRect();
		return { x: clientX - rect.left, y: clientY - rect.top };
	}

	function handleNodePointerDown(e: PointerEvent, node: SimNode) {
		if (e.button !== 0) return;
		e.preventDefault();
		const p = svgPoint(e.clientX, e.clientY);
		node.fx = node.x;
		node.fy = node.y;
		dragState = { node, pointerId: e.pointerId, startX: p.x, startY: p.y, moved: false };
		if (simulation) simulation.alphaTarget(0.3).restart();
		// Attach listeners on window so {#key tick} remounts don't lose them.
		window.addEventListener('pointermove', handleWindowPointerMove);
		window.addEventListener('pointerup', handleWindowPointerUp);
		window.addEventListener('pointercancel', handleWindowPointerUp);
	}

	function handleWindowPointerMove(e: PointerEvent) {
		if (!dragState || e.pointerId !== dragState.pointerId) return;
		const p = svgPoint(e.clientX, e.clientY);
		if (!dragState.moved) {
			const dx = p.x - dragState.startX;
			const dy = p.y - dragState.startY;
			if (dx * dx + dy * dy >= DRAG_THRESHOLD * DRAG_THRESHOLD) {
				dragState.moved = true;
			}
		}
		dragState.node.fx = p.x;
		dragState.node.fy = p.y;
		dragState.node.x = p.x;
		dragState.node.y = p.y;
		// Bypass Svelte reactivity: write the transform straight to the DOM.
		// This avoids any subtlety with reactive tracking inside keyed each
		// blocks when the node object identity is preserved.
		applyNodePosition(dragState.node);
		applyLinksFor(dragState.node.id);
	}

	function handleWindowPointerUp(e: PointerEvent) {
		if (!dragState || e.pointerId !== dragState.pointerId) return;
		const node = dragState.node;
		const moved = dragState.moved;
		dragState = null;
		window.removeEventListener('pointermove', handleWindowPointerMove);
		window.removeEventListener('pointerup', handleWindowPointerUp);
		window.removeEventListener('pointercancel', handleWindowPointerUp);
		if (simulation) simulation.alphaTarget(0);
		// Node stays pinned where it was dropped (fx/fy retained).
		if (!moved) handleNodeClick(node);
	}

	function getNodeX(node: SimNode): number {
		return node.x ?? 0;
	}

	function getNodeY(node: SimNode): number {
		return node.y ?? 0;
	}

	// Returns a transform string that depends on `tick` so Svelte re-evaluates
	// the attribute whenever the simulation/drag bumps the tick counter,
	// even though node x/y are mutated on a non-reactive object.
	function nodeTransform(node: SimNode, _tick: number): string {
		return `translate(${node.x ?? 0},${node.y ?? 0})`;
	}

	const nodeRefs = new Map<string, SVGGElement>();

	function registerNodeRef(el: SVGGElement, id: string) {
		nodeRefs.set(id, el);
		return {
			destroy() {
				nodeRefs.delete(id);
			}
		};
	}

	function applyNodePosition(node: SimNode) {
		const el = nodeRefs.get(node.id);
		if (el) el.setAttribute('transform', `translate(${node.x ?? 0},${node.y ?? 0})`);
	}

	function applyLinksFor(nodeId: string) {
		if (!svgEl) return;
		const lines = svgEl.querySelectorAll<SVGLineElement>('line.brain-graph__link');
		for (let i = 0; i < lines.length; i++) {
			const link = simLinks[i];
			if (!link) continue;
			const src = link.source as SimNode;
			const tgt = link.target as SimNode;
			if (!src || !tgt) continue;
			if (typeof src !== 'object' || typeof tgt !== 'object') continue;
			if (src.id !== nodeId && tgt.id !== nodeId) continue;
			lines[i].setAttribute('x1', String(src.x ?? 0));
			lines[i].setAttribute('y1', String(src.y ?? 0));
			lines[i].setAttribute('x2', String(tgt.x ?? 0));
			lines[i].setAttribute('y2', String(tgt.y ?? 0));
		}
	}

	function getLinkSourceNode(link: SimLink): SimNode {
		return link.source as SimNode;
	}

	function getLinkTargetNode(link: SimLink): SimNode {
		return link.target as SimNode;
	}
</script>

<div class="brain-graph" bind:this={containerEl}>
	{#if !graph || graph.nodes.length === 0}
		<div class="brain-graph__empty">
			<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<circle cx="12" cy="5" r="2" />
				<circle cx="5" cy="19" r="2" />
				<circle cx="19" cy="19" r="2" />
				<path d="M12 7v4m0 0-5 6m5-6 5 6" />
			</svg>
			<p>No graph data yet</p>
		</div>
	{:else}
		<svg bind:this={svgEl} {width} {height} class="brain-graph__svg">
			<g class="links">
				{#each simLinks as link}
					{@const src = getLinkSourceNode(link)}
					{@const tgt = getLinkTargetNode(link)}
					{#if src && tgt}
						<line
							x1={getNodeX(src)}
							y1={getNodeY(src)}
							x2={getNodeX(tgt)}
							y2={getNodeY(tgt)}
							class="brain-graph__link"
						/>
					{/if}
				{/each}
			</g>
			<g class="nodes">
				{#each simNodes as node (node.id)}
					{@const isSelected = selectedFileId === node.id}
					{@const isHighlighted = searchHighlights.includes(node.id)}
					{@const color = clusterColor(node.cluster_id)}
					<g
						class="brain-graph__node-group"
						transform={nodeTransform(node, tick)}
						use:registerNodeRef={node.id}
						onpointerdown={(e) => handleNodePointerDown(e, node)}
						role="button"
						tabindex="0"
						onkeydown={(e) => e.key === 'Enter' && handleNodeClick(node)}
					>
						{#if isHighlighted}
							<circle r="16" fill={color} opacity="0.25" class="brain-graph__pulse" />
						{/if}
						{#if isSelected}
							<circle r="12" fill="none" stroke="#6366f1" stroke-width="3" />
						{/if}
						<circle
							r="8"
							fill={color}
							class="brain-graph__node"
							class:brain-graph__node--selected={isSelected}
						/>
						<text
							y="22"
							text-anchor="middle"
							class="brain-graph__node-label"
						>{node.title}</text>
					</g>
				{/each}
			</g>

			{#if graph}
				{@const labeled = graph.clusters.filter((c) => c.label && c.label.trim() !== '')}
				{#if labeled.length > 0}
					<g class="legend" transform="translate(12,12)">
						{#each labeled as cluster, i}
							<g transform="translate(0,{i * 20})">
								<circle cx="6" cy="6" r="5" fill={cluster.color} />
								<text x="16" y="10" class="brain-graph__legend-label">{cluster.label}</text>
							</g>
						{/each}
					</g>
				{/if}
			{/if}
		</svg>
	{/if}

</div>

<style lang="scss">
	.brain-graph {
		position: relative;
		width: 100%;
		height: 100%;
		background: $neutral-50;
		border-radius: $radius-xl;
		overflow: hidden;

		&__svg {
			display: block;
		}

		&__empty {
			@include flex-center;
			flex-direction: column;
			gap: $space-3;
			height: 100%;
			color: $neutral-400;
			font-size: $text-sm;

			p { margin: 0; }
		}

		&__link {
			stroke: $neutral-300;
			stroke-width: 1.5;
			opacity: 0.6;
		}

		&__node-group {
			cursor: grab;
			touch-action: none;
			outline: none;

			&:active {
				cursor: grabbing;
			}

			&:focus,
			&:focus-visible {
				outline: none;
			}

			&:hover circle.brain-graph__node {
				filter: brightness(1.15);
			}
		}

		&__node {
			transition: r 0.15s ease;

			&--selected {
				filter: brightness(1.2);
			}
		}

		&__pulse {
			animation: pulse-ring 1s ease-in-out infinite;
		}

		&__legend-label {
			font-size: 10px;
			fill: $neutral-500;
			font-family: $font-sans;
		}

		&__node-label {
			font-size: 11px;
			fill: $neutral-500;
			font-family: $font-sans;
			pointer-events: none;
			user-select: none;
		}
	}

	@keyframes pulse-ring {
		0%, 100% { opacity: 0.25; r: 12; }
		50% { opacity: 0.5; r: 16; }
	}
</style>
