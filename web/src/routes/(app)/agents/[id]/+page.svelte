<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import type { AgentRole, Agent } from '$lib/api/types';
	import { listBrainFiles } from '$lib/api/brain';

	let agentId = $derived(page.params.id);
	let notFound = $state(false);
	let name = $state('');
	let description = $state('');
	let role = $state<AgentRole>('worker');
	let systemPrompt = $state('');
	let provider = $state('anthropic');
	let model = $state('claude-sonnet');
	let sharing = $state<'personal' | 'team' | 'org'>('personal');
	let tools = $state<string[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let saved = $state(false);
	let brainFileCount = $state(0);

	const availableTools = [
		{ id: 'bash', name: 'Shell', desc: 'Execute shell commands', icon: 'M4 17l6-6-6-6M12 19h8' },
		{ id: 'web_fetch', name: 'HTTP', desc: 'Fetch data from URLs', icon: 'M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.66 0 3-4.03 3-9s-1.34-9-3-9m0 18c-1.66 0-3-4.03-3-9s1.34-9 3-9' },
		{ id: 'file_read', name: 'Read File', desc: 'Read from VM filesystem', icon: 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z M14 2v6h6' },
		{ id: 'file_write', name: 'Write File', desc: 'Write to VM filesystem', icon: 'M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7 M18.5 2.5a2.12 2.12 0 013 3L12 15l-4 1 1-4 9.5-9.5z' },
		{ id: 'code_interpreter', name: 'Code', desc: 'Run Python code', icon: 'M16 18l6-6-6-6M8 6l-6 6 6 6' }
	];

	onMount(async () => {
		const agents: Agent[] = JSON.parse(localStorage.getItem('forgebox_agents') ?? '[]');
		const agent = agents.find((a) => a.id === agentId);
		if (!agent) { notFound = true; return; }
		name = agent.name;
		description = agent.description;
		role = agent.role;
		systemPrompt = agent.system_prompt;
		provider = agent.provider;
		model = agent.model;
		tools = [...agent.tools];
		sharing = agent.sharing;

		try {
			const files = await listBrainFiles(agentId);
			brainFileCount = files.length;
		} catch {
			brainFileCount = 0;
		}
	});

	function toggleTool(id: string) {
		if (tools.includes(id)) tools = tools.filter((t) => t !== id);
		else tools = [...tools, id];
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim()) return;
		loading = true;
		error = null;
		saved = false;
		try {
			const agents: Agent[] = JSON.parse(localStorage.getItem('forgebox_agents') ?? '[]');
			const idx = agents.findIndex((a) => a.id === agentId);
			if (idx === -1) throw new Error('Agent not found');
			agents[idx] = {
				...agents[idx],
				name: name.trim(),
				description: description.trim(),
				role,
				system_prompt: systemPrompt.trim(),
				provider,
				model,
				tools,
				sharing,
				updated_at: new Date().toISOString()
			};
			localStorage.setItem('forgebox_agents', JSON.stringify(agents));
			saved = true;
			setTimeout(() => { saved = false; }, 2000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			loading = false;
		}
	}

	const sharingLabel = $derived(
		sharing === 'org' ? 'Organization' : sharing === 'team' ? 'Team' : 'Personal'
	);
</script>

{#if notFound}
	<div class="cr">
		<div class="cr__main">
			<p>Agent not found.</p>
			<a href="/agents">Back to agents</a>
		</div>
	</div>
{:else}
<div class="cr">
	<div class="cr__main">
		<div class="cr__top">
			<a href="/agents" class="cr__back">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
			</a>
			<div class="cr__top-main">
				<h1 class="cr__title">Edit Agent</h1>
				<p class="cr__sub">Update this agent's configuration.</p>
			</div>
		</div>

		{#if error}
			<div class="cr__error">{error}</div>
		{/if}

		<form class="cr__form" onsubmit={handleSubmit}>
			<section class="sec">
				<div class="sec__row">
					<label class="fld fld--grow">
						<span class="fld__lbl">Name</span>
						<input class="fld__input" type="text" bind:value={name} placeholder="e.g. Code Reviewer" required disabled={loading} />
					</label>
				</div>

				<div class="fld">
					<div class="fld__lbl-row">
						<span class="fld__lbl">Description</span>
						<span class="hint-wrap">
							<svg class="hint-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
							<span class="hint-bubble">Visible to your team. Helps others understand when to use this agent.</span>
						</span>
					</div>
					<textarea class="fld__input fld__input--ta" bind:value={description} placeholder="What does this agent do?" rows="2" disabled={loading}></textarea>
				</div>
			</section>

			<section class="sec">
				<div class="sec__head">
					<span class="sec__label">Role</span>
					<span class="hint-wrap">
						<svg class="hint-icon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
						<span class="hint-bubble hint-bubble--wide">
							<strong>Orchestrator</strong> agents spawn and coordinate other agents. They plan complex tasks, delegate to workers, and verify completion.<br/><br/>
							<strong>Worker</strong> agents are specialized for specific tasks using their assigned tools.
						</span>
					</span>
				</div>
				<div class="roles">
					<button type="button" class="roles__btn" class:roles__btn--on={role === 'orchestrator'} onclick={() => { role = 'orchestrator'; }} disabled={loading}>
						<div class="roles__icon">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 2v4m0 12v4m-7.07-15.07l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/></svg>
						</div>
						<strong>Orchestrator</strong>
						<span>Plans, delegates, verifies</span>
					</button>
					<button type="button" class="roles__btn" class:roles__btn--on={role === 'worker'} onclick={() => { role = 'worker'; }} disabled={loading}>
						<div class="roles__icon">
							<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z"/></svg>
						</div>
						<strong>Worker</strong>
						<span>Executes specific tasks</span>
					</button>
				</div>
			</section>

			<section class="sec">
				<span class="sec__label">Model</span>
				<div class="sec__row">
					<label class="fld fld--grow">
						<span class="fld__lbl">Provider</span>
						<select class="fld__input" bind:value={provider} disabled={loading}>
							<option value="anthropic">Anthropic</option>
							<option value="openai">OpenAI</option>
							<option value="ollama">Ollama</option>
						</select>
					</label>
					<label class="fld fld--grow">
						<span class="fld__lbl">Model ID</span>
						<input class="fld__input" type="text" bind:value={model} placeholder="claude-sonnet" disabled={loading} />
					</label>
				</div>
			</section>

			<section class="sec">
				<span class="sec__label">System Prompt</span>
				<label class="fld">
					<span class="fld__lbl">Instructions</span>
					<textarea class="fld__input fld__input--ta fld__input--code" bind:value={systemPrompt} placeholder="You are a helpful assistant that..." rows="7" disabled={loading}></textarea>
				</label>
			</section>

			<section class="sec">
				<span class="sec__label">Tools</span>
				<div class="tgrid">
					{#each availableTools as tool}
						<button
							type="button"
							class="tgrid__item"
							class:tgrid__item--on={tools.includes(tool.id)}
							onclick={() => toggleTool(tool.id)}
							disabled={loading}
						>
							<svg class="tgrid__icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d={tool.icon} /></svg>
							<strong>{tool.name}</strong>
							<span>{tool.desc}</span>
						</button>
					{/each}
				</div>
			</section>

			<section class="sec">
				<span class="sec__label">Visibility</span>
				<div class="vis">
					{#each [
						{ val: 'personal', lbl: 'Personal', desc: 'Only you' },
						{ val: 'team', lbl: 'Team', desc: 'Your team' },
						{ val: 'org', lbl: 'Organization', desc: 'Everyone' }
					] as opt}
						<button
							type="button"
							class="vis__opt"
							class:vis__opt--on={sharing === opt.val}
							onclick={() => { sharing = opt.val as typeof sharing; }}
							disabled={loading}
						>
							<strong>{opt.lbl}</strong>
							<span>{opt.desc}</span>
						</button>
					{/each}
				</div>
			</section>

			<button type="submit" class="cr__submit" class:cr__submit--saved={saved} disabled={loading || !name.trim()}>
				{#if loading}
					<span class="cr__spinner"></span> Saving...
				{:else if saved}
					Saved
				{:else}
					Save Changes
				{/if}
			</button>
		</form>
	</div>

	<aside class="cr__preview">
		<div class="preview">
			<span class="preview__tag">Preview</span>
			<div class="preview__card">
				<div class="preview__avatar" class:preview__avatar--orch={role === 'orchestrator'}>
					{#if role === 'orchestrator'}
						<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M12 2v4m0 12v4m-7.07-15.07l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/></svg>
					{:else}
						<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l3.77-3.77a6 6 0 01-7.94 7.94l-6.91 6.91a2.12 2.12 0 01-3-3l6.91-6.91a6 6 0 017.94-7.94l-3.76 3.76z"/></svg>
					{/if}
				</div>
				<h3 class="preview__name">{name || 'Untitled Agent'}</h3>
				<p class="preview__desc">{description || 'No description'}</p>
				<div class="preview__meta">
					<span class="preview__chip preview__chip--role">{role}</span>
					<span class="preview__chip">{provider}/{model}</span>
					<span class="preview__chip">{sharingLabel}</span>
				</div>
				{#if tools.length > 0}
					<div class="preview__tools">
						{#each tools as t}
							<span class="preview__tool">{availableTools.find((a) => a.id === t)?.name ?? t}</span>
						{/each}
					</div>
				{/if}
				{#if systemPrompt.trim()}
					<div class="preview__prompt">
						<span class="preview__prompt-lbl">System prompt</span>
						<p>{systemPrompt.length > 120 ? systemPrompt.slice(0, 120) + '...' : systemPrompt}</p>
					</div>
				{/if}
			</div>
			<a href="/agents/{agentId}/brain" class="cr__brain">
				<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<path d="M9.5 2a4.5 4.5 0 0 0-4.5 4.5v0A3.5 3.5 0 0 0 3 10v0a3.5 3.5 0 0 0 2 3.16V15a4 4 0 0 0 4 4h.5" />
					<path d="M14.5 2a4.5 4.5 0 0 1 4.5 4.5v0A3.5 3.5 0 0 1 21 10v0a3.5 3.5 0 0 1-2 3.16V15a4 4 0 0 1-4 4h-.5" />
					<path d="M9.5 2v20" />
					<path d="M14.5 2v20" />
				</svg>
				<span class="cr__brain-label">Brain</span>
				<span class="cr__brain-count">({brainFileCount})</span>
			</a>
		</div>
	</aside>
</div>
{/if}

<style lang="scss">
	.cr {
		display: flex;
		gap: $space-8;
		max-width: 960px;
		margin: 0 auto;

		&__main {
			flex: 1;
			min-width: 0;
		}

		&__top {
			display: flex;
			align-items: flex-start;
			gap: $space-3;
			margin-bottom: $space-6;
		}

		&__top-main {
			flex: 1;
			min-width: 0;
		}

		&__brain-label {
			line-height: 1;
		}

		&__brain-count {
			font-weight: $font-normal;
			font-size: $text-xs;
			line-height: 1;
			opacity: 0.7;
			transform: translateY(1px);
		}

		&__brain {
			display: flex;
			width: 100%;
			align-items: center;
			justify-content: center;
			gap: $space-2;
			padding: $space-3 $space-4;
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: #fff;
			background: linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%);
			border: 1px solid #6d28d9;
			border-radius: $radius-lg;
			box-shadow: 0 1px 2px rgba(109, 40, 217, 0.25), 0 0 0 1px rgba(139, 92, 246, 0.15);
			transition: all $transition-fast;

			&:hover {
				background: linear-gradient(135deg, #7c3aed 0%, #5b21b6 100%);
				box-shadow: 0 4px 12px rgba(109, 40, 217, 0.35);
				transform: translateY(-1px);
			}
		}

		&__back {
			@include flex-center;
			width: 36px;
			height: 36px;
			flex-shrink: 0;
			border-radius: $radius-lg;
			color: $neutral-400;
			transition: all $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__title {
			font-size: $text-2xl;
			font-weight: $font-bold;
			color: $neutral-900;
			line-height: 1.2;
		}

		&__sub {
			font-size: $text-sm;
			color: $neutral-400;
			margin-top: 2px;
		}

		&__error {
			padding: $space-3;
			margin-bottom: $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}

		&__form {
			display: flex;
			flex-direction: column;
			gap: $space-5;
		}

		&__submit {
			@include btn;
			width: 100%;
			padding: $space-3;
			font-weight: $font-semibold;
			color: $neutral-0;
			background: $primary-600;
			border-radius: $radius-xl;
			margin-top: $space-2;

			&:hover:not(:disabled) { background: $primary-700; }
			&:disabled { opacity: 0.5; }

			&--saved {
				background: $success-500;
				&:hover:not(:disabled) { background: $success-600; }
			}
		}

		&__spinner {
			display: inline-block;
			width: 14px;
			height: 14px;
			border: 2px solid rgba(255, 255, 255, 0.3);
			border-top-color: $neutral-0;
			border-radius: 50%;
			animation: spin 0.6s linear infinite;
			margin-right: $space-2;
		}

		&__preview {
			width: 280px;
			flex-shrink: 0;
			position: sticky;
			top: $space-4;
			align-self: flex-start;
		}
	}

	.sec {
		@include card;
		padding: $space-4 $space-5;
		display: flex;
		flex-direction: column;
		gap: $space-3;

		&__head {
			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__label {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: $neutral-400;
		}

		&__row {
			display: flex;
			gap: $space-3;
		}
	}

	.fld {
		display: flex;
		flex-direction: column;
		gap: $space-1;

		&--grow { flex: 1; }

		&__lbl-row {
			display: flex;
			align-items: center;
			gap: $space-2;
		}

		&__lbl {
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
		}

		&__input {
			@include input-base;
			font-size: $text-sm;
			padding: 7px $space-3;
			border-radius: $radius-md;

			&--ta {
				resize: vertical;
				min-height: 48px;
			}

			&--code {
				font-family: $font-mono;
				font-size: $text-xs;
				line-height: $leading-relaxed;
			}
		}
	}

	.hint-wrap {
		position: relative;
		display: inline-flex;
		cursor: help;

		&:hover .hint-bubble {
			opacity: 1;
			transform: translateY(0);
			pointer-events: auto;
		}
	}

	.hint-icon { color: $neutral-300; }

	.hint-bubble {
		position: absolute;
		bottom: calc(100% + 8px);
		left: 0;
		width: 240px;
		padding: $space-2 $space-3;
		background: $neutral-800;
		color: $neutral-300;
		font-size: $text-xs;
		font-weight: $font-normal;
		line-height: $leading-relaxed;
		border-radius: $radius-lg;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
		opacity: 0;
		pointer-events: none;
		transition: opacity 0.12s ease, transform 0.12s ease;
		transform: translateY(4px);
		z-index: 50;

		strong { color: $neutral-0; }

		&--wide { width: 300px; }

		&::after {
			content: '';
			position: absolute;
			top: 100%;
			left: 16px;
			border: 5px solid transparent;
			border-top-color: $neutral-800;
		}
	}

	.roles {
		display: flex;
		gap: $space-2;

		&__btn {
			flex: 1;
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: $space-1;
			padding: $space-4 $space-3;
			border: 1px solid $neutral-200;
			border-radius: $radius-xl;
			background: $neutral-0;
			cursor: pointer;
			text-align: center;
			transition: all $transition-fast;

			&:hover { border-color: $neutral-300; background: $neutral-50; }

			&--on {
				border-color: $primary-500;
				background: $primary-50;
				box-shadow: 0 0 0 1px $primary-500;
			}

			strong {
				font-size: $text-sm;
				font-weight: $font-semibold;
				color: $neutral-800;
			}

			span {
				font-size: $text-xs;
				color: $neutral-400;
			}
		}

		&__icon {
			@include flex-center;
			width: 40px;
			height: 40px;
			border-radius: $radius-lg;
			background: $neutral-100;
			color: $neutral-500;
			margin-bottom: $space-1;
			transition: all $transition-fast;

			.roles__btn--on & {
				background: $primary-100;
				color: $primary-600;
			}
		}
	}

	.tgrid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
		gap: $space-2;

		&__item {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 4px;
			padding: $space-3 $space-2;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			background: $neutral-0;
			cursor: pointer;
			text-align: center;
			transition: all $transition-fast;

			&:hover { border-color: $neutral-300; }

			&--on {
				border-color: $primary-500;
				background: $primary-50;

				.tgrid__icon { color: $primary-600; }
			}

			strong {
				font-size: $text-xs;
				font-weight: $font-semibold;
				color: $neutral-700;
			}

			span {
				font-size: 10px;
				color: $neutral-400;
				line-height: 1.3;
			}
		}

		&__icon {
			color: $neutral-400;
			margin-bottom: 2px;
			transition: color $transition-fast;
		}
	}

	.vis {
		display: flex;
		gap: $space-2;

		&__opt {
			flex: 1;
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 2px;
			padding: $space-3;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			background: $neutral-0;
			cursor: pointer;
			text-align: center;
			transition: all $transition-fast;

			&:hover { border-color: $neutral-300; }

			&--on {
				border-color: $primary-500;
				background: $primary-50;
				box-shadow: 0 0 0 1px $primary-500;
			}

			strong {
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-800;
			}

			span {
				font-size: $text-xs;
				color: $neutral-400;
			}
		}
	}

	.preview {
		display: flex;
		flex-direction: column;
		gap: $space-3;

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: $neutral-400;
		}

		&__card {
			@include card;
			padding: $space-5;
			display: flex;
			flex-direction: column;
			gap: $space-3;
		}

		&__avatar {
			@include flex-center;
			width: 48px;
			height: 48px;
			border-radius: $radius-xl;
			background: $neutral-100;
			color: $neutral-500;

			&--orch {
				background: $primary-100;
				color: $primary-600;
			}
		}

		&__name {
			font-size: $text-base;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__desc {
			font-size: $text-xs;
			color: $neutral-400;
			line-height: $leading-relaxed;
		}

		&__meta {
			display: flex;
			flex-wrap: wrap;
			gap: 4px;
		}

		&__chip {
			font-family: $font-mono;
			font-size: 10px;
			padding: 2px 7px;
			border-radius: $radius-sm;
			background: $neutral-100;
			color: $neutral-600;

			&--role {
				background: $primary-100;
				color: $primary-700;
				text-transform: capitalize;
			}
		}

		&__tools {
			display: flex;
			flex-wrap: wrap;
			gap: 4px;
			padding-top: $space-2;
			border-top: 1px solid $neutral-100;
		}

		&__tool {
			font-family: $font-mono;
			font-size: 10px;
			padding: 2px 7px;
			border-radius: $radius-sm;
			background: $success-50;
			color: $success-600;
		}

		&__prompt {
			padding-top: $space-2;
			border-top: 1px solid $neutral-100;

			p {
				font-family: $font-mono;
				font-size: 10px;
				color: $neutral-500;
				line-height: $leading-relaxed;
				word-break: break-word;
			}
		}

		&__prompt-lbl {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-bold;
			text-transform: uppercase;
			letter-spacing: 0.06em;
			color: $neutral-300;
			display: block;
			margin-bottom: $space-1;
		}
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	@include md {
		/* handled by flex already */
	}

	@media (max-width: 768px) {
		.cr {
			flex-direction: column;

			&__preview {
				width: 100%;
				position: static;
			}
		}
	}
</style>
