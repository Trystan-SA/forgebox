<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { getAutomation, updateAutomation, getAutomationYaml } from '$lib/api/client';
	import type { Automation } from '$lib/api/types';
	import {
		SvelteFlow,
		Controls,
		Background,
		MiniMap,
		addEdge,
		type Node,
		type Edge,
		type Connection,
		type NodeTypes,
		type EdgeTypes
	} from '@xyflow/svelte';
	import '@xyflow/svelte/dist/style.css';

	import TriggerNode from '$lib/components/nodes/TriggerNode.svelte';
	import AIStepNode from '$lib/components/nodes/AIStepNode.svelte';
	import ToolNode from '$lib/components/nodes/ToolNode.svelte';
	import ConditionNode from '$lib/components/nodes/ConditionNode.svelte';
	import SwitchNode from '$lib/components/nodes/SwitchNode.svelte';
	import MetroEdge from '$lib/components/nodes/MetroEdge.svelte';
	import NodeConfigPanel from '$lib/components/nodes/NodeConfigPanel.svelte';
	import NodeContextMenu from '$lib/components/nodes/NodeContextMenu.svelte';
	import YamlPreviewModal from '$lib/components/YamlPreviewModal.svelte';

	const nodeTypes: NodeTypes = {
		trigger: TriggerNode,
		aiStep: AIStepNode,
		tool: ToolNode,
		condition: ConditionNode,
		switch: SwitchNode
	} as unknown as NodeTypes;

	const edgeTypes: EdgeTypes = {
		metro: MetroEdge
	} as unknown as EdgeTypes;

	let automation = $state<Automation | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	let nodes = $state.raw<Node[]>([]);
	let edges = $state.raw<Edge[]>([]);

	const defaultNodes: Node[] = [];

	onMount(async () => {
		const id = page.params.id;
		try {
			automation = await getAutomation(id);
			try {
				nodes = JSON.parse(automation.nodes);
				edges = JSON.parse(automation.edges);
			} catch {
				nodes = [];
				edges = [];
			}
			if (nodes.length === 0) {
				nodes = [...defaultNodes];
			}
		} catch {
			error = 'Workflow not found';
		} finally {
			loading = false;
		}
	});

	let selectedNodeId = $state<string | null>(null);
	const selectedNode = $derived(nodes.find((n) => n.id === selectedNodeId) ?? null);

	function updateNodeData(id: string, data: Record<string, any>) {
		nodes = nodes.map((n) => (n.id === id ? { ...n, data } : n));
	}

	let contextMenu = $state<{ x: number; y: number } | null>(null);
	let canvasEl: HTMLDivElement;
	let searchQuery = $state('');
	let searchInput: HTMLInputElement;
	let pendingConnection = $state<{ sourceId: string; sourceHandle: string | null } | null>(null);

	interface NodeOption {
		type: string;
		label: string;
		icon: string;
		desc: string;
		data: Record<string, string>;
	}

	interface NodeCategory {
		label: string;
		items: NodeOption[];
	}

	const nodeCategories: NodeCategory[] = [
		{
			label: 'Triggers',
			items: [
				{ type: 'trigger', label: 'Manual', icon: '', desc: 'Start manually from the dashboard', data: { triggerType: 'manual' } },
				{ type: 'trigger', label: 'Webhook', icon: '', desc: 'Triggered by an HTTP POST request', data: { triggerType: 'webhook' } },
				{ type: 'trigger', label: 'Schedule', icon: '', desc: 'Run on a cron schedule', data: { triggerType: 'schedule' } }
			]
		},
		{
			label: 'AI',
			items: [
				{ type: 'aiStep', label: 'Claude', icon: '', desc: 'Anthropic Claude completion', data: { provider: 'anthropic', model: 'claude-sonnet' } },
				{ type: 'aiStep', label: 'GPT-4o', icon: '', desc: 'OpenAI GPT-4o completion', data: { provider: 'openai', model: 'gpt-4o' } },
				{ type: 'aiStep', label: 'Ollama', icon: '', desc: 'Local LLM via Ollama', data: { provider: 'ollama', model: 'llama3' } }
			]
		},
		{
			label: 'Tools',
			items: [
				{ type: 'tool', label: 'Shell', icon: '', desc: 'Execute a shell command', data: { tool: 'bash' } },
				{ type: 'tool', label: 'HTTP Request', icon: '', desc: 'Fetch data from a URL', data: { tool: 'web_fetch' } },
				{ type: 'tool', label: 'File Read', icon: '', desc: 'Read a file from the VM', data: { tool: 'file_read' } },
				{ type: 'tool', label: 'File Write', icon: '', desc: 'Write a file to the VM', data: { tool: 'file_write' } }
			]
		},
		{
			label: 'Flow',
			items: [
				{ type: 'condition', label: 'If / Else', icon: '', desc: 'Branch on a true/false condition', data: {} },
				{ type: 'condition', label: 'Boolean Check', icon: '', desc: 'Check if a value is true or false', data: { operator: 'is_true', valueType: 'boolean' } },
				{ type: 'condition', label: 'String Compare', icon: '', desc: 'Compare text values', data: { operator: 'equals', valueType: 'string' } },
				{ type: 'condition', label: 'Number Compare', icon: '', desc: 'Compare numeric values', data: { operator: 'gt', valueType: 'number' } },
				{ type: 'switch', label: 'Switch', icon: '', desc: 'Route to different branches by value', data: {} }
			]
		}
	];

	let expandedCategory = $state<string | null>(null);

	const isSearching = $derived(searchQuery.trim().length > 0);

	const filteredCategories = $derived(
		nodeCategories
			.map((cat) => ({
				...cat,
				items: isSearching
					? cat.items.filter((o) => o.label.toLowerCase().includes(searchQuery.toLowerCase()))
					: cat.items
			}))
			.filter((cat) => cat.items.length > 0)
	);

	function autoFocus(node: HTMLElement) {
		requestAnimationFrame(() => node.focus());
	}

	const allFilteredItems = $derived(filteredCategories.flatMap((c) => c.items));

	function handleContextMenu(e: MouseEvent) {
		const target = e.target as HTMLElement | null;
		if (target?.closest('.svelte-flow__node')) {
			// Node context menu is handled by onnodecontextmenu.
			return;
		}
		e.preventDefault();
		nodeContextMenu = null;
		const rect = canvasEl.getBoundingClientRect();
		searchQuery = '';
		expandedCategory = null;
		contextMenu = {
			x: e.clientX - rect.left,
			y: e.clientY - rect.top
		};
		tick().then(() => searchInput?.focus());
	}

	let nodeContextMenu = $state<{ x: number; y: number; nodeId: string } | null>(null);

	function openNodeContextMenu(nodeId: string, clientX: number, clientY: number) {
		const rect = canvasEl.getBoundingClientRect();
		contextMenu = null;
		nodeContextMenu = {
			nodeId,
			x: clientX - rect.left,
			y: clientY - rect.top
		};
	}

	function toggleNodeDisabled(nodeId: string) {
		nodes = nodes.map((n) => {
			if (n.id !== nodeId) return n;
			const disabled = !n.data?.disabled;
			return {
				...n,
				data: { ...n.data, disabled },
				className: disabled ? 'nd-disabled' : ''
			};
		});
		nodeContextMenu = null;
	}

	function deleteNode(nodeId: string) {
		nodes = nodes.filter((n) => n.id !== nodeId);
		edges = edges.filter((e) => e.source !== nodeId && e.target !== nodeId);
		if (selectedNodeId === nodeId) selectedNodeId = null;
		nodeContextMenu = null;
	}

	const nodeContextTarget = $derived(
		nodeContextMenu ? nodes.find((n) => n.id === nodeContextMenu!.nodeId) ?? null : null
	);

	function handleCanvasKeydown(e: KeyboardEvent) {
		if (e.key === ' ') {
			e.preventDefault();
			nodeContextMenu = null;
			if (contextMenu) {
				contextMenu = null;
			} else {
				const rect = canvasEl.getBoundingClientRect();
				searchQuery = '';
				expandedCategory = null;
				contextMenu = {
					x: rect.width / 2,
					y: rect.height / 2
				};
				tick().then(() => searchInput?.focus());
			}
		}
		if (e.key === 'Escape') {
			if (nodeContextMenu) { nodeContextMenu = null; return; }
			if (selectedNodeId) { selectedNodeId = null; return; }
			contextMenu = null;
		}
	}

	function getViewportCenter(): { x: number; y: number } {
		const rect = canvasEl?.getBoundingClientRect();
		if (!rect) return { x: 250, y: 150 };

		const viewport = document.querySelector('.svelte-flow__viewport');
		if (!viewport) return { x: 250, y: 150 };

		const transform = getComputedStyle(viewport).transform;
		if (!transform || transform === 'none') return { x: rect.width / 2, y: rect.height / 2 };

		const matrix = new DOMMatrix(transform);
		const scale = matrix.a;
		const tx = matrix.e;
		const ty = matrix.f;

		return {
			x: (rect.width / 2 - tx) / scale,
			y: (rect.height / 2 - ty) / scale
		};
	}

	function addNodeFromMenu(type: string, label: string, data: Record<string, string>) {
		const id = crypto.randomUUID().slice(0, 8);
		const position = getViewportCenter();

		nodes = [
			...nodes,
			{
				id,
				type,
				position,
				data: { label, ...data }
			}
		];

		if (pendingConnection) {
			edges = addEdge(
				{
					source: pendingConnection.sourceId,
					target: id,
					sourceHandle: pendingConnection.sourceHandle,
					targetHandle: null
				},
				edges
			);
			pendingConnection = null;
		}

		contextMenu = null;
		expandedCategory = null;
	}

	async function handleSave() {
		if (!automation) return;
		saving = true;
		try {
			await updateAutomation(automation.id, {
				nodes: JSON.stringify(nodes),
				edges: JSON.stringify(edges)
			});
		} catch (err) {
			alert(err instanceof Error ? err.message : 'Save failed');
		} finally {
			saving = false;
		}
	}

	let yamlOpen = $state(false);
	let yamlText = $state('');
	let yamlLoading = $state(false);
	let yamlError = $state<string | null>(null);

	async function openYamlPreview() {
		if (!automation) return;
		yamlOpen = true;
		yamlLoading = true;
		yamlError = null;
		try {
			yamlText = await getAutomationYaml(automation.id);
		} catch (err) {
			yamlError = err instanceof Error ? err.message : 'Failed to load YAML';
		} finally {
			yamlLoading = false;
		}
	}
</script>

{#if loading}
	<div class="loading">Loading editor...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if automation}
	<div class="editor">
		<div class="editor__toolbar">
			<div class="editor__left">
				<a href="/automations" class="editor__back">
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
				</a>
				<h2 class="editor__title">{automation.name}</h2>
			</div>
			<div class="editor__right">
				<button class="btn-secondary editor__yaml" onclick={openYamlPreview} title="Preview automation as YAML">
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="16 18 22 12 16 6" /><polyline points="8 6 2 12 8 18" /></svg>
					<span>YAML</span>
				</button>
				<button class="btn-primary editor__save" onclick={handleSave} disabled={saving}>
					{saving ? 'Saving...' : 'Save'}
				</button>
			</div>
		</div>

		<div class="editor__main">
		<div class="editor__canvas" bind:this={canvasEl} onkeydown={handleCanvasKeydown} oncontextmenu={handleContextMenu} tabindex="-1">
			<SvelteFlow
				bind:nodes
				bind:edges
				{nodeTypes}
				{edgeTypes}
				fitView
				zoomOnScroll
				panOnDrag
				selectionOnDrag
				deleteKey="Delete"
				snapGrid={[20, 20]}
				defaultEdgeOptions={{ type: 'metro' }}
				onnodeclick={({ node }) => { selectedNodeId = node.id; }}
				onnodecontextmenu={({ event, node }) => {
					event.preventDefault();
					event.stopPropagation();
					openNodeContextMenu(node.id, (event as MouseEvent).clientX, (event as MouseEvent).clientY);
				}}
				onpaneclick={() => { selectedNodeId = null; }}
				onconnectstart={(_, params) => {
					if (params.nodeId) {
						pendingConnection = { sourceId: params.nodeId, sourceHandle: params.handleId ?? null };
					}
				}}
				onconnect={(connection: Connection) => {
					pendingConnection = null;
				}}
				onconnectend={(event) => {
					if (!pendingConnection) return;
					const target = (event as MouseEvent).target as HTMLElement;
					if (target?.closest('.svelte-flow__node')) return;

					const rect = canvasEl.getBoundingClientRect();
					const e = event as MouseEvent;
					searchQuery = '';
					expandedCategory = null;
					contextMenu = {
						x: e.clientX - rect.left,
						y: e.clientY - rect.top
					};
					tick().then(() => searchInput?.focus());
				}}
			>
				<Controls />
				<Background />
				<MiniMap />
			</SvelteFlow>

			{#if nodes.length === 0}
				<div class="empty-canvas">
					<div class="empty-canvas__card">
						<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
							<circle cx="12" cy="12" r="10" />
							<line x1="12" y1="8" x2="12" y2="16" />
							<line x1="8" y1="12" x2="16" y2="12" />
						</svg>
						<h3>No trigger yet</h3>
						<p>Press <kbd>Space</kbd> or <kbd>Right-click</kbd> to add a trigger and start building your workflow.</p>
					</div>
				</div>
			{/if}

			{#if contextMenu}
				<button class="ctx-overlay" onclick={() => { contextMenu = null; expandedCategory = null; pendingConnection = null; }}></button>
				<div class="ctx-menu" style="left: {contextMenu.x}px; top: {contextMenu.y}px;">
					<div class="ctx-menu__head">
						<div class="ctx-menu__search">
							<svg class="ctx-menu__search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
							<input
								bind:this={searchInput}
								bind:value={searchQuery}
								use:autoFocus
								type="text"
								placeholder="Search nodes..."
								onkeydown={(e) => {
									if (e.key === 'Escape') { contextMenu = null; expandedCategory = null; canvasEl?.focus(); }
									if (e.key === 'Enter' && allFilteredItems.length === 1) {
										const opt = allFilteredItems[0];
										addNodeFromMenu(opt.type, opt.label, opt.data);
									}
								}}
							/>
							<kbd class="ctx-menu__kbd">esc</kbd>
						</div>
					</div>
					<div class="ctx-menu__body">
						{#each filteredCategories as cat}
							{#if isSearching}
								<div class="ctx-menu__group ctx-menu__group--{cat.label.toLowerCase()}">
									<span class="ctx-menu__group-label">{cat.label}</span>
									{#each cat.items as opt}
										<button class="ctx-menu__item" onclick={() => addNodeFromMenu(opt.type, opt.label, opt.data)}>
											<div class="ctx-menu__item-info">
												<span class="ctx-menu__item-text">{opt.label}</span>
												<span class="ctx-menu__item-desc">{opt.desc}</span>
											</div>
											<span class="ctx-menu__item-type">{opt.type === 'aiStep' ? 'ai' : opt.type}</span>
										</button>
									{/each}
								</div>
							{:else}
								<div class="ctx-menu__group ctx-menu__group--{cat.label.toLowerCase()}">
									<button
										class="ctx-menu__cat"
										class:ctx-menu__cat--open={expandedCategory === cat.label}
										onclick={() => { expandedCategory = expandedCategory === cat.label ? null : cat.label; }}
									>
										<span class="ctx-menu__cat-dot"></span>
										<span class="ctx-menu__cat-name">{cat.label}</span>
										<span class="ctx-menu__cat-count">{cat.items.length}</span>
										<svg class="ctx-menu__cat-arrow" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
											<polyline points="6 9 12 15 18 9" />
										</svg>
									</button>
									{#if expandedCategory === cat.label}
										<div class="ctx-menu__items">
											{#each cat.items as opt, i}
												<button
													class="ctx-menu__item"
													style="animation-delay: {i * 30}ms"
													onclick={() => addNodeFromMenu(opt.type, opt.label, opt.data)}
												>
													<div class="ctx-menu__item-info">
													<span class="ctx-menu__item-text">{opt.label}</span>
													<span class="ctx-menu__item-desc">{opt.desc}</span>
												</div>
												</button>
											{/each}
										</div>
									{/if}
								</div>
							{/if}
						{:else}
							<div class="ctx-menu__empty">
								<span>No nodes match</span>
								<span class="ctx-menu__empty-hint">Try "trigger", "ai", or "shell"</span>
							</div>
						{/each}
					</div>
					<div class="ctx-menu__foot">
						<span><kbd>Space</kbd> toggle</span>
						<span><kbd>Enter</kbd> select</span>
					</div>
				</div>
			{/if}

			<NodeContextMenu
				open={!!nodeContextMenu && !!nodeContextTarget}
				x={nodeContextMenu?.x ?? 0}
				y={nodeContextMenu?.y ?? 0}
				node={nodeContextTarget}
				ontoggleDisabled={() => nodeContextMenu && toggleNodeDisabled(nodeContextMenu.nodeId)}
				ondelete={() => nodeContextMenu && deleteNode(nodeContextMenu.nodeId)}
				onclose={() => { nodeContextMenu = null; }}
			/>
		</div>
		{#if selectedNode}
			<NodeConfigPanel
				node={selectedNode}
				onupdate={updateNodeData}
				onclose={() => { selectedNodeId = null; }}
			/>
		{/if}
		</div>

		<YamlPreviewModal
			open={yamlOpen}
			title={automation.name}
			yaml={yamlText}
			loading={yamlLoading}
			error={yamlError}
			footer="Reflects the last saved state. Save first to include unsaved changes."
			onclose={() => { yamlOpen = false; }}
		/>
	</div>
{/if}

<style lang="scss">
	.loading, .error {
		padding: $space-8;
		text-align: center;
		color: $neutral-500;
	}

	.error { color: $error-600; }

	.editor {
		display: flex;
		flex-direction: column;
		height: 100%;

		&__toolbar {
			@include flex-between;
			padding: $space-3 $space-4;
			border-bottom: 1px solid $neutral-200;
			background: $neutral-0;
			gap: $space-4;
			flex-shrink: 0;
		}

		&__left {
			display: flex;
			align-items: center;
			gap: $space-3;
		}

		&__back {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 32px;
			height: 32px;
			border-radius: $radius-lg;
			color: $neutral-500;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__title {
			font-size: $text-base;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__right {
			display: flex;
			align-items: center;
			gap: $space-4;
		}

		&__palette {
			display: flex;
			gap: $space-2;
		}

		&__save {
			padding: $space-2 $space-4;
		}

		&__yaml {
			display: inline-flex;
			align-items: center;
			gap: $space-2;
			padding: $space-2 $space-3;
			font-family: $font-mono;
			font-size: $text-xs;
			text-transform: uppercase;
			letter-spacing: 0.04em;

			svg { color: $neutral-500; }
		}

		&__main {
			flex: 1;
			display: flex;
			overflow: hidden;
		}

		&__canvas {
			flex: 1;
			position: relative;
		}
	}

	:global(.svelte-flow .svelte-flow__node.nd-disabled) {
		opacity: 0.5;
		filter: grayscale(0.6);
	}

	:global(.svelte-flow .svelte-flow__handle) {
		width: 12px;
		height: 12px;
		background: $neutral-0;
		border: 2px solid $neutral-400;
		border-radius: 50%;
		transition: all $transition-fast;
	}

	:global(.svelte-flow .svelte-flow__handle:hover) {
		border-color: $primary-500;
		background: $primary-50;
	}

	:global(.svelte-flow .svelte-flow__handle.connecting) {
		border-color: $primary-500;
		background: $primary-100;
	}

	:global(.svelte-flow .svelte-flow__edges) {
		width: 100%;
		height: 100%;
	}

	:global(.svelte-flow .svelte-flow__edge-path) {
		stroke: $neutral-400;
		stroke-width: 2;
	}

	:global(.svelte-flow .svelte-flow__connection-path) {
		stroke: $primary-500;
		stroke-width: 2;
	}

	.empty-canvas {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		pointer-events: none;
		z-index: 1;

		&__card {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: $space-3;
			text-align: center;
			max-width: 300px;
			color: $neutral-400;

			svg { opacity: 0.4; }

			h3 {
				font-size: $text-base;
				font-weight: $font-semibold;
				color: $neutral-600;
			}

			p {
				font-size: $text-sm;
				color: $neutral-400;
				line-height: $leading-relaxed;
			}

			kbd {
				font-family: $font-mono;
				font-size: $text-xs;
				background: $neutral-100;
				color: $neutral-600;
				padding: 1px 5px;
				border-radius: $radius-sm;
				border: 1px solid $neutral-200;
			}
		}
	}

	.ctx-overlay {
		position: absolute;
		inset: 0;
		z-index: 10;
		background: rgba($neutral-900, 0.08);
		border: none;
		cursor: default;
		animation: ctx-fade 0.12s ease-out;
	}

	.ctx-menu {
		position: absolute;
		z-index: 11;
		transform: translate(-50%, -50%);
		width: 520px;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-2xl;
		box-shadow: $shadow-lg;
		display: flex;
		flex-direction: column;
		animation: ctx-appear 0.15s cubic-bezier(0.16, 1, 0.3, 1);
		overflow: hidden;

		&__head {
			padding: $space-2 $space-3;
			border-bottom: 1px solid $neutral-100;
		}

		&__search {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: $space-1 $space-2;
			background: $neutral-50;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			transition: border-color $transition-fast;

			&:focus-within {
				border-color: $primary-400;
			}

			input {
				flex: 1;
				background: none;
				border: none;
				outline: none;
				font-size: $text-sm;
				color: $neutral-800;
				padding: $space-1 0;
				font-family: $font-sans;

				&::placeholder { color: $neutral-400; }
			}
		}

		&__search-icon {
			color: $neutral-400;
			flex-shrink: 0;
		}

		&__kbd {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-medium;
			color: $neutral-400;
			background: $neutral-100;
			padding: 2px 5px;
			border-radius: $radius-sm;
			border: 1px solid $neutral-200;
			line-height: 1;
			text-transform: uppercase;
			flex-shrink: 0;
		}

		&__body {
			max-height: 360px;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-1;
		}

		&__group {
			display: flex;
			flex-direction: column;
		}

		&__group-label {
			display: block;
			padding: $space-1 $space-3 2px;
			font-size: 10px;
			font-weight: $font-bold;
			color: $neutral-400;
			text-transform: uppercase;
			letter-spacing: 0.06em;
		}

		&__cat {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: 6px $space-3;
			background: none;
			border: none;
			cursor: pointer;
			border-radius: $radius-lg;
			transition: all $transition-fast;
			text-align: left;
			width: 100%;

			&:hover { background: $neutral-50; }
			&--open { background: $neutral-50; }
		}

		&__cat-dot {
			width: 8px;
			height: 8px;
			border-radius: 50%;
			flex-shrink: 0;

			.ctx-menu__group--triggers & { background: $success-500; }
			.ctx-menu__group--ai & { background: $primary-500; }
			.ctx-menu__group--tools & { background: $warning-500; }
			.ctx-menu__group--flow & { background: $info-500; }
		}

		&__cat-name {
			flex: 1;
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-800;
		}

		&__cat-count {
			font-family: $font-mono;
			font-size: 10px;
			color: $neutral-400;
		}

		&__cat-arrow {
			color: $neutral-400;
			transition: transform $transition-fast;
			flex-shrink: 0;

			.ctx-menu__cat--open & { transform: rotate(180deg); }
		}

		&__items {
			display: flex;
			flex-direction: column;
			padding-left: $space-5;
			margin-left: 4px;
			border-left: 2px solid $neutral-100;
		}

		&__item {
			display: flex;
			align-items: flex-start;
			gap: $space-3;
			padding: $space-2 $space-3;
			background: none;
			border: none;
			cursor: pointer;
			border-radius: $radius-lg;
			transition: all $transition-fast;
			text-align: left;
			animation: ctx-item-in 0.12s ease-out both;

			&:hover { background: $neutral-50; }
		}

		&__item-info {
			flex: 1;
			display: flex;
			flex-direction: column;
			gap: 1px;
			min-width: 0;
		}

		&__item-text {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-800;
		}

		&__item-desc {
			font-size: $text-xs;
			color: $neutral-400;
			line-height: $leading-normal;
		}

		&__item-type {
			font-family: $font-mono;
			font-size: 9px;
			color: $neutral-400;
			background: $neutral-100;
			padding: 2px 5px;
			border-radius: $radius-sm;
			flex-shrink: 0;
			margin-top: 2px;
		}

		&__empty {
			padding: $space-6 $space-4;
			text-align: center;
			display: flex;
			flex-direction: column;
			gap: $space-1;

			span { font-size: $text-sm; color: $neutral-400; }
		}

		&__empty-hint {
			font-size: $text-xs !important;
			color: $neutral-300 !important;
			font-family: $font-mono;
		}

		&__foot {
			display: flex;
			justify-content: center;
			gap: $space-4;
			padding: $space-2 $space-3;
			border-top: 1px solid $neutral-100;

			span {
				font-size: 10px;
				color: $neutral-400;
				display: flex;
				align-items: center;
				gap: $space-1;
			}

			kbd {
				font-family: $font-mono;
				font-size: 9px;
				color: $neutral-500;
				background: $neutral-100;
				padding: 1px 4px;
				border-radius: 3px;
				border: 1px solid $neutral-200;
			}
		}
	}

	@keyframes ctx-fade {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes ctx-appear {
		from { opacity: 0; transform: translate(-50%, -50%) scale(0.95); }
		to { opacity: 1; transform: translate(-50%, -50%) scale(1); }
	}

	@keyframes ctx-item-in {
		from { opacity: 0; transform: translateX(-4px); }
		to { opacity: 1; transform: translateX(0); }
	}

</style>
