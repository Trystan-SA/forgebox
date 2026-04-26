<script lang="ts">
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
	import {
		forceSimulation,
		forceManyBody,
		forceLink,
		type Simulation,
		type SimulationNodeDatum
	} from 'd3-force';
	import type { BrainGraph, GraphNode, BrainLink, GraphCluster } from '$lib/api/types';

	interface Props {
		graph: BrainGraph | null;
		selectedFileId: string | null;
		searchHighlights: string[];
	}

	let { graph, selectedFileId, searchHighlights }: Props = $props();

	const dispatch = createEventDispatcher<{
		select: { file_id: string };
		deselect: Record<string, never>;
	}>();

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

	interface SimLink {
		sourceId: string;
		targetId: string;
	}

	// d3-force mutates link.source/link.target into node references after the
	// first tick, so we keep a separate array just for the simulation and don't
	// touch our SimLink objects (which the template indexes by sourceId/targetId).
	type ForceLink = { source: string | SimNode; target: string | SimNode };

	// Reassigning these arrays signals the templates; per-element mutations
	// of .x/.y during drag or layout bypass reactivity and go straight to
	// the DOM via setAttribute.
	let simNodes = $state.raw<SimNode[]>([]);
	let simLinks = $state.raw<SimLink[]>([]);

	const nodeById = new Map<string, SimNode>();

	function clusterColor(clusterId: number): string {
		if (!graph) return '#6366f1';
		const cluster = graph.clusters.find((c: GraphCluster) => c.id === clusterId);
		return cluster?.color ?? '#6366f1';
	}

	function buildSimulation(g: BrainGraph): { nodes: SimNode[]; links: SimLink[] } {
		const nodes: SimNode[] = g.nodes.map((n: GraphNode) => ({
			id: n.file_id,
			title: n.title,
			cluster_id: n.cluster_id,
			hashtags: n.hashtags ?? [],
			x: n.x ?? width / 2,
			y: n.y ?? height / 2
		}));

		const nodeIds = new Set(nodes.map((n) => n.id));

		const links: SimLink[] = (g.links ?? [])
			.filter((l) => nodeIds.has(l.source_file_id) && nodeIds.has(l.target_file_id))
			.map((l: BrainLink): SimLink => ({
				sourceId: l.source_file_id,
				targetId: l.target_file_id
			}));

		simNodes = nodes;
		simLinks = links;
		nodeById.clear();
		for (const n of nodes) nodeById.set(n.id, n);
		return { nodes, links };
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
		if (ro) ro.disconnect();
		simulation?.stop();
		window.removeEventListener('pointermove', handleWindowPointerMove);
		window.removeEventListener('pointerup', handleWindowPointerUp);
		window.removeEventListener('pointercancel', handleWindowPointerUp);
		window.removeEventListener('pointermove', handlePanPointerMove);
		window.removeEventListener('pointerup', handlePanPointerUp);
		window.removeEventListener('pointercancel', handlePanPointerUp);
	});

	// Rebuild only when the graph data itself changes — not on every resize,
	// which would otherwise reset positions while the user is dragging.
	// startLayout is called with the freshly-built arrays directly so the
	// effect doesn't read simNodes/simLinks (which it just wrote) — that
	// would self-trigger and exceed Svelte's update-depth guard.
	$effect(() => {
		if (graph && graph.nodes.length > 0) {
			const { nodes, links } = buildSimulation(graph);
			startLayout(nodes, links);
		} else {
			simulation?.stop();
			simulation = null;
			simNodes = [];
			simLinks = [];
			nodeById.clear();
		}
	});

	let simulation: Simulation<SimNode, ForceLink> | null = null;

	function startLayout(nodes: SimNode[], links: SimLink[]) {
		simulation?.stop();
		if (nodes.length < 2) return;
		const forceLinks: ForceLink[] = links.map((l) => ({ source: l.sourceId, target: l.targetId }));
		simulation = forceSimulation<SimNode, ForceLink>(nodes)
			.force('charge', forceManyBody<SimNode>().strength(-200).distanceMax(160))
			.force(
				'link',
				forceLink<SimNode, ForceLink>(forceLinks)
					.id((d) => d.id)
					.distance(90)
					.strength(0.18)
			)
			.alphaMin(0.04)
			.on('tick', applyAllPositions);
	}

	function handleNodeClick(node: SimNode) {
		dispatch('select', { file_id: node.id });
	}

	const DRAG_THRESHOLD = 4;
	const MIN_ZOOM = 0.2;
	const MAX_ZOOM = 3;

	let panX = $state(0);
	let panY = $state(0);
	let zoom = $state(1);
	let isPanning = $state(false);

	let dragState: {
		node: SimNode;
		pointerId: number;
		startX: number;
		startY: number;
		moved: boolean;
	} | null = null;

	let panState: {
		pointerId: number;
		startClientX: number;
		startClientY: number;
		startPanX: number;
		startPanY: number;
		moved: boolean;
	} | null = null;

	function clientToScreen(clientX: number, clientY: number): { x: number; y: number } {
		const rect = svgEl.getBoundingClientRect();
		return { x: clientX - rect.left, y: clientY - rect.top };
	}

	function svgPoint(clientX: number, clientY: number): { x: number; y: number } {
		const s = clientToScreen(clientX, clientY);
		return { x: (s.x - panX) / zoom, y: (s.y - panY) / zoom };
	}

	function handleNodePointerDown(e: PointerEvent, node: SimNode) {
		if (e.button !== 0) return;
		e.preventDefault();
		// Stop any running auto-layout so it doesn't fight the user.
		simulation?.stop();
		const p = svgPoint(e.clientX, e.clientY);
		dragState = { node, pointerId: e.pointerId, startX: p.x, startY: p.y, moved: false };
		// Attach listeners on window so re-renders don't lose them.
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
		dragState.node.x = p.x;
		dragState.node.y = p.y;
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
		if (!moved) {
			handleNodeClick(node);
		} else {
			// Pin the manually placed node (d3 holds nodes at fx/fy) so the
			// upcoming layout pass settles the rest of the graph around it
			// instead of pulling it back.
			node.fx = node.x;
			node.fy = node.y;
			startLayout(simNodes, simLinks);
		}
	}

	function getNodeX(node: SimNode): number {
		return node.x ?? 0;
	}

	function getNodeY(node: SimNode): number {
		return node.y ?? 0;
	}

	function nodeTransform(node: SimNode): string {
		return `translate(${node.x ?? 0},${node.y ?? 0})`;
	}

	const nodeRefs = new Map<string, SVGGElement>();
	const linkRefs = new Map<string, SVGLineElement>();

	function linkKey(sourceId: string, targetId: string): string {
		return `${sourceId}|${targetId}`;
	}

	function registerNodeRef(el: SVGGElement, id: string) {
		nodeRefs.set(id, el);
		return {
			destroy() {
				nodeRefs.delete(id);
			}
		};
	}

	function registerLinkRef(el: SVGLineElement, key: string) {
		linkRefs.set(key, el);
		return {
			destroy() {
				linkRefs.delete(key);
			}
		};
	}

	function applyNodePosition(node: SimNode) {
		const el = nodeRefs.get(node.id);
		if (el) el.setAttribute('transform', `translate(${node.x ?? 0},${node.y ?? 0})`);
	}

	function applyLink(link: SimLink) {
		const el = linkRefs.get(linkKey(link.sourceId, link.targetId));
		if (!el) return;
		const src = nodeById.get(link.sourceId);
		const tgt = nodeById.get(link.targetId);
		if (!src || !tgt) return;
		el.setAttribute('x1', String(src.x ?? 0));
		el.setAttribute('y1', String(src.y ?? 0));
		el.setAttribute('x2', String(tgt.x ?? 0));
		el.setAttribute('y2', String(tgt.y ?? 0));
	}

	function applyLinksFor(nodeId: string) {
		for (const link of simLinks) {
			if (link.sourceId === nodeId || link.targetId === nodeId) applyLink(link);
		}
	}

	function applyAllPositions() {
		for (const node of simNodes) applyNodePosition(node);
		for (const link of simLinks) applyLink(link);
	}

	function handleBackgroundPointerDown(e: PointerEvent) {
		if (e.target !== svgEl) return;
		if (e.button !== 0) return;
		e.preventDefault();
		panState = {
			pointerId: e.pointerId,
			startClientX: e.clientX,
			startClientY: e.clientY,
			startPanX: panX,
			startPanY: panY,
			moved: false
		};
		window.addEventListener('pointermove', handlePanPointerMove);
		window.addEventListener('pointerup', handlePanPointerUp);
		window.addEventListener('pointercancel', handlePanPointerUp);
	}

	function handlePanPointerMove(e: PointerEvent) {
		if (!panState || e.pointerId !== panState.pointerId) return;
		const dx = e.clientX - panState.startClientX;
		const dy = e.clientY - panState.startClientY;
		if (!panState.moved && dx * dx + dy * dy >= DRAG_THRESHOLD * DRAG_THRESHOLD) {
			panState.moved = true;
		}
		if (panState.moved) {
			isPanning = true;
			panX = panState.startPanX + dx;
			panY = panState.startPanY + dy;
		}
	}

	function handlePanPointerUp(e: PointerEvent) {
		if (!panState || e.pointerId !== panState.pointerId) return;
		const moved = panState.moved;
		panState = null;
		isPanning = false;
		window.removeEventListener('pointermove', handlePanPointerMove);
		window.removeEventListener('pointerup', handlePanPointerUp);
		window.removeEventListener('pointercancel', handlePanPointerUp);
		if (!moved) dispatch('deselect', {} as never);
	}

	function handleWheel(e: WheelEvent) {
		e.preventDefault();
		const screen = clientToScreen(e.clientX, e.clientY);
		const worldX = (screen.x - panX) / zoom;
		const worldY = (screen.y - panY) / zoom;
		const factor = e.deltaY < 0 ? 1.1 : 1 / 1.1;
		const newZoom = Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, zoom * factor));
		panX = screen.x - worldX * newZoom;
		panY = screen.y - worldY * newZoom;
		zoom = newZoom;
	}

	function findNode(id: string): SimNode | undefined {
		return nodeById.get(id);
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
		<svg
			bind:this={svgEl}
			{width}
			{height}
			class="brain-graph__svg"
			class:brain-graph__svg--panning={isPanning}
			onpointerdown={handleBackgroundPointerDown}
			onwheel={handleWheel}
		>
			<g class="brain-graph__viewport" transform="translate({panX},{panY}) scale({zoom})">
				<g class="links">
					{#each simLinks as link (linkKey(link.sourceId, link.targetId))}
						{@const src = findNode(link.sourceId)}
						{@const tgt = findNode(link.targetId)}
						{#if src && tgt}
							<line
								x1={getNodeX(src)}
								y1={getNodeY(src)}
								x2={getNodeX(tgt)}
								y2={getNodeY(tgt)}
								class="brain-graph__link"
								use:registerLinkRef={linkKey(link.sourceId, link.targetId)}
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
							transform={nodeTransform(node)}
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
			cursor: grab;
			touch-action: none;

			&--panning {
				cursor: grabbing;
			}
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
