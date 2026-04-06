<script lang="ts">
	import { isAuthenticated, isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	$effect(() => {
		if ($isAuthenticated) {
			goto($isAdmin ? '/dashboard' : '/home');
		} else {
			goto('/login');
		}
	});
</script>
