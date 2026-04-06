<script lang="ts">
	import { page } from '$app/state';

	interface NavItem {
		name: string;
		href: string;
		icon: string;
		children?: NavItem[];
	}

	interface NavGroup {
		label: string;
		items: NavItem[];
	}

	interface Props {
		groups: NavGroup[];
		title?: string;
	}

	let { groups, title = 'ForgeBox' }: Props = $props();

	function isActive(href: string, exact = false): boolean {
		if (exact) return page.url.pathname === href;
		return page.url.pathname === href ||
			(href !== '/' && page.url.pathname.startsWith(href));
	}

	function isExpanded(item: NavItem): boolean {
		if (!item.children) return false;
		return isActive(item.href) || item.children.some((c) => isActive(c.href));
	}
</script>

<aside class="sidebar">
	<div class="sidebar__logo">
		<svg class="sidebar__icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
			<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
		</svg>
		<span class="sidebar__title">{title}</span>
	</div>

	<nav class="sidebar__nav">
		{#each groups as group}
			<div class="sidebar__group">
				<span class="sidebar__group-label">{group.label}</span>
				{#each group.items as item}
					<a
						href={item.href}
						class="sidebar__link"
						class:sidebar__link--active={isActive(item.href) && !item.children}
						class:sidebar__link--expanded={isExpanded(item)}
					>
						<span class="sidebar__link-icon">{item.icon}</span>
						{item.name}
						{#if item.children}
							<svg class="sidebar__chevron" class:sidebar__chevron--open={isExpanded(item)} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
								<polyline points="9 18 15 12 9 6" />
							</svg>
						{/if}
					</a>
					{#if item.children && isExpanded(item)}
						<div class="sidebar__sub">
							{#each item.children as child}
								<a
									href={child.href}
									class="sidebar__sublink"
									class:sidebar__sublink--active={isActive(child.href, true)}
								>
									{child.name}
								</a>
							{/each}
						</div>
					{/if}
				{/each}
			</div>
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
			gap: $space-5;
			@include scrollbar-thin;
			overflow-y: auto;
		}

		&__group {
			display: flex;
			flex-direction: column;
			gap: $space-1;
		}

		&__group-label {
			font-size: $text-xs;
			font-weight: $font-semibold;
			text-transform: uppercase;
			letter-spacing: 0.05em;
			color: $neutral-400;
			padding: 0 $space-3;
			margin-bottom: $space-1;
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

			&--expanded {
				color: $neutral-900;
				font-weight: $font-semibold;
			}
		}

		&__link-icon {
			width: 1.25rem;
			text-align: center;
			flex-shrink: 0;
		}

		&__chevron {
			margin-left: auto;
			color: $neutral-400;
			transition: transform $transition-fast;

			&--open {
				transform: rotate(90deg);
			}
		}

		&__sub {
			display: flex;
			flex-direction: column;
			margin-left: calc($space-3 + 0.625rem);
			padding-left: $space-4;
			padding-top: $space-1;
			padding-bottom: $space-1;
			position: relative;

			&::before {
				content: '';
				position: absolute;
				left: 0;
				top: 0;
				bottom: 0;
				width: 2px;
				background: $neutral-200;
				border-radius: 1px;
			}
		}

		&__sublink {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: 6px $space-3;
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-500;
			border-radius: $radius-md;
			transition: all $transition-fast;
			position: relative;

			&::before {
				content: '';
				width: 5px;
				height: 5px;
				border-radius: 50%;
				background: $neutral-300;
				flex-shrink: 0;
				transition: all $transition-fast;
			}

			&:hover {
				color: $neutral-800;
				background: $neutral-50;

				&::before {
					background: $neutral-500;
				}
			}

			&--active {
				color: $primary-700;
				background: $primary-50;

				&::before {
					background: $primary-500;
					box-shadow: 0 0 0 2px rgba($primary-500, 0.2);
				}
			}
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
