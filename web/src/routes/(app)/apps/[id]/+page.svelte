<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import type { App } from '$lib/api/types';
	import { getApp, deleteApp, createTask, getTask } from '$lib/api/client';

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		status?: 'pending' | 'running' | 'completed' | 'failed';
		taskId?: string;
	}

	let app = $state<App | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let input = $state('');
	let sending = $state(false);
	let messages = $state<ChatMessage[]>([]);
	let messagesEl: HTMLDivElement;

	const id = $derived(page.params.id);

	function parseTools(tools: string): string[] {
		try { return JSON.parse(tools); }
		catch { return []; }
	}

	function toolLabel(t: string) {
		if (t === 'database') return 'Database';
		if (t === 'api') return 'API';
		if (t === 'ai') return 'AI';
		return t;
	}

	onMount(async () => {
		try {
			app = await getApp(id);
			const tools = parseTools(app.tools).map(toolLabel);
			messages = [
				{
					id: 'init-user',
					role: 'user',
					content: app.description || app.name
				},
				{
					id: 'init-assistant',
					role: 'assistant',
					content: `I'm setting up "${app.name}" with ${tools.join(', ')} access in an isolated VM.\n\nWhat would you like this app to do? Describe the features, data sources, or workflows and I'll build it for you.`
				}
			];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load app';
		} finally {
			loading = false;
		}
	});

	async function scrollToBottom() {
		await tick();
		if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
	}

	async function pollTask(taskId: string, msgIndex: number) {
		let attempts = 0;
		while (attempts < 120) {
			await new Promise((r) => setTimeout(r, 2000));
			attempts++;
			try {
				const task = await getTask(taskId);
				if (task.status === 'completed') {
					messages[msgIndex] = { ...messages[msgIndex], content: task.result || 'Done.', status: 'completed' };
					await scrollToBottom();
					return;
				} else if (task.status === 'failed') {
					messages[msgIndex] = { ...messages[msgIndex], content: task.error || 'Something went wrong.', status: 'failed' };
					await scrollToBottom();
					return;
				}
			} catch { /* keep polling */ }
		}
		messages[msgIndex] = { ...messages[msgIndex], content: 'Timed out waiting for response.', status: 'failed' };
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		const prompt = input.trim();
		if (!prompt || sending || !app) return;

		input = '';
		sending = true;

		messages = [...messages, { id: crypto.randomUUID(), role: 'user', content: prompt }];

		const assistantMsg: ChatMessage = { id: crypto.randomUUID(), role: 'assistant', content: '', status: 'pending' };
		messages = [...messages, assistantMsg];
		const assistantIndex = messages.length - 1;
		await scrollToBottom();

		try {
			const context = `You are building an internal tool app called "${app.name}". Description: ${app.description}. Available tools: ${app.tools}. The user is iterating on this app with you.\n\nUser request: ${prompt}`;
			const res = await createTask({ prompt: context });
			messages[assistantIndex] = { ...messages[assistantIndex], taskId: res.task_id, status: 'running' };
			sending = false;
			pollTask(res.task_id, assistantIndex);
		} catch (err) {
			messages[assistantIndex] = { ...messages[assistantIndex], content: err instanceof Error ? err.message : 'Failed to send', status: 'failed' };
			sending = false;
		}

		await scrollToBottom();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			handleSubmit(e);
		}
	}

	async function handleDelete() {
		if (!app || !confirm('Delete this app? This cannot be undone.')) return;
		try {
			await deleteApp(app.id);
			goto('/apps');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete';
		}
	}
</script>

{#if loading}
	<div class="chat chat--started">
		<div class="chat__messages">
			<div class="chat__content">
				<div class="chat__loading"><span class="msg__spinner"></span></div>
			</div>
		</div>
	</div>
{:else if error && !app}
	<div class="chat chat--started">
		<div class="chat__messages">
			<div class="chat__content">
				<div class="chat__error">{error}</div>
				<a href="/apps" class="btn-secondary">Back to Apps</a>
			</div>
		</div>
	</div>
{:else if app}
	<div class="chat chat--started">
		<div class="chat__topbar">
			<a href="/apps" class="chat__back">
				<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
				Back to Apps
			</a>
		</div>

		<div class="chat__messages" bind:this={messagesEl}>
			<div class="chat__content">
				<div class="chat__thread">
					{#each messages as msg}
						<div class="msg msg--{msg.role}">
							<div class="msg__bubble">
								{#if msg.status === 'pending'}
									<span class="msg__creating"><span class="msg__spinner"></span> Thinking...</span>
								{:else if msg.status === 'running'}
									<span class="msg__creating"><span class="msg__spinner"></span> Thinking...</span>
								{:else}
									<span class="msg__text">{msg.content}</span>
								{/if}
							</div>
							{#if msg.status === 'failed'}
								<span class="msg__failed">Failed</span>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		</div>

		<div class="chat__bottom-bar">
			<form class="chat__input-form" onsubmit={handleSubmit}>
				<textarea
					bind:value={input}
					onkeydown={handleKeydown}
					placeholder="Describe what you'd like to build or change..."
					rows="1"
					disabled={sending}
				></textarea>
				<button type="submit" class="btn-primary" disabled={sending || !input.trim()}>
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="22" y1="2" x2="11" y2="13" />
						<polygon points="22 2 15 22 11 13 2 9 22 2" />
					</svg>
				</button>
			</form>
		</div>
	</div>
{/if}

<style lang="scss">
	.chat {
		display: flex;
		flex-direction: column;
		height: 100%;
		/* fullscreen layout — no margin needed */

		&__messages {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			display: flex;
			flex-direction: column;
			align-items: center;
			padding: $space-6;
		}

		&--started &__messages {
			padding-top: $space-6;
		}

		&__content {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: $space-5;
			width: 100%;
			max-width: 700px;
		}

		&__thread {
			display: flex;
			flex-direction: column;
			gap: $space-3;
			width: 100%;
		}

		&__loading {
			@include flex-center;
			padding: $space-16;
		}

		&__error {
			padding: $space-3 $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}

		&__input-form {
			display: flex;
			align-items: flex-end;
			gap: $space-3;
			width: 100%;

			textarea {
				flex: 1;
				@include input-base;
				resize: none;
				min-height: 44px;
				max-height: 160px;
				padding: $space-3;
				font-size: $text-sm;
				border-radius: $radius-xl;
			}

			button {
				flex-shrink: 0;
				width: 44px;
				height: 44px;
				padding: 0;
				@include flex-center;
				border-radius: $radius-xl;
			}
		}

		&__bottom-bar {
			padding: $space-4 $space-6;
			border-top: 1px solid $neutral-200;
			background: $neutral-0;
			display: flex;
			justify-content: center;
		}

		&--started &__bottom-bar {
			opacity: 1;
			transform: translateY(0);
			max-height: 120px;
		}

		&__bottom-bar &__input-form {
			max-width: 700px;
		}

		&__topbar {
			display: flex;
			align-items: center;
			padding: $space-3 $space-6;
			border-bottom: 1px solid $neutral-100;
			flex-shrink: 0;
		}

		&__back {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-400;
			text-decoration: none;
			transition: color $transition-fast;

			&:hover { color: $neutral-700; }
		}
	}

	.msg {
		display: flex;
		flex-direction: column;

		&--user {
			align-items: flex-end;

			.msg__bubble {
				background: $primary-600;
				color: $neutral-0;
				border-radius: $radius-xl $radius-xl $radius-sm $radius-xl;
			}
		}

		&--assistant {
			align-items: flex-start;

			.msg__bubble {
				background: $neutral-100;
				color: $neutral-800;
				border-radius: $radius-xl $radius-xl $radius-xl $radius-sm;
			}
		}

		&__bubble {
			max-width: 85%;
			padding: $space-3 $space-4;
			font-size: $text-sm;
			line-height: $leading-relaxed;
			word-break: break-word;
		}

		&__text {
			white-space: pre-wrap;
		}

		&__creating {
			display: flex;
			align-items: center;
			gap: $space-2;
			color: $neutral-500;
		}

		&__spinner {
			display: inline-block;
			width: 14px;
			height: 14px;
			border: 2px solid $neutral-300;
			border-top-color: $primary-500;
			border-radius: 50%;
			animation: spin 0.7s linear infinite;
		}

		&__failed {
			font-size: $text-xs;
			color: $error-600;
			margin-top: $space-1;
			padding: 0 $space-1;
		}
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
