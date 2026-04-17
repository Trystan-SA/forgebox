<script lang="ts">
	import { goto } from '$app/navigation';
	import { createApp } from '$lib/api/client';
	import { tick } from 'svelte';

	interface ChatMessage {
		id: string;
		role: 'user' | 'assistant';
		content: string;
		status?: 'pending' | 'creating' | 'done';
	}

	let input = $state('');
	let sending = $state(false);
	let hasStarted = $state(false);
	let messagesEl: HTMLDivElement;
	let bottomInput: HTMLTextAreaElement;

	let messages = $state<ChatMessage[]>([
		{
			id: 'welcome',
			role: 'assistant',
			content: 'What do you want to build?'
		}
	]);

	async function scrollToBottom() {
		await tick();
		if (messagesEl) messagesEl.scrollTop = messagesEl.scrollHeight;
	}

	function extractName(text: string): string {
		const cleaned = text.replace(/^(i want to build|create|make|build)\s+(a|an|the)?\s*/i, '');
		const first = cleaned.split(/[.\n]/)[0].trim();
		if (first.length > 50) return first.slice(0, 50).trim();
		return first.charAt(0).toUpperCase() + first.slice(1);
	}

	function detectTools(text: string): string[] {
		const lower = text.toLowerCase();
		const tools: string[] = [];
		if (/databas|sql|postgres|sqlite|store|persist|record|table/i.test(lower)) tools.push('database');
		if (/api|http|fetch|endpoint|rest|webhook|external|integrat/i.test(lower)) tools.push('api');
		if (/ai|llm|gpt|claude|model|generat|analyz|summar|classify|nlp|intelligent/i.test(lower)) tools.push('ai');
		if (tools.length === 0) tools.push('ai');
		return tools;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		const prompt = input.trim();
		if (!prompt || sending) return;

		input = '';
		sending = true;
		hasStarted = true;

		await tick();
		if (bottomInput) bottomInput.focus();

		messages = [...messages, {
			id: crypto.randomUUID(),
			role: 'user',
			content: prompt
		}];

		const assistantMsg: ChatMessage = {
			id: crypto.randomUUID(),
			role: 'assistant',
			content: '',
			status: 'creating'
		};
		messages = [...messages, assistantMsg];
		await scrollToBottom();

		try {
			const name = extractName(prompt);
			const tools = detectTools(prompt);

			const app = await createApp({
				name,
				description: prompt,
				tools: JSON.stringify(tools)
			});

			const toolLabels = tools.map(t => t === 'database' ? 'Database' : t === 'api' ? 'API' : 'AI');
			messages[messages.length - 1] = {
				...assistantMsg,
				content: `Created "${app.name}" with ${toolLabels.join(', ')} access. Setting up your workspace...`,
				status: 'done'
			};
			await scrollToBottom();

			setTimeout(() => goto(`/apps/${app.id}`), 1200);
		} catch (err) {
			messages[messages.length - 1] = {
				...assistantMsg,
				content: err instanceof Error ? err.message : 'Something went wrong. Please try again.',
				status: 'done'
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

<div class="chat" class:chat--started={hasStarted}>
	<div class="chat__topbar">
		<a href="/apps" class="chat__back">
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
			Back to Apps
		</a>
	</div>

	<div class="chat__messages" bind:this={messagesEl}>
		<div class="chat__content">
			<div class="chat__icon">
				<svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
				</svg>
			</div>

			<div class="chat__thread">
				{#each messages as msg}
					<div class="msg msg--{msg.role}">
						<div class="msg__bubble">
							{#if msg.status === 'creating'}
								<span class="msg__creating">
									<span class="msg__spinner"></span>
									Creating your app...
								</span>
							{:else}
								{msg.content}
							{/if}
						</div>
					</div>
				{/each}
			</div>

			<div class="chat__welcome-input">
				<form class="chat__input-form" onsubmit={handleSubmit}>
					<textarea
						bind:value={input}
						onkeydown={handleKeydown}
						placeholder="Describe the tool you want to build..."
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

			<div class="chat__hints">
				<button class="hint" onclick={() => { input = 'An invoice generator that pulls order data from our API and creates PDF invoices'; }}>
					Invoice Generator
				</button>
				<button class="hint" onclick={() => { input = 'A dashboard that queries our database and shows key metrics with charts'; }}>
					Metrics Dashboard
				</button>
				<button class="hint" onclick={() => { input = 'A support ticket classifier that uses AI to categorize and prioritize incoming tickets'; }}>
					Ticket Classifier
				</button>
			</div>
		</div>
	</div>

	<div class="chat__bottom-bar">
		<form class="chat__input-form" onsubmit={handleSubmit}>
			<textarea
				bind:this={bottomInput}
				bind:value={input}
				onkeydown={handleKeydown}
				placeholder="Describe the tool you want to build..."
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

<style lang="scss">
	$transition-layout: 0.5s cubic-bezier(0.4, 0, 0.2, 1);

	.chat {
		display: flex;
		flex-direction: column;
		height: 100%;
		/* fullscreen layout — no margin needed */

		/* ---- Messages area ---- */
		&__messages {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			display: flex;
			flex-direction: column;
			align-items: center;
			padding: $space-6;
			padding-top: 25vh;
			transition: padding-top $transition-layout;
		}

		&--started &__messages {
			padding-top: $space-6;
		}

		/* ---- Content wrapper ---- */
		&__content {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: $space-5;
			width: 100%;
			max-width: 700px;
		}

		/* ---- Icon — fades + shrinks ---- */
		&__icon {
			@include flex-center;
			width: 64px;
			height: 64px;
			border-radius: $radius-xl;
			background: $primary-50;
			color: $primary-500;
			transition: opacity $transition-layout, transform $transition-layout, max-height $transition-layout, margin-bottom $transition-layout;
			max-height: 64px;
			overflow: hidden;
		}

		&--started &__icon {
			opacity: 0;
			transform: scale(0.6);
			max-height: 0;
			margin-bottom: -$space-5;
			pointer-events: none;
		}

		/* ---- Thread ---- */
		&__thread {
			display: flex;
			flex-direction: column;
			gap: $space-3;
			width: 100%;
		}

		/* ---- Inline input (welcome state) — fades + collapses ---- */
		&__welcome-input {
			width: 100%;
			max-height: 80px;
			opacity: 1;
			overflow: hidden;
			transition: opacity 0.3s ease, max-height 0.4s ease, margin-bottom 0.4s ease;
		}

		&--started &__welcome-input {
			opacity: 0;
			max-height: 0;
			margin-bottom: -$space-5;
			pointer-events: none;
		}

		/* ---- Hints — fade + collapse ---- */
		&__hints {
			display: flex;
			flex-wrap: wrap;
			justify-content: center;
			gap: $space-2;
			max-height: 60px;
			opacity: 1;
			overflow: hidden;
			transition: opacity 0.3s ease, max-height 0.4s ease;
		}

		&--started &__hints {
			opacity: 0;
			max-height: 0;
			pointer-events: none;
		}

		/* ---- Shared input form ---- */
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

		/* ---- Bottom bar — slides up ---- */
		&__bottom-bar {
			padding: $space-4 $space-6;
			border-top: 1px solid $neutral-200;
			background: $neutral-0;
			display: flex;
			justify-content: center;
			opacity: 0;
			transform: translateY(100%);
			max-height: 0;
			overflow: hidden;
			transition: opacity 0.4s ease 0.15s, transform 0.4s ease 0.15s, max-height 0.4s ease 0.15s, padding 0.4s ease 0.15s;
			padding-top: 0;
			padding-bottom: 0;
		}

		&--started &__bottom-bar {
			opacity: 1;
			transform: translateY(0);
			max-height: 120px;
			padding-top: $space-4;
			padding-bottom: $space-4;
		}

		&__bottom-bar &__input-form {
			max-width: 700px;
		}

		/* ---- Top bar ---- */
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

	/* ---- Message bubbles ---- */
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
	}

	.hint {
		padding: $space-2 $space-3;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-full;
		font-size: $text-xs;
		font-weight: $font-medium;
		color: $neutral-500;
		cursor: pointer;
		transition: all $transition-fast;
		white-space: nowrap;

		&:hover {
			border-color: $primary-300;
			color: $primary-700;
			background: $primary-50;
		}
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
