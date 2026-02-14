import { json, type RequestHandler } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const GET: RequestHandler = async ({ cookies }) => {
	const accessToken = cookies.get('access_token');

	if (!accessToken) {
		return json({ user: null }, { status: 200 });
	}

	try {
		const res = await fetch(`${BFF_URL}/v1/me`, {
			headers: { Authorization: `Bearer ${accessToken}` }
		});

		if (!res.ok) {
			// Token might be expired, try refresh
			return json({ user: null }, { status: 200 });
		}

		const data = await res.json();
		return json({ user: data });
	} catch (err) {
		console.error('Me error:', err);
		return json({ user: null }, { status: 200 });
	}
};
