import { json, type RequestHandler } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const POST: RequestHandler = async ({ request, cookies }) => {
	try {
		const body = await request.json();

		const res = await fetch(`${BFF_URL}/v1/auth/register`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body)
		});

		const data = await res.json();

		if (!res.ok) {
			return json(data, { status: res.status });
		}

		// Set tokens in httpOnly cookies
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
			maxAge: 60 * 60 * 24 * 30 // 30 days
		});

		// Return user info without tokens
		return json({ user: data.user });
	} catch (err) {
		console.error('Register error:', err);
		return json({ error: { message: 'Internal server error' } }, { status: 500 });
	}
};
