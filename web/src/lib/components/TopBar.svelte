<script lang="ts">
	import { auth, currentUser } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	interface Props {
		title?: string;
	}

	let { title }: Props = $props();

	function handleLogout() {
		auth.logout();
		goto('/login');
	}
</script>

<header class="topbar">
	<div class="topbar__left">
		{#if title}
			<h1 class="topbar__title">{title}</h1>
		{/if}
	</div>

	<div class="topbar__right">
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
