<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
</script>

<div class="page">
	<div class="page__header">
		<h1>Users & Teams</h1>
		<p>Manage users, roles, and team assignments</p>
	</div>

	<EmptyState
		title="No users to display"
		description="User management will be available when the auth backend is connected."
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
