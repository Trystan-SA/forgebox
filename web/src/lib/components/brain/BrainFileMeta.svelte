<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { BrainFile } from '$lib/api/types';

	interface Props {
		file: BrainFile;
		hashtags: string[];
	}

	let { file, hashtags }: Props = $props();

	const dispatch = createEventDispatcher<{ titleChange: string }>();

	let editTitle = $state(file.title);

	$effect(() => {
		editTitle = file.title;
	});

	function handleTitleBlur() {
		const trimmed = editTitle.trim();
		if (trimmed && trimmed !== file.title) {
			dispatch('titleChange', trimmed);
		}
	}

	function relativeTime(dateStr: string): string {
		const now = Date.now();
		const then = new Date(dateStr).getTime();
		const diffMs = now - then;
		const diffSec = Math.floor(diffMs / 1000);
		const diffMin = Math.floor(diffSec / 60);
		const diffHr = Math.floor(diffMin / 60);
		const diffDay = Math.floor(diffHr / 24);

		const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

		if (diffDay >= 1) return rtf.format(-diffDay, 'day');
		if (diffHr >= 1) return rtf.format(-diffHr, 'hour');
		if (diffMin >= 1) return rtf.format(-diffMin, 'minute');
		return rtf.format(-diffSec, 'second');
	}

	const createdByType = $derived(
		file.created_by === 'agent' ? 'agent' : 'user'
	);
</script>

<div class="brain-meta">
	<div class="brain-meta__top">
		<input
			class="brain-meta__title"
			type="text"
			bind:value={editTitle}
			onblur={handleTitleBlur}
			placeholder="File title"
		/>

		<div class="brain-meta__badges">
			<span
				class="brain-meta__badge"
				class:brain-meta__badge--agent={createdByType === 'agent'}
				class:brain-meta__badge--user={createdByType === 'user'}
			>
				{createdByType === 'agent' ? 'Agent' : 'User'}
			</span>
		</div>
	</div>

	<div class="brain-meta__timestamps">
		<span>Created {relativeTime(file.created_at)}</span>
		<span class="brain-meta__sep">·</span>
		<span>Updated {relativeTime(file.updated_at)}</span>
	</div>

	{#if hashtags.length > 0}
		<div class="brain-meta__tags">
			{#each hashtags as tag}
				<span class="brain-meta__tag">#{tag}</span>
			{/each}
		</div>
	{/if}
</div>

<style lang="scss">
	.brain-meta {
		padding: $space-3 $space-4;
		background: $neutral-0;
		border-top: 1px solid $neutral-100;
		border-radius: 0 0 $radius-xl $radius-xl;

		&__top {
			display: flex;
			align-items: center;
			gap: $space-3;
			margin-bottom: $space-1;
		}

		&__title {
			flex: 1;
			font-size: $text-base;
			font-weight: $font-semibold;
			color: $neutral-900;
			border: none;
			outline: none;
			background: transparent;
			padding: 2px 0;
			border-bottom: 1px solid transparent;
			transition: border-color $transition-fast;

			&:focus {
				border-bottom-color: $primary-400;
			}

			&::placeholder {
				color: $neutral-400;
			}
		}

		&__badges {
			display: flex;
			gap: $space-1;
		}

		&__badge {
			@include badge;
			font-size: 10px;

			&--agent {
				background: $neutral-100;
				color: $neutral-600;
			}

			&--user {
				background: $info-100;
				color: $info-600;
			}
		}

		&__timestamps {
			font-size: $text-xs;
			color: $neutral-400;
			display: flex;
			gap: $space-2;
			align-items: center;
		}

		&__sep {
			color: $neutral-300;
		}

		&__tags {
			display: flex;
			flex-wrap: wrap;
			gap: $space-1;
			margin-top: $space-2;
		}

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			padding: 2px 7px;
			border-radius: $radius-full;
			background: $primary-50;
			color: $primary-700;
		}
	}
</style>
