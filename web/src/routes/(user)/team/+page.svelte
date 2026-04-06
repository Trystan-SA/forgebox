<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Team</h1>
		<p>View team members, shared tasks, and workflows</p>
	</div>

	<EmptyState
		title="No team assigned"
		description="You'll see team members and shared resources here once you're assigned to a team."
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
