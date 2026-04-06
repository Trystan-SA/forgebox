<script lang="ts">
	import { setupAccount } from '$lib/api/client';
	import { auth, isAuthenticated } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let setupPassword = $state('');
	let error = $state<string | null>(null);
	let loading = $state(false);
	let step = $state<'credentials' | 'account'>('credentials');

	$effect(() => {
		if ($isAuthenticated) {
			goto('/dashboard');
		}
	});

	function handleNext(e: Event) {
		e.preventDefault();
		if (!setupPassword.trim()) return;
		step = 'account';
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!name.trim() || !email.trim() || !password.trim()) return;

		loading = true;
		error = null;

		try {
			await setupAccount({
				name: name.trim(),
				email: email.trim(),
				password: password.trim(),
				setup_password: setupPassword.trim()
			});
			await auth.login(email.trim(), password.trim());
		} catch (err) {
			error = err instanceof Error ? err.message : 'Setup failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="setup">
	<div class="welcome">
		<div class="welcome-inner">
			<div class="welcome-logo">
				<svg width="44" height="44" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
					<polyline points="3.27 6.96 12 12.01 20.73 6.96" />
					<line x1="12" y1="22.08" x2="12" y2="12" />
				</svg>
			</div>
			<h1 class="welcome-title">ForgeBox</h1>
			<p class="welcome-tagline">Secure AI automation for every team</p>

			<ul class="features">
				<li>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg>
					Isolated VM execution
				</li>
				<li>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /></svg>
					Team-based permissions
				</li>
				<li>
					<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" /></svg>
					Multi-provider LLM support
				</li>
			</ul>

			<div class="steps">
				<span class="step" class:active={step === 'credentials'}>1. Verify</span>
				<span class="step-divider"></span>
				<span class="step" class:active={step === 'account'}>2. Account</span>
			</div>
		</div>
	</div>

	<div class="form-panel">
		<div class="form-wrap">
			{#if step === 'credentials'}
				<span class="badge">First-time setup</span>
				<h2>Verify server access</h2>
				<p class="subtitle">
					Enter the <code>FORGEBOX_FIRST_PASSWORD</code> value from your server environment to begin.
				</p>

				{#if error}
					<div class="error-msg">{error}</div>
				{/if}

				<form onsubmit={handleNext}>
					<label class="field">
						<span>Setup password</span>
						<input type="password" bind:value={setupPassword} placeholder="Paste your setup password" required />
					</label>

					<button type="submit" class="btn-primary submit-btn">
						Continue
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6" /></svg>
					</button>
				</form>
			{:else}
				<button class="back-btn" onclick={() => (step = 'credentials')}>
					<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6" /></svg>
					Back
				</button>
				<h2>Create admin account</h2>
				<p class="subtitle">This will be the first administrator of your ForgeBox instance.</p>

				{#if error}
					<div class="error-msg">{error}</div>
				{/if}

				<form onsubmit={handleSubmit}>
					<label class="field">
						<span>Full name</span>
						<input type="text" bind:value={name} placeholder="Jane Smith" disabled={loading} required />
					</label>

					<label class="field">
						<span>Email address</span>
						<input type="email" bind:value={email} placeholder="admin@yourcompany.com" disabled={loading} required />
					</label>

					<label class="field">
						<span>Password</span>
						<input type="password" bind:value={password} placeholder="Choose a strong password" disabled={loading} required minlength="8" />
					</label>

					<button type="submit" class="btn-primary submit-btn" disabled={loading}>
						{#if loading}
							<span class="spinner"></span>
							Creating account...
						{:else}
							Create admin account
						{/if}
					</button>
				</form>
			{/if}
		</div>
	</div>
</div>

<style lang="scss">
	.setup {
		display: flex;
		min-height: 100vh;
	}

	@media (max-width: $bp-md) {
		.setup {
			flex-direction: column;
		}
	}

	.welcome {
		flex: 0 0 420px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: linear-gradient(160deg, $primary-900 0%, $primary-700 50%, $primary-600 100%);
		color: $neutral-0;
		padding: $space-10;
		position: relative;
		overflow: hidden;
	}

	.welcome::before {
		content: '';
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(rgba(255, 255, 255, 0.04) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255, 255, 255, 0.04) 1px, transparent 1px);
		background-size: 32px 32px;
		pointer-events: none;
	}

	@media (max-width: $bp-md) {
		.welcome {
			flex: none;
			padding: $space-8 $space-6;
		}
	}

	.welcome-inner {
		position: relative;
		z-index: 1;
		max-width: 320px;
	}

	.welcome-logo {
		display: inline-flex;
		padding: $space-3;
		background: rgba(255, 255, 255, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.15);
		border-radius: $radius-xl;
		margin-bottom: $space-6;
	}

	.welcome-title {
		font-size: $text-3xl;
		font-weight: $font-bold;
		letter-spacing: -0.02em;
		margin-bottom: $space-2;
		color: $neutral-0;
	}

	.welcome-tagline {
		font-size: $text-base;
		color: rgba(255, 255, 255, 0.65);
		line-height: $leading-relaxed;
		margin-bottom: $space-8;
	}

	.features {
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: $space-3;
		margin-bottom: $space-10;
		padding: 0;
	}

	.features li {
		display: flex;
		align-items: center;
		gap: $space-3;
		font-size: $text-sm;
		color: rgba(255, 255, 255, 0.75);
	}

	.features li svg {
		flex-shrink: 0;
		opacity: 0.5;
	}

	.steps {
		display: flex;
		align-items: center;
		gap: $space-3;
		padding-top: $space-6;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
	}

	.step {
		font-size: $text-xs;
		font-weight: $font-medium;
		color: rgba(255, 255, 255, 0.35);
		transition: color $transition-base;
	}

	.step.active {
		color: rgba(255, 255, 255, 0.95);
	}

	.step-divider {
		width: 32px;
		height: 1px;
		background: rgba(255, 255, 255, 0.15);
	}

	.form-panel {
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: $space-10;
		background: $neutral-50;
	}

	@media (max-width: $bp-md) {
		.form-panel {
			padding: $space-6;
		}
	}

	.form-wrap {
		width: 100%;
		max-width: 400px;
	}

	.badge {
		@include badge;
		background: $primary-50;
		color: $primary-700;
		border: 1px solid $primary-100;
		margin-bottom: $space-4;
	}

	.form-wrap h2 {
		font-size: $text-2xl;
		font-weight: $font-bold;
		color: $neutral-900;
		margin-bottom: $space-2;
	}

	.subtitle {
		font-size: $text-sm;
		color: $neutral-500;
		line-height: $leading-relaxed;
		margin-bottom: $space-6;
	}

	.subtitle code {
		font-family: $font-mono;
		font-size: $text-xs;
		background: $neutral-100;
		color: $primary-700;
		padding: 2px $space-1;
		border-radius: $radius-sm;
	}

	.error-msg {
		padding: $space-3;
		margin-bottom: $space-4;
		font-size: $text-sm;
		color: $error-700;
		background: $error-50;
		border: 1px solid $error-100;
		border-radius: $radius-lg;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: $space-4;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: $space-1;
	}

	.field span {
		font-size: $text-sm;
		font-weight: $font-medium;
		color: $neutral-700;
	}

	.field input {
		@include input-base;
	}

	.submit-btn {
		width: 100%;
		margin-top: $space-2;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: $space-2;
	}

	.back-btn {
		display: inline-flex;
		align-items: center;
		gap: $space-1;
		font-size: $text-sm;
		color: $neutral-500;
		background: none;
		border: none;
		cursor: pointer;
		padding: 0;
		margin-bottom: $space-4;
		transition: color $transition-fast;
	}

	.back-btn:hover {
		color: $neutral-700;
	}

	.spinner {
		display: inline-block;
		width: 16px;
		height: 16px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: $neutral-0;
		border-radius: $radius-full;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}
</style>
