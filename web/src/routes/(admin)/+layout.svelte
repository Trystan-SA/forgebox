<script lang="ts">
	import Sidebar from '$lib/components/Sidebar.svelte';
	import TopBar from '$lib/components/TopBar.svelte';
	import { isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	const navItems = [
		{ name: 'Dashboard', href: '/dashboard', icon: '📊' },
		{ name: 'Users & Teams', href: '/users', icon: '👥' },
		{ name: 'Token Usage', href: '/token-usage', icon: '🔢' },
		{ name: 'Observability', href: '/observability', icon: '📈' },
		{ name: 'Providers', href: '/providers', icon: '🔌' },
		{ name: 'Channels', href: '/channels', icon: '📡' },
		{ name: 'VM Settings', href: '/vm-settings', icon: '⚙️' },
		{ name: 'Audit Log', href: '/audit', icon: '🛡️' }
	];

	$effect(() => {
		if (!$isAdmin) {
			goto('/home');
		}
	});
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
