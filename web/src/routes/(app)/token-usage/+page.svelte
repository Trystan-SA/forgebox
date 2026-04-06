<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Token Usage</h1>
		<p>Usage statistics per user, team, and provider</p>
	</div>

	<EmptyState
		title="No usage data"
		description="Token usage tracking will appear here once tasks have been run."
	/>
</div>

<style lang="scss">
	.page {
		&__header {
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
