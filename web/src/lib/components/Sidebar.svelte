<script lang="ts">
	import { page } from '$app/state';

	interface NavItem {
		name: string;
		href: string;
		icon: string;
	}

	interface Props {
		items: NavItem[];
		title?: string;
	}

	let { items, title = 'ForgeBox' }: Props = $props();
</script>

<aside class="sidebar">
	<div class="sidebar__logo">
		<svg class="sidebar__icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
		</svg>
		<span class="sidebar__title">{title}</span>
	</div>

	<nav class="sidebar__nav">
		{#each items as item}
			{@const isActive = page.url.pathname === item.href ||
				(item.href !== '/' && page.url.pathname.startsWith(item.href))}
			<a
				href={item.href}
				class="sidebar__link"
				class:sidebar__link--active={isActive}
			>
				<span class="sidebar__link-icon">{item.icon}</span>
				{item.name}
			</a>
		{/each}
	</nav>

	<div class="sidebar__footer">
		<div class="sidebar__status">
			<span class="sidebar__dot"></span>
			<span>Gateway connected</span>
		</div>
		<p class="sidebar__version">ForgeBox v0.1.0</p>
	</div>
</aside>

<style lang="scss">
	.sidebar {
		display: flex;
		flex-direction: column;
		width: $sidebar-width;
		border-right: 1px solid $neutral-200;
		background: $neutral-0;
		height: 100%;

		&__logo {
			display: flex;
			align-items: center;
			gap: $space-2;
			height: $topbar-height;
			padding: 0 $space-6;
			border-bottom: 1px solid $neutral-200;
		}

		&__icon { color: $primary-600; }

		&__title {
			font-size: $text-lg;
			font-weight: $font-bold;
			color: $neutral-900;
		}

		&__nav {
			flex: 1;
			padding: $space-4 $space-3;
			display: flex;
			flex-direction: column;
			gap: $space-1;
			@include scrollbar-thin;
			overflow-y: auto;
		}

		&__link {
			display: flex;
			align-items: center;
			gap: $space-3;
			padding: $space-2 $space-3;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-600;
			border-radius: $radius-lg;
			transition: all $transition-fast;

			&:hover {
				background: $neutral-50;
				color: $neutral-900;
			}

			&--active {
				background: $primary-50;
				color: $primary-700;
			}
		}

		&__link-icon {
			width: 1.25rem;
			text-align: center;
			flex-shrink: 0;
		}

		&__footer {
			border-top: 1px solid $neutral-200;
			padding: $space-4;
		}

		&__status {
			display: flex;
			align-items: center;
			gap: $space-2;
			font-size: $text-xs;
			color: $neutral-500;
		}

		&__dot {
			width: 8px;
			height: 8px;
			border-radius: 50%;
			background: $success-500;
		}

		&__version {
			margin-top: $space-1;
			font-size: $text-xs;
			color: $neutral-400;
		}
	}
</style>
