import { writable } from 'svelte/store';

export interface User {
	user_id: string;
	email: string;
	username: string;
}

interface AuthState {
	user: User | null;
	loading: boolean;
	initialized: boolean;
}

function createAuthStore() {
	const { subscribe, set, update } = writable<AuthState>({
		user: null,
		loading: true,
		initialized: false
	});

	return {
		subscribe,

		async init() {
			update((s) => ({ ...s, loading: true }));
			try {
				const res = await fetch('/api/auth/me');
				const data = await res.json();
				set({ user: data.user, loading: false, initialized: true });
			} catch {
				set({ user: null, loading: false, initialized: true });
			}
		},

		async login(login: string, password: string) {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ login, password })
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error?.message || 'Login failed');
			}

			const data = await res.json();
			update((s) => ({ ...s, user: data.user }));
			return data.user;
		},

		async register(email: string, username: string, password: string) {
			const res = await fetch('/api/auth/register', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, username, password })
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error?.message || 'Registration failed');
			}

			const data = await res.json();
			update((s) => ({ ...s, user: data.user }));
			return data.user;
		},

		async logout() {
			await fetch('/api/auth/logout', { method: 'POST' });
			update((s) => ({ ...s, user: null }));
		},

		async refresh() {
			try {
				const res = await fetch('/api/auth/refresh', { method: 'POST' });
				if (res.ok) {
					const data = await res.json();
					update((s) => ({ ...s, user: data.user }));
					return true;
				}
			} catch {
				// Ignore
			}
			update((s) => ({ ...s, user: null }));
			return false;
		}
	};
}

export const auth = createAuthStore();
