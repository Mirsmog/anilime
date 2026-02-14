import type { Handle } from '@sveltejs/kit';
import { redirect } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

const protectedRoutes = ['/watch', '/profile', '/settings'];
const authRoutes = ['/login', '/register'];

export const handle: Handle = async ({ event, resolve }) => {
	const accessToken = event.cookies.get('access_token');
	const refreshToken = event.cookies.get('refresh_token');

	// Try to get user info
	let user = null;
	if (accessToken) {
		try {
			const res = await fetch(`${BFF_URL}/v1/me`, {
				headers: { Authorization: `Bearer ${accessToken}` }
			});
			if (res.ok) {
				user = await res.json();
			} else if (res.status === 401 && refreshToken) {
				// Try refresh
				const refreshRes = await fetch(`${BFF_URL}/v1/auth/refresh`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ refresh_token: refreshToken })
				});

				if (refreshRes.ok) {
					const data = await refreshRes.json();
					event.cookies.set('access_token', data.access_token, {
						path: '/',
						httpOnly: true,
						secure: process.env.NODE_ENV === 'production',
						sameSite: 'lax',
						maxAge: data.expires_in || 3600
					});
					event.cookies.set('refresh_token', data.refresh_token, {
						path: '/',
						httpOnly: true,
						secure: process.env.NODE_ENV === 'production',
						sameSite: 'lax',
						maxAge: 60 * 60 * 24 * 30
					});
					user = data.user;
				} else {
					// Clear invalid tokens
					event.cookies.delete('access_token', { path: '/' });
					event.cookies.delete('refresh_token', { path: '/' });
				}
			}
		} catch (err) {
			console.error('Auth check error:', err);
		}
	}

	// Set user in locals for access in load functions
	event.locals.user = user;
	event.locals.accessToken = accessToken || null;

	const path = event.url.pathname;

	// Redirect authenticated users away from auth pages
	if (user && authRoutes.some((r) => path.startsWith(r))) {
		throw redirect(302, '/');
	}

	// Redirect unauthenticated users from protected routes
	if (!user && protectedRoutes.some((r) => path.startsWith(r))) {
		const returnUrl = encodeURIComponent(path);
		throw redirect(302, `/login?return=${returnUrl}`);
	}

	return resolve(event);
};
