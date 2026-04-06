<script lang="ts">
	import '$lib/styles/global.scss';
	import { isAuthenticated } from '$lib/stores/auth';
	import { checkSetupStatus } from '$lib/api/client';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
	let ready = $state(false);

	const publicPaths = ['/login', '/setup'];

	onMount(async () => {
		if ($isAuthenticated) {
			ready = true;
			return;
		}

		try {
			const res = await checkSetupStatus();
			if (res.setup_required) {
				await goto('/setup');
			}
		} catch {
			/* backend not reachable — proceed */
		}

		ready = true;
	});

	$effect(() => {
		if (!ready) return;

		const path = page.url.pathname;

		if (!$isAuthenticated && !publicPaths.includes(path)) {
			goto('/login');
		} else if ($isAuthenticated && (path === '/' || path === '/login' || path === '/setup')) {
			goto('/dashboard');
		}
	});
</script>

{#if ready}
	{@render children()}
{/if}
