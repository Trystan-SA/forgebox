<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import Toasts from '$lib/components/Toasts.svelte';
	import { providersStore, loadProviders } from '$lib/stores/providers.svelte';
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
	const isFlush = $derived(
		/^\/agents\/[^/]+\/brain$/.test(page.url.pathname)
	);
	let sidebarCollapsed = $state(false);
	const hasProviders = $derived(!providersStore.loaded || providersStore.providers.length > 0);

	$effect(() => {
		if (isFullscreen) sidebarCollapsed = true;
	});

	onMount(() => {
		loadProviders().catch(() => {
			// Treat fetch failure as "providers configured" so we don't show a
			// misleading warning indicator on transient API errors.
		});
	});

	const navGroups = $derived([
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
		}
	]);

	const settingsLink = $derived({
		name: 'Settings',
		href: '/settings',
		icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z',
		warning: !hasProviders,
		warningLabel: 'No providers configured'
	});
</script>

<div class="layout">
	<Sidebar groups={navGroups} footerLink={settingsLink} bind:collapsed={sidebarCollapsed} />
	<div class="layout__main">
		<main class="layout__content" class:layout__content--fullscreen={isFullscreen} class:layout__content--flush={isFlush}>
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

			&--flush {
				padding: 0;
			}
		}
	}
</style>
