<script lang="ts">
	import { page } from '$app/state';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Toasts from '$lib/components/Toasts.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const isFullscreen = $derived(
		/^\/automations\/[^/]+$/.test(page.url.pathname) ||
		/^\/apps\/new$/.test(page.url.pathname) ||
		/^\/apps\/[^/]+$/.test(page.url.pathname)
	);
	let sidebarCollapsed = $state(false);

	$effect(() => {
		if (isFullscreen) sidebarCollapsed = true;
	});

	const navGroups = [
		{
			label: 'Overview',
			items: [
				{ name: 'Dashboard', href: '/dashboard', icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0h4' }
			]
		},
		{
			label: 'Build',
			items: [
				{ name: 'Agents', href: '/agents', icon: 'M12 3v3 M18 8H6a2 2 0 00-2 2v6a2 2 0 002 2h12a2 2 0 002-2v-6a2 2 0 00-2-2z M9 13h0 M15 13h0 M8 21h8' },
				{ name: 'Apps', href: '/apps', icon: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z' },
				{
					name: 'Workflows', href: '/automations',
					icon: 'M3 12h9 M12 12l4-5h5 M12 12l4 5h5 M18 4l3 3-3 3 M18 14l3 3-3 3',
					children: [
						{ name: 'All Workflows', href: '/automations', icon: '' },
						{ name: 'Create New', href: '/automations/new', icon: '' }
					]
				}
			]
		},
		{
			label: 'Administration',
			items: [
				{ name: 'Team', href: '/users', icon: 'M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z' },
				{ name: 'Providers', href: '/providers', icon: 'M13 10V3L4 14h7v7l9-11h-7z' },
				{ name: 'Channels', href: '/channels', icon: 'M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.858 15.355-5.858 21.213 0' },
				{ name: 'VM Settings', href: '/vm-settings', icon: 'M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01' },
				{ name: 'Audit Log', href: '/audit', icon: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z' }
			]
		},
		{
			label: 'System',
			items: [
				{ name: 'Token Usage', href: '/token-usage', icon: 'M7 20l4-16m2 16l4-16M6 9h14M4 15h14' },
				{ name: 'Observability', href: '/observability', icon: 'M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z' },
				{ name: 'Settings', href: '/settings', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z' }
			]
		}
	];
</script>

<div class="layout">
	<Sidebar groups={navGroups} bind:collapsed={sidebarCollapsed} />
	<div class="layout__main">
		<main class="layout__content" class:layout__content--fullscreen={isFullscreen}>
			{@render children()}
		</main>
	</div>
</div>

<Toasts />

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

			&--fullscreen {
				padding: 0;
				overflow: hidden;
			}
		}
	}
</style>
