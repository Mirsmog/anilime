import { json, type RequestHandler } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const POST: RequestHandler = async ({ request, cookies }) => {
	const accessToken = cookies.get('access_token');

	if (!accessToken) {
		return json({ error: { message: 'Unauthorized' } }, { status: 401 });
	}

	try {
		const body = await request.json();

		const res = await fetch(`${BFF_URL}/v1/activity/progress`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${accessToken}`
			},
			body: JSON.stringify(body)
		});

		const data = await res.json();
		return json(data, { status: res.status });
	} catch (err) {
		console.error('Progress save error:', err);
		return json({ error: { message: 'Internal server error' } }, { status: 500 });
	}
};
