<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<div>
			<h1>Workflows</h1>
			<p>Create and manage reusable automation workflows</p>
		</div>
		<button class="btn-primary">New Workflow</button>
	</div>

	<EmptyState
		title="No workflows yet"
		description="Create a workflow to automate repetitive tasks with reusable templates."
	>
		{#snippet action()}
			<button class="btn-primary">Create Workflow</button>
		{/snippet}
	</EmptyState>
</div>

<style lang="scss">
	.page {
		&__header {
			@include flex-between;
			margin-bottom: $space-8;

			h1 { font-size: $text-2xl; font-weight: $font-bold; color: $neutral-900; }
			p { margin-top: $space-1; font-size: $text-sm; color: $neutral-500; }
		}
	}
</style>
