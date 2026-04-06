<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const navGroups = [
		{
			label: 'Overview',
			items: [
				{ name: 'Dashboard', href: '/dashboard', icon: '📊' }
			]
		},
		{
			label: 'Tasks',
			items: [
				{ name: 'Run Task', href: '/tasks/new', icon: '▶️' },
				{ name: 'My Tasks', href: '/tasks', icon: '📋' },
				{
					name: 'Automations', href: '/automations', icon: '🔄',
					children: [
						{ name: 'All Automations', href: '/automations', icon: '' },
						{ name: 'Create New', href: '/automations/new', icon: '' }
					]
				}
			]
		},
		{
			label: 'Administration',
			items: [
				{ name: 'Users & Teams', href: '/users', icon: '👥' },
				{ name: 'Providers', href: '/providers', icon: '🔌' },
				{ name: 'Channels', href: '/channels', icon: '📡' },
				{ name: 'VM Settings', href: '/vm-settings', icon: '⚙️' },
				{ name: 'Audit Log', href: '/audit', icon: '🛡️' }
			]
		},
		{
			label: 'System',
			items: [
				{ name: 'Token Usage', href: '/token-usage', icon: '🔢' },
				{ name: 'Observability', href: '/observability', icon: '📈' },
				{ name: 'Settings', href: '/settings', icon: '⚙️' }
			]
		}
	];
</script>

<div class="layout">
	<Sidebar groups={navGroups} />
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
		}
	}
</style>
