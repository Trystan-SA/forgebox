<script lang="ts">
	import '$lib/styles/global.scss';
	import { isAuthenticated } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	$effect(() => {
		if (!$isAuthenticated && page.url.pathname !== '/login') {
			goto('/login');
		}
	});
</script>

{@render children()}
