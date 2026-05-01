<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { createTask, getTask } from '$lib/api/client';
	import { currentUser } from '$lib/stores/auth';
	import { loadProviders, providersStore } from '$lib/stores/providers.svelte';
	import ModelSelector from '$lib/components/ModelSelector.svelte';

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		status?: 'pending' | 'running' | 'completed' | 'failed';
		taskId?: string;
	}

	const STORAGE_KEY = 'forgebox.chat.lastModel';

	let input = $state('');
	let provider = $state('');
	let model = $state('');
	let messages = $state<ChatMessage[]>([]);
	let sending = $state(false);
	let messagesEl: HTMLDivElement;

	onMount(() => {
		// Restore the user's last picked (provider, model) before providers
		// arrive. If the saved pair no longer exists in the list, the
		// ModelSelector's snap-to-default effect will fall back to the first
		// provider's strongest model per specs/3.3.3.
		try {
			const saved = localStorage.getItem(STORAGE_KEY);
			if (saved) {
				const parsed = JSON.parse(saved);
				if (typeof parsed?.provider === 'string' && typeof parsed?.model === 'string') {
					provider = parsed.provider;
					model = parsed.model;
				}
			}
		} catch {
			/* ignore malformed storage */
		}
		void loadProviders();
	});

	$effect(() => {
		if (provider && model) {
			localStorage.setItem(STORAGE_KEY, JSON.stringify({ provider, model }));
		}
	});

	async function scrollToBottom() {
		await tick();
		if (messagesEl) {
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}
	}

	async function pollTask(taskId: string, msgIndex: number) {
		let attempts = 0;
		const maxAttempts = 120;

		while (attempts < maxAttempts) {
			await new Promise((r) => setTimeout(r, 2000));
			attempts++;

			try {
				const task = await getTask(taskId);
				if (task.status === 'completed') {
					messages[msgIndex] = {
						...messages[msgIndex],
						content: task.result || 'Task completed.',
						status: 'completed'
					};
					return;
				} else if (task.status === 'failed') {
					messages[msgIndex] = {
						...messages[msgIndex],
						content: task.error || 'Task failed.',
						status: 'failed'
					};
					return;
				}
			} catch {
				/* keep polling */
			}
		}

		messages[msgIndex] = {
			...messages[msgIndex],
			content: 'Task timed out waiting for response.',
			status: 'failed'
		};
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		const prompt = input.trim();
		if (!prompt || sending) return;

		input = '';
		sending = true;

		const userMsg: ChatMessage = {
			id: crypto.randomUUID(),
			role: 'user',
			content: prompt
		};
		messages = [...messages, userMsg];

		const assistantMsg: ChatMessage = {
			id: crypto.randomUUID(),
			role: 'assistant',
			content: '',
			status: 'pending'
		};
		messages = [...messages, assistantMsg];
		const assistantIndex = messages.length - 1;

		await scrollToBottom();

		try {
			const res = await createTask({
				prompt,
				provider: provider || undefined,
				model: model || undefined
			});
			messages[assistantIndex] = {
				...messages[assistantIndex],
				taskId: res.task_id,
				status: 'running',
				content: ''
			};
			sending = false;

			pollTask(res.task_id, assistantIndex);
		} catch (err) {
			messages[assistantIndex] = {
				...messages[assistantIndex],
				content: err instanceof Error ? err.message : 'Failed to send',
				status: 'failed'
			};
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
</script>

<div class="chat">
	<div class="chat__messages" bind:this={messagesEl}>
		{#if messages.length === 0}
			<div class="chat__empty">
				<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
					<polyline points="3.27 6.96 12 12.01 20.73 6.96" />
					<line x1="12" y1="22.08" x2="12" y2="12" />
				</svg>
				<h2>Welcome{$currentUser ? `, ${$currentUser.name}` : ''}</h2>
				<p>Ask ForgeBox anything. Tasks run securely inside isolated VMs.</p>

				<form class="chat__input chat__input--centered" onsubmit={handleSubmit}>
					<textarea
						bind:value={input}
						onkeydown={handleKeydown}
						placeholder="Ask a question or describe a task..."
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
				<div class="chat__model">
					<ModelSelector providers={providersStore.providers} bind:provider bind:model disabled={sending} compact />
				</div>
			</div>
		{:else}
			{#each messages as msg}
				<div class="msg msg--{msg.role}">
					{#if msg.role === 'assistant' && msg.status === 'failed'}
						<div class="msg__error" role="alert">
							<div class="msg__error-head">
								<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
									<circle cx="12" cy="12" r="10" />
									<line x1="12" y1="8" x2="12" y2="12" />
									<line x1="12" y1="16" x2="12.01" y2="16" />
								</svg>
								<span>Task failed</span>
							</div>
							<pre class="msg__error-body">{msg.content}</pre>
						</div>
					{:else}
						<div class="msg__bubble">
							{#if msg.role === 'assistant' && msg.status === 'pending'}
								<span class="msg__dots">
									<span></span><span></span><span></span>
								</span>
							{:else if msg.role === 'assistant' && msg.status === 'running'}
								<span class="msg__running">
									<span class="msg__spinner"></span>
									Running task...
								</span>
							{:else}
								{msg.content}
							{/if}
						</div>
					{/if}
				</div>
			{/each}
		{/if}
	</div>

	{#if messages.length > 0}
		<div class="chat__footer">
			<div class="chat__model chat__model--inline">
				<ModelSelector providers={providersStore.providers} bind:provider bind:model disabled={sending} compact />
			</div>
			<form class="chat__input" onsubmit={handleSubmit}>
				<textarea
					bind:value={input}
					onkeydown={handleKeydown}
					placeholder="Ask a question or describe a task..."
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
	{/if}
</div>

<style lang="scss">
	.chat {
		display: flex;
		flex-direction: column;
		height: 100%;
		margin: (-$space-8) (-$space-6);
		padding: 0;

		&__messages {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-6;
			display: flex;
			flex-direction: column;
			gap: $space-4;
		}

		&__empty {
			flex: 1;
			display: flex;
			flex-direction: column;
			align-items: center;
			justify-content: center;
			gap: $space-3;
			color: $neutral-400;
			text-align: center;

			svg { opacity: 0.3; }

			h2 {
				font-size: $text-xl;
				font-weight: $font-semibold;
				color: $neutral-700;
			}

			p {
				font-size: $text-sm;
				color: $neutral-500;
				max-width: 400px;
			}
		}

		&__input {
			display: flex;
			align-items: flex-end;
			gap: $space-3;
			padding: $space-4 $space-6;
			border-top: 1px solid $neutral-200;
			background: $neutral-0;

			textarea {
				flex: 1;
				@include input-base;
				resize: none;
				min-height: 40px;
				max-height: 160px;
				padding: $space-2 $space-3;
			}

			button {
				flex-shrink: 0;
				width: 40px;
				height: 40px;
				padding: 0;
				display: flex;
				align-items: center;
				justify-content: center;
				border-radius: $radius-lg;
			}

			&--centered {
				border-top: none;
				padding: 0;
				margin-top: $space-8;
				width: 100%;
				max-width: 600px;
			}
		}

		&__footer {
			display: flex;
			flex-direction: column;
			border-top: 1px solid $neutral-200;
			background: $neutral-0;

			.chat__input {
				border-top: none;
				padding-top: $space-2;
			}
		}

		&__model {
			display: flex;
			align-items: center;
			justify-content: center;
			margin-top: $space-2;
			width: 100%;
			max-width: 600px;

			&--inline {
				justify-content: flex-start;
				padding: $space-3 $space-6 0;
				margin-top: 0;
				max-width: none;
			}
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
			max-width: 70%;
			padding: $space-3 $space-4;
			font-size: $text-sm;
			line-height: $leading-relaxed;
			white-space: pre-wrap;
			word-break: break-word;
		}

		&__error {
			max-width: 70%;
			padding: $space-3 $space-4;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
			color: $error-700;
			display: flex;
			flex-direction: column;
			gap: $space-2;
		}

		&__error-head {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $error-700;
		}

		&__error-body {
			margin: 0;
			padding: 0;
			font-family: $font-mono;
			font-size: $text-xs;
			line-height: $leading-relaxed;
			white-space: pre-wrap;
			word-break: break-word;
			color: $error-600;
		}

		&__running {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-sm;
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

		&__dots {
			display: inline-flex;
			gap: 4px;
			padding: $space-1 0;

			span {
				width: 6px;
				height: 6px;
				background: $neutral-400;
				border-radius: 50%;
				animation: dot-pulse 1.4s ease-in-out infinite;
			}

			span:nth-child(2) { animation-delay: 0.2s; }
			span:nth-child(3) { animation-delay: 0.4s; }
		}
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	@keyframes dot-pulse {
		0%, 80%, 100% { opacity: 0.3; transform: scale(0.8); }
		40% { opacity: 1; transform: scale(1); }
	}
</style>
