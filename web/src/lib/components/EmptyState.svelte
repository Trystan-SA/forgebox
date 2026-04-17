<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description: string;
		icon?: Snippet;
		action?: Snippet;
	}

	let { title, description, icon, action }: Props = $props();
</script>

<div class="empty">
	<div class="empty__icon">
		{#if icon}
			{@render icon()}
		{:else}
			<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
				<path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
			</svg>
		{/if}
	</div>
	<h3 class="empty__title">{title}</h3>
	<p class="empty__desc">{description}</p>
	{#if action}
		<div class="empty__action">
			{@render action()}
		</div>
	{/if}
</div>

<style lang="scss">
	.empty {
		@include flex-center;
		flex-direction: column;
		max-width: 420px;
		margin: 0 auto;
		padding: $space-16 $space-6;
		text-align: center;
		border: 2px dashed $neutral-200;
		border-radius: $radius-xl;

		&__icon { color: $neutral-400; margin-bottom: $space-3; }
		&__title { font-size: $text-sm; font-weight: $font-semibold; color: $neutral-900; }
		&__desc { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		&__action { margin-top: $space-4; }
	}
</style>
