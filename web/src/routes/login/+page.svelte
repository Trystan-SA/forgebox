<script lang="ts">
	import { auth, isAuthenticated, isAdmin } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	let email = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let loading = $state(false);

	$effect(() => {
		if ($isAuthenticated) {
			goto($isAdmin ? '/dashboard' : '/home');
		}
	});

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!email.trim() || !password.trim()) return;

		loading = true;
		error = null;

		try {
			await auth.login(email.trim(), password.trim());
		} catch (err) {
			error = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<div class="login">
	<div class="login__card">
		<div class="login__header">
			<svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
			</svg>
			<h1>ForgeBox</h1>
			<p>Sign in to your account</p>
		</div>

		{#if error}
			<div class="login__error">{error}</div>
		{/if}

		<form class="login__form" onsubmit={handleSubmit}>
			<label class="login__field">
				<span>Email</span>
				<input
					type="email"
					bind:value={email}
					placeholder="you@example.com"
					disabled={loading}
					required
				/>
			</label>

			<label class="login__field">
				<span>Password</span>
				<input
					type="password"
					bind:value={password}
					placeholder="Enter password"
					disabled={loading}
					required
				/>
			</label>

			<button type="submit" class="btn-primary login__submit" disabled={loading}>
				{loading ? 'Signing in...' : 'Sign in'}
			</button>
		</form>
	</div>
</div>

<style lang="scss">
	.login {
		@include flex-center;
		min-height: 100vh;
		background: $neutral-50;

		&__card {
			@include card;
			width: 100%;
			max-width: 400px;
			padding: $space-8;
		}

		&__header {
			text-align: center;
			margin-bottom: $space-8;
			color: $primary-600;

			h1 {
				margin-top: $space-3;
				font-size: $text-2xl;
				font-weight: $font-bold;
				color: $neutral-900;
			}

			p {
				margin-top: $space-1;
				font-size: $text-sm;
				color: $neutral-500;
			}
		}

		&__error {
			padding: $space-3;
			margin-bottom: $space-4;
			font-size: $text-sm;
			color: $error-700;
			background: $error-50;
			border: 1px solid $error-100;
			border-radius: $radius-lg;
		}

		&__form {
			display: flex;
			flex-direction: column;
			gap: $space-4;
		}

		&__field {
			display: flex;
			flex-direction: column;
			gap: $space-1;

			span {
				font-size: $text-sm;
				font-weight: $font-medium;
				color: $neutral-700;
			}

			input {
				@include input-base;
			}
		}

		&__submit {
			width: 100%;
			margin-top: $space-2;
		}
	}
</style>
