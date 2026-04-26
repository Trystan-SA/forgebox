// Force-directed layout engine for the brain graph.
//
// Each frame computes pairwise repulsion (within REPULSION_RADIUS) and
// spring attraction along links (toward LINK_DISTANCE), then moves every
// node a fraction (LAYOUT_STEP) of the way toward its target. The loop
// stops when the average per-node motion drops below LAYOUT_ENERGY_STOP.
//
// The engine mutates the node objects directly (not through any reactive
// proxy) and notifies a single onTick callback per frame so the consumer
// can write positions to the DOM without triggering reactivity churn.

export interface LayoutNode {
	id: string;
	x: number;
	y: number;
	/** When true, the node still exerts forces on others but does not move. */
	pinned?: boolean;
}

export interface LayoutLink {
	sourceId: string;
	targetId: string;
}

export interface LayoutOptions {
	/** Pairwise repulsion zone radius (px). */
	repulsionRadius?: number;
	/** Comfortable distance between linked nodes (px). */
	linkDistance?: number;
	/** Spring strength along links. */
	linkStrength?: number;
	/** Fraction of the target displacement applied per frame (0..1). */
	step?: number;
	/** Average per-node motion (px) below which the loop stops. */
	energyStop?: number;
	/** Hard cap on iterations to guarantee termination. */
	maxIterations?: number;
}

const DEFAULTS: Required<LayoutOptions> = {
	repulsionRadius: 120,
	linkDistance: 90,
	linkStrength: 0.18,
	step: 0.12,
	energyStop: 0.04,
	maxIterations: 800
};

export class GraphLayout {
	private nodes: LayoutNode[] = [];
	private links: LayoutLink[] = [];
	private opts: Required<LayoutOptions>;
	private frameId: number | null = null;
	private iter = 0;
	private onTickCb: (() => void) | null = null;

	constructor(options: LayoutOptions = {}) {
		this.opts = { ...DEFAULTS, ...options };
	}

	setData(nodes: LayoutNode[], links: LayoutLink[]): void {
		this.nodes = nodes;
		this.links = links;
	}

	onTick(cb: () => void): void {
		this.onTickCb = cb;
	}

	start(): void {
		this.cancel();
		if (this.nodes.length < 2) return;
		this.iter = 0;
		const tick = () => {
			this.iter++;
			const energy = this.step();
			this.onTickCb?.();
			if (energy < this.opts.energyStop || this.iter >= this.opts.maxIterations) {
				this.frameId = null;
				return;
			}
			this.frameId = requestAnimationFrame(tick);
		};
		this.frameId = requestAnimationFrame(tick);
	}

	cancel(): void {
		if (this.frameId !== null) {
			cancelAnimationFrame(this.frameId);
			this.frameId = null;
		}
	}

	get running(): boolean {
		return this.frameId !== null;
	}

	/** Pin or unpin a node. Pinned nodes still exert forces but do not move. */
	setPinned(id: string, pinned: boolean): void {
		const node = this.nodes.find((n) => n.id === id);
		if (node) node.pinned = pinned;
	}

	private step(): number {
		const { repulsionRadius, linkDistance, linkStrength, step: stepFraction } = this.opts;
		const nodeCount = this.nodes.length;
		const forceX = new Array<number>(nodeCount).fill(0);
		const forceY = new Array<number>(nodeCount).fill(0);

		// Pairwise repulsion: nodes inside the repulsion zone push each other away.
		for (let indexA = 0; indexA < nodeCount; indexA++) {
			for (let indexB = indexA + 1; indexB < nodeCount; indexB++) {
				const nodeA = this.nodes[indexA];
				const nodeB = this.nodes[indexB];
				let deltaX = nodeB.x - nodeA.x;
				let deltaY = nodeB.y - nodeA.y;
				let distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
				if (distance < 0.01) {
					// Coincident: nudge apart with a tiny random offset.
					deltaX = Math.random() - 0.5;
					deltaY = Math.random() - 0.5;
					distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY) || 1;
				}
				if (distance < repulsionRadius) {
					const pushFactor = (repulsionRadius - distance) / distance;
					forceX[indexA] -= deltaX * pushFactor;
					forceY[indexA] -= deltaY * pushFactor;
					forceX[indexB] += deltaX * pushFactor;
					forceY[indexB] += deltaY * pushFactor;
				}
			}
		}

		// Spring attraction: linked nodes pull toward each other when too far apart
		// and gently push apart when too close.
		const indexById = new Map<string, number>();
		for (let i = 0; i < nodeCount; i++) indexById.set(this.nodes[i].id, i);
		for (const link of this.links) {
			const sourceIndex = indexById.get(link.sourceId);
			const targetIndex = indexById.get(link.targetId);
			if (sourceIndex === undefined || targetIndex === undefined) continue;
			const sourceNode = this.nodes[sourceIndex];
			const targetNode = this.nodes[targetIndex];
			const deltaX = targetNode.x - sourceNode.x;
			const deltaY = targetNode.y - sourceNode.y;
			const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY) || 0.01;
			const pullFactor = ((distance - linkDistance) * linkStrength) / distance;
			forceX[sourceIndex] += deltaX * pullFactor;
			forceY[sourceIndex] += deltaY * pullFactor;
			forceX[targetIndex] -= deltaX * pullFactor;
			forceY[targetIndex] -= deltaY * pullFactor;
		}

		// Move each node a fraction of the way toward its target.
		// Pinned nodes still influence others (their forces accumulated above)
		// but they themselves don't move.
		let totalMotion = 0;
		for (let i = 0; i < nodeCount; i++) {
			const node = this.nodes[i];
			if (node.pinned) continue;
			const moveX = forceX[i] * stepFraction;
			const moveY = forceY[i] * stepFraction;
			node.x += moveX;
			node.y += moveY;
			totalMotion += Math.abs(moveX) + Math.abs(moveY);
		}
		return totalMotion / nodeCount;
	}
}
