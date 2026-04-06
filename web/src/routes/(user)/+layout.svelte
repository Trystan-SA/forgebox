<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const navItems = [
		{ name: 'Home', href: '/home', icon: '🏠' },
		{ name: 'Task Runner', href: '/tasks/new', icon: '▶️' },
		{ name: 'My Tasks', href: '/tasks', icon: '📋' },
		{ name: 'Workflows', href: '/workflows', icon: '🔄' },
		{ name: 'Team', href: '/team', icon: '👥' },
		{ name: 'Settings', href: '/settings', icon: '⚙️' }
	];
</script>

<div class="layout">
	<Sidebar items={navItems} />
	<div class="layout__main">
		<TopBar />
		<main class="layout__content">
			{@render children()}
		</main>
	</div>
</div>

<style lang="scss">
	.layout {
		display: flex;
		height: 100vh;
		overflow: hidden;

		&__main {
			flex: 1;
			display: flex;
			flex-direction: column;
			overflow: hidden;
		}

		&__content {
			flex: 1;
			overflow-y: auto;
			@include scrollbar-thin;
			padding: $space-8 $space-6;
			max-width: 80rem;
		}
	}
</style>
