<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { auth, currentUser } from '$lib/stores/auth';

	interface NavItem {
		name: string;
		href: string;
		icon: string;
		children?: NavItem[];
		warning?: boolean;
		warningLabel?: string;
	}

	interface NavGroup {
		label: string;
		items: NavItem[];
	}

	interface Props {
		groups: NavGroup[];
		title?: string;
		collapsed?: boolean;
		footerLink?: NavItem;
	}

	let { groups, title = 'ForgeBox', collapsed = $bindable(false), footerLink }: Props = $props();

	function isActive(href: string, exact = false): boolean {
		if (exact) return page.url.pathname === href;
		return page.url.pathname === href ||
			(href !== '/' && page.url.pathname.startsWith(href));
	}

	function isExpanded(item: NavItem): boolean {
		if (!item.children) return false;
		return isActive(item.href) || item.children.some((c) => isActive(c.href));
	}

	function handleLogout() {
		auth.logout();
		goto('/login');
	}

	function initials(value: string): string {
		return value
			.split(/[\s@.]+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((p) => p[0]?.toUpperCase() ?? '')
			.join('') || '?';
	}
</script>

<aside class="sb" class:sb--c={collapsed}>
	<div class="sb__head">
		<a href="/dashboard" class="sb__brand">
			<div class="sb__logo">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
					<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
				</svg>
			</div>
			{#if !collapsed}
				<span class="sb__brand-name">{title}</span>
			{/if}
		</a>
		<button class="sb__toggle" onclick={() => { collapsed = !collapsed; }} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				{#if collapsed}
					<polyline points="9 18 15 12 9 6" />
				{:else}
					<polyline points="15 18 9 12 15 6" />
				{/if}
			</svg>
		</button>
	</div>

	<nav class="sb__nav">
		{#each groups as group}
			<div class="sb__group">
				{#if !collapsed}
					<span class="sb__group-label">{group.label}</span>
				{:else}
					<span class="sb__divider"></span>
				{/if}
				{#each group.items as item}
					<a
						href={item.href}
						class="sb__link"
						class:sb__link--active={isActive(item.href) && !item.children}
						class:sb__link--parent={isExpanded(item)}
					>
						<span class="sb__link-icon">
							<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d={item.icon} /></svg>
							{#if item.warning && collapsed}
								<span class="sb__warn-dot" aria-hidden="true"></span>
							{/if}
						</span>
						{#if !collapsed}
							<span class="sb__link-name">{item.name}</span>
							{#if item.warning}
								<span class="sb__warn" title={item.warningLabel ?? `${item.name} needs attention`} aria-label={item.warningLabel ?? `${item.name} needs attention`}>
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>
								</span>
							{/if}
							{#if item.children}
								<svg class="sb__chevron" class:sb__chevron--open={isExpanded(item)} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6" /></svg>
							{/if}
						{/if}
						{#if collapsed}
							<span class="sb__tip">{item.warning ? `${item.name} — ${item.warningLabel ?? 'needs attention'}` : item.name}</span>
						{/if}
					</a>
					{#if !collapsed && item.children && isExpanded(item)}
						<div class="sb__sub">
							{#each item.children as child}
								<a href={child.href} class="sb__sub-link" class:sb__sub-link--active={isActive(child.href, true)}>
									<span class="sb__sub-dot"></span>
									<span>{child.name}</span>
								</a>
							{/each}
						</div>
					{/if}
				{/each}
			</div>
		{/each}
	</nav>

	{#if footerLink}
		<div class="sb__pre-foot">
			<a
				href={footerLink.href}
				class="sb__link"
				class:sb__link--active={isActive(footerLink.href)}
			>
				<span class="sb__link-icon">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d={footerLink.icon} /></svg>
					{#if footerLink.warning && collapsed}
						<span class="sb__warn-dot" aria-hidden="true"></span>
					{/if}
				</span>
				{#if !collapsed}
					<span class="sb__link-name">{footerLink.name}</span>
					{#if footerLink.warning}
						<span class="sb__warn" title={footerLink.warningLabel ?? `${footerLink.name} needs attention`} aria-label={footerLink.warningLabel ?? `${footerLink.name} needs attention`}>
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" /><line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>
						</span>
					{/if}
				{/if}
				{#if collapsed}
					<span class="sb__tip">{footerLink.warning ? `${footerLink.name} — ${footerLink.warningLabel ?? 'needs attention'}` : footerLink.name}</span>
				{/if}
			</a>
		</div>
	{/if}

	<div class="sb__foot">
		{#if $currentUser}
			<div class="sb__acct">
				<span class="sb__acct-avatar">{initials($currentUser.name || $currentUser.email)}</span>
				{#if !collapsed}
					<div class="sb__acct-info">
						<span class="sb__acct-name">{$currentUser.name || $currentUser.email.split('@')[0]}</span>
						<span class="sb__acct-email">{$currentUser.email}</span>
					</div>
					<button class="sb__logout" type="button" onclick={handleLogout} aria-label="Sign out">
						<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" /></svg>
					</button>
				{:else}
					<button class="sb__logout sb__logout--mini" type="button" onclick={handleLogout} aria-label="Sign out">
						<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><polyline points="16 17 21 12 16 7" /><line x1="21" y1="12" x2="9" y2="12" /></svg>
						<span class="sb__tip">Sign out</span>
					</button>
				{/if}
			</div>
		{/if}
		<div class="sb__status">
			<span class="sb__pulse"></span>
			{#if !collapsed}
				<span class="sb__foot-text">Connected</span>
				<span class="sb__foot-ver">v0.1.0</span>
			{/if}
		</div>
	</div>
</aside>

<style lang="scss">
	$sb-expanded: $sidebar-width;
	$sb-collapsed: 52px;
	$sb-bg: $neutral-900;
	$sb-hover: rgba(255, 255, 255, 0.06);
	$sb-active: rgba($primary-400, 0.12);
	$sb-border: rgba(255, 255, 255, 0.07);
	$sb-text: $neutral-400;
	$sb-bright: $neutral-100;
	$sb-accent: $primary-400;
	$sb-ease: cubic-bezier(0.4, 0, 0.2, 1);

	.sb {
		display: flex;
		flex-direction: column;
		width: $sb-expanded;
		height: 100%;
		background: $sb-bg;
		overflow: hidden;
		transition: width 0.22s $sb-ease;
		flex-shrink: 0;

		&--c {
			width: $sb-collapsed;
		}

		&__head {
			display: flex;
			align-items: center;
			height: $topbar-height;
			padding: 0 $space-2;
			flex-shrink: 0;
			border-bottom: 1px solid $sb-border;
			gap: $space-1;

			.sb--c & {
				flex-direction: column;
				height: auto;
				padding: $space-2;
				gap: $space-2;
			}
		}

		&__brand {
			display: flex;
			align-items: center;
			gap: $space-2;
			color: $sb-bright;
			text-decoration: none;
			flex: 1;
			min-width: 0;
			padding: 0 $space-1;

			.sb--c & {
				flex: 0;
				padding: 0;
			}
		}

		&__logo {
			@include flex-center;
			width: 32px;
			height: 32px;
			flex-shrink: 0;
			color: $sb-accent;
		}

		&__brand-name {
			font-size: $text-sm;
			font-weight: $font-bold;
			letter-spacing: 0.03em;
			white-space: nowrap;
			@include truncate;
		}

		&__toggle {
			@include flex-center;
			width: 28px;
			height: 28px;
			flex-shrink: 0;
			border: none;
			background: transparent;
			border-radius: $radius-md;
			color: $sb-text;
			cursor: pointer;
			transition: color $transition-fast, background $transition-fast;

			&:hover {
				background: $sb-hover;
				color: $sb-bright;
			}

			.sb--c & {
				width: 36px;
				height: 28px;
				border: 1px solid $sb-border;
				border-radius: $radius-lg;

				&:hover {
					border-color: rgba(255, 255, 255, 0.15);
				}
			}
		}

		&__nav {
			flex: 1;
			padding: $space-2;
			display: flex;
			flex-direction: column;
			gap: $space-3;
			overflow-y: auto;
			scrollbar-width: thin;
			scrollbar-color: rgba(255, 255, 255, 0.08) transparent;
		}

		&__group {
			display: flex;
			flex-direction: column;
			gap: 1px;
		}

		&__group-label {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-semibold;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: $sb-text;
			opacity: 0.45;
			padding: $space-1 $space-2;
			margin-bottom: 2px;
			white-space: nowrap;
		}

		&__divider {
			display: block;
			height: 1px;
			background: $sb-border;
			margin: $space-1 $space-2;
		}

		&__link {
			position: relative;
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: 7px $space-2;
			border-radius: $radius-lg;
			color: $sb-text;
			text-decoration: none;
			font-size: $text-sm;
			font-weight: $font-medium;
			white-space: nowrap;
			transition: color $transition-fast, background $transition-fast;

			.sb--c & {
				justify-content: center;
				padding: 8px;
			}

			&:hover {
				background: $sb-hover;
				color: $sb-bright;
			}

			&--active {
				background: $sb-active;
				color: $sb-accent;
			}

			&--parent {
				color: $sb-bright;
			}
		}

		&__link-icon {
			@include flex-center;
			position: relative;
			width: 20px;
			height: 20px;
			flex-shrink: 0;

			svg { display: block; }
		}

		&__warn {
			@include flex-center;
			flex-shrink: 0;
			color: $warning-500;
			margin-left: auto;

			svg {
				display: block;
				filter: drop-shadow(0 0 4px rgba($warning-500, 0.45));
			}
		}

		&__warn-dot {
			position: absolute;
			top: -2px;
			right: -2px;
			width: 8px;
			height: 8px;
			border-radius: 50%;
			background: $warning-500;
			box-shadow: 0 0 0 2px $sb-bg, 0 0 6px rgba($warning-500, 0.6);
		}

		&__link-name {
			flex: 1;
			min-width: 0;
			@include truncate;
		}

		&__chevron {
			flex-shrink: 0;
			color: $sb-text;
			opacity: 0.4;
			transition: transform $transition-fast;

			&--open { transform: rotate(90deg); }
		}

		&__tip {
			position: absolute;
			left: calc(100% + 10px);
			top: 50%;
			transform: translateY(-50%);
			background: $neutral-800;
			color: $sb-bright;
			font-size: $text-xs;
			font-weight: $font-medium;
			padding: 4px 10px;
			border-radius: $radius-md;
			white-space: nowrap;
			opacity: 0;
			pointer-events: none;
			transition: opacity 0.12s ease;
			z-index: 200;
			box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);

			&::before {
				content: '';
				position: absolute;
				right: 100%;
				top: 50%;
				transform: translateY(-50%);
				border: 4px solid transparent;
				border-right-color: $neutral-800;
			}

			.sb__link:hover & {
				opacity: 1;
			}
		}

		&__sub {
			display: flex;
			flex-direction: column;
			padding-left: 30px;
			margin: 2px 0 $space-1;
			position: relative;

			&::before {
				content: '';
				position: absolute;
				left: 19px;
				top: 0;
				bottom: 4px;
				width: 1px;
				background: $sb-border;
			}
		}

		&__sub-link {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: 5px $space-2;
			border-radius: $radius-md;
			color: $sb-text;
			text-decoration: none;
			font-size: $text-xs;
			font-weight: $font-medium;
			white-space: nowrap;
			transition: color $transition-fast, background $transition-fast;

			&:hover {
				color: $sb-bright;
				background: $sb-hover;
			}

			&--active {
				color: $sb-accent;

				.sb__sub-dot {
					background: $sb-accent;
					box-shadow: 0 0 0 2px rgba($primary-400, 0.2);
				}
			}
		}

		&__sub-dot {
			width: 5px;
			height: 5px;
			border-radius: 50%;
			background: rgba(255, 255, 255, 0.15);
			flex-shrink: 0;
			transition: all $transition-fast;
		}

		&__pre-foot {
			padding: 0 $space-2 $space-2;
			flex-shrink: 0;

			.sb--c & {
				align-items: center;
				padding: 0 $space-2 $space-2;
			}
		}

		&__foot {
			display: flex;
			flex-direction: column;
			gap: 2px;
			padding: $space-2;
			border-top: 1px solid $sb-border;
			flex-shrink: 0;

			.sb--c & {
				align-items: center;
				padding: $space-2;
			}
		}

		&__status {
			display: flex;
			align-items: center;
			gap: $space-2;
			padding: 4px $space-2 2px;
			opacity: 0.7;

			.sb--c & {
				justify-content: center;
				padding: 4px 0 2px;
			}
		}

		&__acct {
			display: flex;
			align-items: center;
			gap: $space-2;
			width: 100%;
			padding: 6px $space-1;
			border-radius: $radius-lg;
			min-width: 0;

			.sb--c & {
				padding: 0;
				justify-content: center;
			}
		}

		&__acct-avatar {
			@include flex-center;
			width: 28px;
			height: 28px;
			flex-shrink: 0;
			border-radius: 999px;
			background: rgba($primary-400, 0.18);
			color: $primary-300;
			font-size: 11px;
			font-weight: $font-semibold;
			letter-spacing: 0.02em;
			border: 1px solid rgba($primary-400, 0.25);
		}

		&__acct-info {
			flex: 1;
			min-width: 0;
			display: flex;
			flex-direction: column;
			gap: 1px;
			line-height: 1.1;
		}

		&__acct-name {
			font-size: $text-xs;
			font-weight: $font-semibold;
			color: $sb-bright;
			@include truncate;
		}

		&__acct-email {
			font-family: $font-mono;
			font-size: 10px;
			color: $sb-text;
			opacity: 0.7;
			@include truncate;
		}

		&__logout {
			@include flex-center;
			width: 26px;
			height: 26px;
			flex-shrink: 0;
			border: none;
			background: transparent;
			border-radius: $radius-md;
			color: $sb-text;
			opacity: 0.55;
			cursor: pointer;
			transition: color $transition-fast, background $transition-fast, opacity $transition-fast;

			&:hover {
				opacity: 1;
				color: $error-500;
				background: rgba($error-500, 0.08);
			}

			&--mini {
				position: relative;
			}
		}

		&__pulse {
			width: 7px;
			height: 7px;
			border-radius: 50%;
			background: $success-500;
			flex-shrink: 0;
			box-shadow: 0 0 0 2px rgba($success-500, 0.2);
		}

		&__foot-text {
			font-size: $text-xs;
			color: $sb-text;
		}

		&__foot-ver {
			font-family: $font-mono;
			font-size: 10px;
			color: $sb-text;
			opacity: 0.35;
			margin-left: auto;
		}
	}
</style>
