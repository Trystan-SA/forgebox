<script lang="ts">
	import { auth, currentUser, isAdmin, userRole } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	interface Props {
		title?: string;
	}

	let { title }: Props = $props();

	function handleLogout() {
		auth.logout();
		goto('/login');
	}

	function switchView() {
		const role = $userRole;
		if (role === 'admin') {
			// Toggle between admin and user view
			const isAdminRoute = window.location.pathname.startsWith('/dashboard') ||
				window.location.pathname.startsWith('/users') ||
				window.location.pathname.startsWith('/token-usage') ||
				window.location.pathname.startsWith('/observability') ||
				window.location.pathname.startsWith('/providers') ||
				window.location.pathname.startsWith('/channels') ||
				window.location.pathname.startsWith('/vm-settings') ||
				window.location.pathname.startsWith('/audit');

			goto(isAdminRoute ? '/home' : '/dashboard');
		}
	}
</script>

<header class="topbar">
	<div class="topbar__left">
		{#if title}
			<h1 class="topbar__title">{title}</h1>
		{/if}
	</div>

	<div class="topbar__right">
		{#if $isAdmin}
			<button class="topbar__switch" onclick={switchView}>
				Switch View
			</button>
		{/if}

		{#if $currentUser}
			<span class="topbar__user">{$currentUser.name || $currentUser.email}</span>
		{/if}

		<button class="topbar__logout" onclick={handleLogout}>
			Logout
		</button>
	</div>
</header>

<style lang="scss">
	.topbar {
		@include flex-between;
		height: $topbar-height;
		padding: 0 $space-6;
		border-bottom: 1px solid $neutral-200;
		background: $neutral-0;

		&__left { display: flex; align-items: center; }

		&__title {
			font-size: $text-lg;
			font-weight: $font-semibold;
			color: $neutral-900;
		}

		&__right {
			display: flex;
			align-items: center;
			gap: $space-4;
		}

		&__switch {
			@include btn;
			background: $primary-50;
			color: $primary-700;
			font-size: $text-xs;
			padding: $space-1 $space-3;

			&:hover { background: $primary-100; }
		}

		&__user {
			font-size: $text-sm;
			color: $neutral-600;
		}

		&__logout {
			@include btn;
			font-size: $text-sm;
			color: $neutral-500;
			background: transparent;

			&:hover { color: $error-600; }
		}
	}
</style>
