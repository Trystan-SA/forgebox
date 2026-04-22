<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import * as brain from '$lib/stores/brain.svelte';

	const dispatch = createEventDispatcher<{ highlight: { fileIds: string[] } }>();

	let query = $state('');
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;
	let isOpen = $state(false);

	function handleInput() {
		if (debounceTimer) clearTimeout(debounceTimer);

		if (!query.trim()) {
			brain.search('');
			isOpen = false;
			dispatch('highlight', { fileIds: [] });
			return;
		}

		debounceTimer = setTimeout(async () => {
			await brain.search(query);
			isOpen = brain.state.searchResults.length > 0;
			dispatch('highlight', { fileIds: brain.state.searchResults.map((r) => r.id) });
		}, 300);
	}

	function handleSelectResult(fileId: string) {
		brain.selectFile(fileId);
		dispatch('highlight', { fileIds: brain.state.searchResults.map((r) => r.id) });
		isOpen = false;
	}

	function handleClear() {
		query = '';
		brain.search('');
		isOpen = false;
		dispatch('highlight', { fileIds: [] });
	}

	function handleBlur() {
		setTimeout(() => { isOpen = false; }, 150);
	}

	function scorePercent(score?: number): string {
		if (score == null) return '';
		return `${Math.round(score * 100)}%`;
	}
</script>

<div class="brain-search">
	<div class="brain-search__input-wrap">
		<svg class="brain-search__icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<circle cx="11" cy="11" r="8" />
			<path d="m21 21-4.35-4.35" />
		</svg>
		<input
			class="brain-search__input"
			type="text"
			placeholder="Search brain..."
			bind:value={query}
			oninput={handleInput}
			onfocus={() => { if (brain.state.searchResults.length > 0) isOpen = true; }}
			onblur={handleBlur}
		/>
		{#if query}
			<button class="brain-search__clear" type="button" onclick={handleClear} aria-label="Clear search">
				<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
		{/if}
	</div>

	{#if isOpen && brain.state.searchResults.length > 0}
		<div class="brain-search__dropdown">
			{#each brain.state.searchResults as result}
				<button
					type="button"
					class="brain-search__result"
					onclick={() => handleSelectResult(result.id)}
				>
					<div class="brain-search__result-top">
						<span class="brain-search__result-title">{result.title}</span>
						{#if result.score != null}
							<span class="brain-search__result-score">{scorePercent(result.score)}</span>
						{/if}
					</div>
					{#if result.content}
						<p class="brain-search__result-snippet">
							{result.content.slice(0, 120)}{result.content.length > 120 ? '…' : ''}
						</p>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.brain-search {
		position: relative;
		min-width: 220px;

		&__input-wrap {
			position: relative;
			display: flex;
			align-items: center;
		}

		&__icon {
			position: absolute;
			left: $space-3;
			color: $neutral-400;
			pointer-events: none;
		}

		&__input {
			@include input-base;
			padding-left: 2rem;
			padding-right: 2rem;
			font-size: $text-sm;
			border-radius: $radius-lg;
		}

		&__clear {
			position: absolute;
			right: $space-2;
			@include flex-center;
			width: 20px;
			height: 20px;
			border-radius: $radius-sm;
			color: $neutral-400;
			background: transparent;
			border: none;
			cursor: pointer;
			transition: color $transition-fast;

			&:hover { color: $neutral-700; }
		}

		&__dropdown {
			position: absolute;
			top: calc(100% + 4px);
			left: 0;
			right: 0;
			z-index: 50;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			box-shadow: $shadow-md;
			max-height: 320px;
			overflow-y: auto;
			@include scrollbar-thin;
		}

		&__result {
			display: block;
			width: 100%;
			text-align: left;
			padding: $space-3 $space-4;
			background: transparent;
			border: none;
			border-bottom: 1px solid $neutral-100;
			cursor: pointer;
			transition: background $transition-fast;

			&:last-child { border-bottom: none; }
			&:hover { background: $neutral-50; }
		}

		&__result-top {
			display: flex;
			align-items: center;
			justify-content: space-between;
			margin-bottom: $space-1;
		}

		&__result-title {
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-800;
		}

		&__result-score {
			font-family: $font-mono;
			font-size: 10px;
			color: $success-600;
			background: $success-50;
			padding: 1px 6px;
			border-radius: $radius-sm;
		}

		&__result-snippet {
			font-size: $text-xs;
			color: $neutral-500;
			line-height: $leading-relaxed;
			margin: 0;
			@include truncate;
		}
	}
</style>
