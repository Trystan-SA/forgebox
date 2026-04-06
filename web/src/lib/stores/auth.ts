import { writable, derived } from 'svelte/store';
import type { User, UserRole } from '$lib/api/types';
import { login as apiLogin } from '$lib/api/client';

interface AuthState {
	user: User | null;
	token: string | null;
}

function createAuthStore() {
	const initial: AuthState = {
		user: null,
		token: typeof window !== 'undefined' ? localStorage.getItem('forgebox_token') : null
	};

	// Try to restore user from localStorage
	if (typeof window !== 'undefined') {
		const stored = localStorage.getItem('forgebox_user');
		if (stored) {
			try {
				initial.user = JSON.parse(stored);
			} catch {
				// ignore
			}
		}
	}

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,
		async login(email: string, password: string) {
			const res = await apiLogin({ email, password });
			localStorage.setItem('forgebox_token', res.token);
			localStorage.setItem('forgebox_user', JSON.stringify(res.user));
			set({ user: res.user, token: res.token });
		},
		logout() {
			localStorage.removeItem('forgebox_token');
			localStorage.removeItem('forgebox_user');
			set({ user: null, token: null });
		},
		setUser(user: User, token: string) {
			localStorage.setItem('forgebox_token', token);
			localStorage.setItem('forgebox_user', JSON.stringify(user));
			set({ user, token });
		}
	};
}

export const auth = createAuthStore();
export const currentUser = derived(auth, ($auth) => $auth.user);
export const isAuthenticated = derived(auth, ($auth) => !!$auth.token && !!$auth.user);
export const isAdmin = derived(auth, ($auth) => $auth.user?.role === 'admin');
export const userRole = derived(auth, ($auth): UserRole | null => $auth.user?.role ?? null);
