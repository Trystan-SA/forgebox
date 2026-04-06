<script lang="ts">
	import { createTask, getTask } from '$lib/api/client';
	import { currentUser } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		status?: 'pending' | 'running' | 'completed' | 'failed';
		taskId?: string;
	}

	let input = $state('');
	let messages = $state<ChatMessage[]>([]);
	let sending = $state(false);
	let messagesEl: HTMLDivElement;

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
			const res = await createTask({ prompt });
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

				<div class="quick-actions">
					<button class="quick-action" onclick={() => input = 'Using Claude Sonnet: '}>
						<span class="quick-action__icon">🧠</span>
						<span class="quick-action__label">Claude Sonnet</span>
					</button>
					<button class="quick-action" onclick={() => input = 'Using GPT-4o: '}>
						<span class="quick-action__icon">⚡</span>
						<span class="quick-action__label">GPT-4o</span>
					</button>
					<button class="quick-action" onclick={() => input = 'Using Ollama (local): '}>
						<span class="quick-action__icon">🏠</span>
						<span class="quick-action__label">Ollama Local</span>
					</button>
					<button class="quick-action" onclick={() => { goto('/automations'); }}>
						<span class="quick-action__icon">🔄</span>
						<span class="quick-action__label">Automations</span>
					</button>
					<button class="quick-action" onclick={() => input = 'Run an agent that: '}>
						<span class="quick-action__icon">🤖</span>
						<span class="quick-action__label">Run Agent</span>
					</button>
					<button class="quick-action" onclick={() => input = 'Analyze this code: '}>
						<span class="quick-action__icon">🔍</span>
						<span class="quick-action__label">Code Review</span>
					</button>
				</div>
			</div>
		{:else}
			{#each messages as msg}
				<div class="msg msg--{msg.role}">
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
					{#if msg.role === 'assistant' && msg.status === 'failed'}
						<span class="msg__status msg__status--error">Failed</span>
					{/if}
				</div>
			{/each}
		{/if}
	</div>

	{#if messages.length > 0}
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

		&__status {
			font-size: $text-xs;
			margin-top: $space-1;
			padding: 0 $space-1;

			&--error { color: $error-600; }
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

	.quick-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: $space-2;
		margin-top: $space-5;
		max-width: 600px;
		width: 100%;
	}

	.quick-action {
		display: flex;
		align-items: center;
		gap: $space-2;
		padding: $space-2 $space-3;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-full;
		font-size: $text-xs;
		font-weight: $font-medium;
		color: $neutral-600;
		cursor: pointer;
		transition: all $transition-fast;

		&:hover {
			border-color: $primary-300;
			color: $primary-700;
			background: $primary-50;
		}

		&__icon {
			font-size: $text-sm;
		}

		&__label {
			white-space: nowrap;
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
