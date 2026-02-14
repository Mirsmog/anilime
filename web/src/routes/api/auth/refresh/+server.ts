import { json, type RequestHandler } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const POST: RequestHandler = async ({ cookies }) => {
	const refreshToken = cookies.get('refresh_token');

	if (!refreshToken) {
		return json({ error: { message: 'No refresh token' } }, { status: 401 });
	}

	try {
		const res = await fetch(`${BFF_URL}/v1/auth/refresh`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ refresh_token: refreshToken })
		});

		const data = await res.json();

		if (!res.ok) {
			// Clear invalid tokens
			cookies.delete('access_token', { path: '/' });
			cookies.delete('refresh_token', { path: '/' });
			return json(data, { status: res.status });
		}

		// Update tokens
		cookies.set('access_token', data.access_token, {
			path: '/',
			httpOnly: true,
			secure: process.env.NODE_ENV === 'production',
			sameSite: 'lax',
			maxAge: data.expires_in || 3600
		});

		cookies.set('refresh_token', data.refresh_token, {
			path: '/',
			httpOnly: true,
			secure: process.env.NODE_ENV === 'production',
			sameSite: 'lax',
			maxAge: 60 * 60 * 24 * 30
		});

		return json({ user: data.user });
	} catch (err) {
		console.error('Refresh error:', err);
		return json({ error: { message: 'Internal server error' } }, { status: 500 });
	}
};
