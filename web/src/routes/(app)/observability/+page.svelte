<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Observability</h1>
		<p>Logs, task execution traces, and error rates</p>
	</div>

	<EmptyState
		title="No trace data"
		description="Execution traces and logs will appear here once the observability pipeline is connected."
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
