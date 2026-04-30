<script lang="ts">
	import { onMount } from 'svelte';
	import { pushToast } from '$lib/stores/toasts.svelte';

	type Role = 'owner' | 'admin' | 'editor' | 'viewer';
	type Status = 'active' | 'invited' | 'suspended';

	interface Member {
		id: string;
		name: string;
		email: string;
		role: Role;
		status: Status;
		last_active: string | null;
		created_at: string;
	}

	const STORAGE_KEY = 'forgebox_team_members';

	const roleMeta: Record<Role, { label: string; hint: string; tone: string }> = {
		owner: { label: 'Owner', hint: 'Full control, billing', tone: 'amber' },
		admin: { label: 'Admin', hint: 'Manage team & settings', tone: 'indigo' },
		editor: { label: 'Editor', hint: 'Create & edit resources', tone: 'teal' },
		viewer: { label: 'Viewer', hint: 'Read-only access', tone: 'slate' }
	};

	const roleOrder: Role[] = ['owner', 'admin', 'editor', 'viewer'];

	const seed: Member[] = [
		{
			id: 'u_001',
			name: 'Trystan Sarrade',
			email: 'trystan.sarrade@somanyways.co',
			role: 'owner',
			status: 'active',
			last_active: new Date(Date.now() - 2 * 60_000).toISOString(),
			created_at: '2026-01-04T10:00:00Z'
		},
		{
			id: 'u_002',
			name: 'Ines Moreau',
			email: 'ines@somanyways.co',
			role: 'admin',
			status: 'active',
			last_active: new Date(Date.now() - 18 * 60_000).toISOString(),
			created_at: '2026-01-10T10:00:00Z'
		},
		{
			id: 'u_003',
			name: 'Karim Ben-Saïd',
			email: 'karim@somanyways.co',
			role: 'editor',
			status: 'active',
			last_active: new Date(Date.now() - 3 * 3600_000).toISOString(),
			created_at: '2026-02-02T10:00:00Z'
		},
		{
			id: 'u_004',
			name: 'Priya Shah',
			email: 'priya.shah@contractor.dev',
			role: 'viewer',
			status: 'invited',
			last_active: null,
			created_at: '2026-04-12T10:00:00Z'
		},
		{
			id: 'u_005',
			name: 'Marcus Weiss',
			email: 'm.weiss@somanyways.co',
			role: 'editor',
			status: 'suspended',
			last_active: new Date(Date.now() - 21 * 24 * 3600_000).toISOString(),
			created_at: '2026-02-24T10:00:00Z'
		}
	];

	let members = $state<Member[]>([]);
	let query = $state('');
	let filter = $state<Role | 'all' | 'pending'>('all');
	let drawerOpen = $state(false);
	let drawerMode = $state<'invite' | 'edit'>('invite');
	let editId = $state<string | null>(null);
	let menuOpenFor = $state<string | null>(null);

	let draftName = $state('');
	let draftEmail = $state('');
	let draftRole = $state<Role>('editor');

	onMount(() => {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) {
			try {
				members = JSON.parse(raw);
				return;
			} catch {}
		}
		members = seed;
		localStorage.setItem(STORAGE_KEY, JSON.stringify(seed));
	});

	function persist() {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(members));
	}

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return members.filter((m) => {
			if (filter === 'pending') {
				if (m.status !== 'invited') return false;
			} else if (filter !== 'all' && m.role !== filter) {
				return false;
			}
			if (!q) return true;
			return m.name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q);
		});
	});

	function initials(name: string): string {
		return name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((p) => p[0]?.toUpperCase() ?? '')
			.join('');
	}

	function avatarTone(name: string): string {
		let h = 0;
		for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
		const hue = h % 360;
		return `hsl(${hue}deg 62% 55%)`;
	}

	function relTime(iso: string | null): string {
		if (!iso) return '—';
		const diff = Date.now() - new Date(iso).getTime();
		const m = Math.floor(diff / 60_000);
		if (m < 1) return 'just now';
		if (m < 60) return `${m}m ago`;
		const h = Math.floor(m / 60);
		if (h < 24) return `${h}h ago`;
		const d = Math.floor(h / 24);
		if (d < 30) return `${d}d ago`;
		return new Date(iso).toLocaleDateString();
	}

	function openInvite() {
		drawerMode = 'invite';
		editId = null;
		draftName = '';
		draftEmail = '';
		draftRole = 'editor';
		drawerOpen = true;
	}

	function openEdit(m: Member) {
		drawerMode = 'edit';
		editId = m.id;
		draftName = m.name;
		draftEmail = m.email;
		draftRole = m.role;
		drawerOpen = true;
		menuOpenFor = null;
	}

	function closeDrawer() {
		drawerOpen = false;
	}

	function handleSubmit(e: Event) {
		e.preventDefault();
		const name = draftName.trim();
		const email = draftEmail.trim();
		if (!name || !email) return;
		if (drawerMode === 'invite') {
			const m: Member = {
				id: 'u_' + crypto.randomUUID().slice(0, 6),
				name,
				email,
				role: draftRole,
				status: 'invited',
				last_active: null,
				created_at: new Date().toISOString()
			};
			members = [...members, m];
			pushToast(`Invite sent to ${email}`, 'success');
		} else if (editId) {
			members = members.map((m) =>
				m.id === editId ? { ...m, name, email, role: draftRole } : m
			);
			pushToast('Member updated', 'success');
		}
		persist();
		drawerOpen = false;
	}

	function updateRole(id: string, role: Role) {
		members = members.map((m) => (m.id === id ? { ...m, role } : m));
		persist();
		pushToast('Role updated', 'success');
		menuOpenFor = null;
	}

	function toggleStatus(m: Member) {
		const next: Status = m.status === 'active' ? 'suspended' : 'active';
		members = members.map((x) => (x.id === m.id ? { ...x, status: next } : x));
		persist();
		pushToast(next === 'active' ? 'Member reinstated' : 'Member suspended', 'success');
		menuOpenFor = null;
	}

	function removeMember(m: Member) {
		if (m.role === 'owner') {
			pushToast('Cannot remove the owner', 'error', 4000);
			menuOpenFor = null;
			return;
		}
		members = members.filter((x) => x.id !== m.id);
		persist();
		pushToast(`${m.name} removed`, 'success');
		menuOpenFor = null;
	}

	function resendInvite(m: Member) {
		pushToast(`Invite re-sent to ${m.email}`, 'success');
		menuOpenFor = null;
	}

	function onKey(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			if (drawerOpen) closeDrawer();
			else if (menuOpenFor) menuOpenFor = null;
		}
	}
</script>

<svelte:window onkeydown={onKey} />

<section class="section">
	<h2>Team</h2>
	<p class="section__hint">Invite teammates, assign roles, and manage access.</p>
	<div class="toolbar">
		<button type="button" class="invite-btn" onclick={openInvite}>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></svg>
			<span>Invite member</span>
		</button>
		<label class="search">
			<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
			<input
				type="text"
				placeholder="Search by name or email…"
				bind:value={query}
			/>
			{#if query}
				<button class="search__clear" type="button" onclick={() => { query = ''; }} aria-label="Clear search">
					<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
				</button>
			{/if}
		</label>

		<div class="filter">
			<span class="filter__lbl">role</span>
			<button class="filter__chip" class:filter__chip--on={filter === 'all'} onclick={() => { filter = 'all'; }}>All</button>
			{#each roleOrder as r}
				<button
					class="filter__chip filter__chip--{roleMeta[r].tone}"
					class:filter__chip--on={filter === r}
					onclick={() => { filter = r; }}
				>
					{roleMeta[r].label}
				</button>
			{/each}
		</div>
	</div>

	<div class="table">
		<div class="row row--head" role="row">
			<span class="col col--member">Member</span>
			<span class="col col--role">Role</span>
			<span class="col col--status">Status</span>
			<span class="col col--active">Last active</span>
			<span class="col col--actions"></span>
		</div>

		{#if filtered.length === 0}
			<div class="empty">
				<div class="empty__icon">
					<svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
				</div>
				<p class="empty__title">No matches</p>
				<p class="empty__desc">
					{query ? `Nothing matches "${query}"` : 'No members with this filter'}
				</p>
			</div>
		{:else}
			{#each filtered as m (m.id)}
				<div class="row">
					<div class="col col--member">
						<div class="avatar" style="--tone: {avatarTone(m.name)}">
							<span>{initials(m.name)}</span>
						</div>
						<div class="ident">
							<span class="ident__name">{m.name}</span>
							<span class="ident__email">{m.email}</span>
						</div>
					</div>

					<div class="col col--role">
						<span class="pill pill--{roleMeta[m.role].tone}" title={roleMeta[m.role].hint}>
							<span class="pill__dot"></span>
							{roleMeta[m.role].label}
						</span>
					</div>

					<div class="col col--status">
						<span class="status status--{m.status}">
							<span class="status__dot"></span>
							{m.status}
						</span>
					</div>

					<div class="col col--active">
						<span class="active-time">{relTime(m.last_active)}</span>
					</div>

					<div class="col col--actions">
						<button
							class="kebab"
							type="button"
							aria-label="Member actions"
							onclick={(e) => { e.stopPropagation(); menuOpenFor = menuOpenFor === m.id ? null : m.id; }}
						>
							<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="5" cy="12" r="1.6" /><circle cx="12" cy="12" r="1.6" /><circle cx="19" cy="12" r="1.6" /></svg>
						</button>
						{#if menuOpenFor === m.id}
							<button class="menu__overlay" onclick={() => { menuOpenFor = null; }} aria-label="Close"></button>
							<div class="menu" role="menu">
								<div class="menu__head">
									<span class="menu__tag">member</span>
									<span class="menu__name">{m.name}</span>
								</div>
								<button class="menu__item" onclick={() => openEdit(m)}>
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9" /><path d="M16.5 3.5a2.121 2.121 0 013 3L7 19l-4 1 1-4 12.5-12.5z" /></svg>
									Edit details
								</button>
								<div class="menu__section">Change role</div>
								{#each roleOrder as r}
									<button
										class="menu__item menu__item--role"
										class:menu__item--active={m.role === r}
										onclick={() => updateRole(m.id, r)}
									>
										<span class="menu__swatch menu__swatch--{roleMeta[r].tone}"></span>
										{roleMeta[r].label}
										<span class="menu__hint">{roleMeta[r].hint}</span>
									</button>
								{/each}
								<div class="menu__sep"></div>
								{#if m.status === 'invited'}
									<button class="menu__item" onclick={() => resendInvite(m)}>
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 11-6.219-8.56" /><polyline points="21 3 21 9 15 9" /></svg>
										Resend invite
									</button>
								{:else if m.role !== 'owner'}
									<button class="menu__item" onclick={() => toggleStatus(m)}>
										{#if m.status === 'active'}
											<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><line x1="4.93" y1="4.93" x2="19.07" y2="19.07" /></svg>
											Suspend
										{:else}
											<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12" /></svg>
											Reinstate
										{/if}
									</button>
								{/if}
								{#if m.role !== 'owner'}
									<button class="menu__item menu__item--danger" onclick={() => removeMember(m)}>
										<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6" /><path d="M19 6l-2 14a2 2 0 01-2 2H9a2 2 0 01-2-2L5 6" /></svg>
										Remove
									</button>
								{/if}
							</div>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
</section>

{#if drawerOpen}
	<button class="drawer-scrim" type="button" onclick={closeDrawer} aria-label="Close"></button>
	<aside class="drawer" role="dialog" aria-label={drawerMode === 'invite' ? 'Invite member' : 'Edit member'}>
		<div class="drawer__head">
			<span class="drawer__tag">{drawerMode === 'invite' ? 'new member' : 'edit'}</span>
			<button class="drawer__close" type="button" onclick={closeDrawer} aria-label="Close">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
			</button>
		</div>

		<form class="drawer__body" onsubmit={handleSubmit}>
			<h2 class="drawer__title">
				{drawerMode === 'invite' ? 'Invite a teammate' : 'Edit member'}
			</h2>
			<p class="drawer__sub">
				{#if drawerMode === 'invite'}
					They'll receive an email with a link to set up their account.
				{:else}
					Changes take effect immediately. The member will be notified if their role changes.
				{/if}
			</p>

			<label class="fld">
				<span class="fld__lbl">Full name</span>
				<input
					class="fld__input"
					type="text"
					bind:value={draftName}
					placeholder="e.g. Alex Rivera"
					required
				/>
			</label>

			<label class="fld">
				<span class="fld__lbl">Email address</span>
				<input
					class="fld__input"
					type="email"
					bind:value={draftEmail}
					placeholder="alex@company.com"
					required
				/>
			</label>

			<div class="fld">
				<span class="fld__lbl">Role</span>
				<div class="roles">
					{#each roleOrder.filter((r) => r !== 'owner') as r}
						<button
							type="button"
							class="roles__btn roles__btn--{roleMeta[r].tone}"
							class:roles__btn--on={draftRole === r}
							onclick={() => { draftRole = r; }}
						>
							<span class="roles__top">
								<span class="roles__swatch"></span>
								<strong>{roleMeta[r].label}</strong>
							</span>
							<span class="roles__hint">{roleMeta[r].hint}</span>
						</button>
					{/each}
				</div>
			</div>

			<div class="drawer__foot">
				<button type="button" class="btn-ghost" onclick={closeDrawer}>Cancel</button>
				<button type="submit" class="btn-primary">
					{drawerMode === 'invite' ? 'Send invite' : 'Save changes'}
				</button>
			</div>
		</form>
	</aside>
{/if}

<style lang="scss">
	$tone-indigo: $primary-500;
	$tone-amber: $warning-500;
	$tone-teal: #0d9488;
	$tone-slate: $neutral-500;

	.section {
		h2 { margin-bottom: $space-4; }

		&__hint {
			margin: -$space-2 0 $space-4;
			font-size: $text-sm;
			color: $neutral-500;
		}
	}

	.invite-btn {
		@include flex-center;
		gap: $space-2;
		padding: 9px $space-4;
		font-size: $text-sm;
		font-weight: $font-semibold;
		color: $neutral-0;
		background: $primary-500;
		border: none;
		border-radius: $radius-md;
		cursor: pointer;
		transition: background $transition-fast;
		flex-shrink: 0;

		&:hover { background: $primary-600; }

		svg { color: $neutral-0; }
	}

	/* ---------- TOOLBAR ---------- */
	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: $space-4;
		margin-bottom: $space-4;
		flex-wrap: wrap;
	}

	.search {
		position: relative;
		display: flex;
		align-items: center;
		gap: $space-2;
		width: 340px;
		max-width: 100%;
		padding: 0 $space-3;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-lg;
		transition: border-color $transition-fast, box-shadow $transition-fast;

		&:focus-within {
			border-color: $primary-500;
			box-shadow: 0 0 0 3px rgba($primary-500, 0.12);
		}

		svg { color: $neutral-400; flex-shrink: 0; }

		input {
			flex: 1;
			padding: 9px 0;
			font-size: $text-sm;
			background: transparent;
			border: none;
			outline: none;
			color: $neutral-900;

			&::placeholder { color: $neutral-400; }
		}

		&__clear {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 20px;
			height: 20px;
			border: none;
			background: $neutral-100;
			border-radius: 999px;
			color: $neutral-500;
			cursor: pointer;

			&:hover { background: $neutral-200; color: $neutral-700; }
		}
	}

	.filter {
		display: flex;
		align-items: center;
		gap: $space-1;

		&__lbl {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-semibold;
			letter-spacing: 0.14em;
			text-transform: uppercase;
			color: $neutral-400;
			margin-right: $space-2;
		}

		&__chip {
			padding: 6px $space-3;
			font-size: $text-xs;
			font-weight: $font-medium;
			color: $neutral-600;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: 999px;
			cursor: pointer;
			transition: all $transition-fast;

			&:hover { border-color: $neutral-300; color: $neutral-800; }

			&--on {
				color: $neutral-0;
				background: $neutral-900;
				border-color: $neutral-900;
			}

			&--indigo.filter__chip--on { background: $primary-600; border-color: $primary-600; }
			&--amber.filter__chip--on { background: $warning-600; border-color: $warning-600; }
			&--teal.filter__chip--on { background: $tone-teal; border-color: $tone-teal; }
			&--slate.filter__chip--on { background: $neutral-700; border-color: $neutral-700; }
		}
	}

	/* ---------- TABLE ---------- */
	.table {
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-xl;
		overflow: hidden;
	}

	.row {
		display: grid;
		grid-template-columns: minmax(260px, 2.4fr) 180px 140px 140px 56px;
		align-items: center;
		padding: $space-3 $space-5;
		border-bottom: 1px solid $neutral-100;
		transition: background $transition-fast;

		&:last-child { border-bottom: none; }

		&--head {
			background: $neutral-50;
			border-bottom: 1px solid $neutral-200;
			padding-top: $space-3;
			padding-bottom: $space-3;

			.col {
				font-family: $font-mono;
				font-size: 10px;
				font-weight: $font-semibold;
				letter-spacing: 0.14em;
				text-transform: uppercase;
				color: $neutral-400;
			}
		}

		&:not(.row--head):hover {
			background: $neutral-50;
		}
	}

	.col {
		font-size: $text-sm;
		min-width: 0;

		&--member {
			display: flex;
			align-items: center;
			gap: $space-3;
		}

		&--actions {
			position: relative;
			text-align: right;
		}
	}

	.avatar {
		@include flex-center;
		width: 34px;
		height: 34px;
		flex-shrink: 0;
		border-radius: 999px;
		background: var(--tone);
		color: $neutral-0;
		font-size: 12px;
		font-weight: $font-semibold;
		letter-spacing: 0.02em;
		box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.1);
	}

	.ident {
		display: flex;
		flex-direction: column;
		min-width: 0;

		&__name {
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-900;
			@include truncate;
		}

		&__email {
			font-size: $text-xs;
			color: $neutral-500;
			font-family: $font-mono;
			@include truncate;
		}
	}

	.pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 3px 10px 3px 8px;
		font-size: 11px;
		font-weight: $font-semibold;
		letter-spacing: 0.02em;
		border-radius: 999px;
		border: 1px solid transparent;

		&__dot {
			width: 6px;
			height: 6px;
			border-radius: 999px;
		}

		&--indigo {
			color: $primary-700;
			background: $primary-50;
			border-color: $primary-100;
			.pill__dot { background: $primary-500; }
		}

		&--amber {
			color: $warning-700;
			background: $warning-50;
			border-color: $warning-100;
			.pill__dot { background: $warning-500; }
		}

		&--teal {
			color: #0f766e;
			background: #ccfbf1;
			border-color: #99f6e4;
			.pill__dot { background: $tone-teal; }
		}

		&--slate {
			color: $neutral-700;
			background: $neutral-100;
			border-color: $neutral-200;
			.pill__dot { background: $neutral-500; }
		}
	}

	.status {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 11px;
		font-weight: $font-medium;
		color: $neutral-600;
		text-transform: capitalize;

		&__dot {
			width: 7px;
			height: 7px;
			border-radius: 999px;
			background: $neutral-400;
		}

		&--active {
			color: $success-700;

			.status__dot {
				background: $success-500;
				box-shadow: 0 0 0 3px rgba($success-500, 0.2);
				animation: pulse-dot 2s ease-in-out infinite;
			}
		}

		&--invited {
			color: $warning-700;
			.status__dot { background: $warning-500; }
		}

		&--suspended {
			color: $error-700;
			.status__dot { background: $error-500; }
		}
	}

	@keyframes pulse-dot {
		0%, 100% { box-shadow: 0 0 0 3px rgba($success-500, 0.2); }
		50% { box-shadow: 0 0 0 5px rgba($success-500, 0.1); }
	}

	.active-time {
		font-family: $font-mono;
		font-size: $text-xs;
		color: $neutral-500;
	}

	.kebab {
		@include flex-center;
		width: 28px;
		height: 28px;
		margin-left: auto;
		background: transparent;
		border: none;
		border-radius: $radius-md;
		color: $neutral-400;
		cursor: pointer;
		transition: background $transition-fast, color $transition-fast;

		&:hover { background: $neutral-100; color: $neutral-800; }
	}

	/* ---------- MENU ---------- */
	.menu {
		position: absolute;
		top: calc(100% - 4px);
		right: 0;
		z-index: 30;
		min-width: 260px;
		background: $neutral-0;
		border: 1px solid $neutral-200;
		border-radius: $radius-xl;
		box-shadow: $shadow-lg;
		padding: $space-1;
		animation: menu-in 0.14s cubic-bezier(0.16, 1, 0.3, 1);
		text-align: left;

		&__overlay {
			position: fixed;
			inset: 0;
			z-index: 29;
			background: transparent;
			border: none;
			cursor: default;
		}

		&__head {
			display: flex;
			flex-direction: column;
			gap: 2px;
			padding: $space-2 $space-3 $space-1;
			border-bottom: 1px solid $neutral-100;
			margin-bottom: $space-1;
		}

		&__tag {
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-bold;
			letter-spacing: 0.12em;
			text-transform: uppercase;
			color: $neutral-400;
		}

		&__name {
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-900;
			@include truncate;
		}

		&__section {
			padding: $space-2 $space-3 4px;
			font-family: $font-mono;
			font-size: 9px;
			font-weight: $font-semibold;
			letter-spacing: 0.12em;
			text-transform: uppercase;
			color: $neutral-400;
		}

		&__sep {
			height: 1px;
			background: $neutral-100;
			margin: 4px 0;
		}

		&__item {
			display: flex;
			align-items: center;
			gap: $space-2;
			width: 100%;
			padding: 8px $space-3;
			font-size: $text-sm;
			font-weight: $font-medium;
			color: $neutral-700;
			background: transparent;
			border: none;
			border-radius: $radius-md;
			cursor: pointer;
			text-align: left;
			transition: background $transition-fast, color $transition-fast;

			&:hover { background: $neutral-100; color: $neutral-900; }

			&--role {
				position: relative;
				padding-right: $space-5;
			}

			&--active {
				background: $primary-50;
				color: $primary-700;

				&:hover { background: $primary-100; }
			}

			&--danger {
				color: $error-700;
				&:hover { background: $error-50; color: $error-700; }
			}
		}

		&__swatch {
			width: 8px;
			height: 8px;
			border-radius: 2px;
			flex-shrink: 0;

			&--indigo { background: $primary-500; }
			&--amber { background: $warning-500; }
			&--teal { background: $tone-teal; }
			&--slate { background: $neutral-500; }
		}

		&__hint {
			margin-left: auto;
			font-size: 11px;
			color: $neutral-400;
			font-weight: $font-normal;
		}
	}

	@keyframes menu-in {
		from { opacity: 0; transform: translateY(-4px) scale(0.98); }
		to { opacity: 1; transform: translateY(0) scale(1); }
	}

	/* ---------- EMPTY ---------- */
	.empty {
		padding: $space-12 $space-4;
		text-align: center;

		&__icon {
			@include flex-center;
			width: 48px;
			height: 48px;
			margin: 0 auto $space-3;
			border-radius: 999px;
			background: $neutral-100;
			color: $neutral-400;
		}

		&__title {
			font-size: $text-sm;
			font-weight: $font-semibold;
			color: $neutral-800;
		}

		&__desc {
			margin-top: 2px;
			font-size: $text-sm;
			color: $neutral-500;
		}
	}

	/* ---------- DRAWER ---------- */
	.drawer-scrim {
		position: fixed;
		inset: 0;
		z-index: 60;
		background: rgba($neutral-900, 0.4);
		backdrop-filter: blur(2px);
		border: none;
		cursor: default;
		animation: scrim-in 0.18s ease-out;
	}

	.drawer {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		z-index: 61;
		width: 440px;
		max-width: 92vw;
		background: $neutral-0;
		border-left: 1px solid $neutral-200;
		box-shadow: -24px 0 48px -12px rgba(0, 0, 0, 0.15);
		display: flex;
		flex-direction: column;
		animation: drawer-in 0.26s cubic-bezier(0.16, 1, 0.3, 1);

		&__head {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: $space-3 $space-5;
			border-bottom: 1px solid $neutral-100;
		}

		&__tag {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-bold;
			letter-spacing: 0.14em;
			text-transform: uppercase;
			color: $primary-700;
			background: $primary-50;
			padding: 3px 8px;
			border-radius: 999px;
		}

		&__close {
			@include flex-center;
			width: 28px;
			height: 28px;
			border: none;
			background: transparent;
			border-radius: $radius-md;
			color: $neutral-400;
			cursor: pointer;

			&:hover { background: $neutral-100; color: $neutral-700; }
		}

		&__body {
			display: flex;
			flex-direction: column;
			gap: $space-4;
			padding: $space-6 $space-5;
			flex: 1;
			overflow-y: auto;
		}

		&__title {
			font-size: $text-xl;
			font-weight: $font-bold;
			color: $neutral-900;
			line-height: 1.2;
		}

		&__sub {
			margin-top: -$space-2;
			font-size: $text-sm;
			color: $neutral-500;
			line-height: 1.5;
		}

		&__foot {
			display: flex;
			justify-content: flex-end;
			gap: $space-2;
			padding-top: $space-4;
			margin-top: auto;
			border-top: 1px solid $neutral-100;
		}
	}

	@keyframes scrim-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@keyframes drawer-in {
		from { transform: translateX(12px); opacity: 0; }
		to { transform: translateX(0); opacity: 1; }
	}

	/* ---------- FORM ---------- */
	.fld {
		display: flex;
		flex-direction: column;
		gap: 6px;

		&__lbl {
			font-family: $font-mono;
			font-size: 10px;
			font-weight: $font-semibold;
			letter-spacing: 0.12em;
			text-transform: uppercase;
			color: $neutral-500;
		}

		&__input {
			@include input-base;
			font-size: $text-sm;
			padding: 9px $space-3;
			border-radius: $radius-md;
		}
	}

	.roles {
		display: grid;
		grid-template-columns: 1fr;
		gap: 8px;

		&__btn {
			display: flex;
			flex-direction: column;
			align-items: flex-start;
			gap: 2px;
			padding: $space-3;
			background: $neutral-0;
			border: 1px solid $neutral-200;
			border-radius: $radius-lg;
			cursor: pointer;
			text-align: left;
			transition: all $transition-fast;

			&:hover { border-color: $neutral-300; }

			&--on {
				border-color: $primary-500;
				box-shadow: 0 0 0 3px rgba($primary-500, 0.12);
				background: $primary-50;
			}
		}

		&__top {
			display: flex;
			align-items: center;
			gap: 8px;

			strong {
				font-size: $text-sm;
				font-weight: $font-semibold;
				color: $neutral-900;
			}
		}

		&__swatch {
			width: 10px;
			height: 10px;
			border-radius: 3px;
			background: $neutral-400;

			.roles__btn--indigo & { background: $primary-500; }
			.roles__btn--amber & { background: $warning-500; }
			.roles__btn--teal & { background: $tone-teal; }
			.roles__btn--slate & { background: $neutral-500; }
		}

		&__hint {
			font-size: $text-xs;
			color: $neutral-500;
			margin-left: 18px;
		}
	}

	@include md {
		.row {
			grid-template-columns: minmax(200px, 2fr) 140px 120px 100px 48px;
			padding: $space-3 $space-4;
		}
	}
</style>
