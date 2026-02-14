import { json, type RequestHandler } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

// POST /api/ratings/[animeId] - submit a rating
export const POST: RequestHandler = async ({ params, request, cookies }) => {
	const accessToken = cookies.get('access_token');
	const { animeId } = params;

	if (!accessToken) {
		return json({ error: { message: 'Unauthorized' } }, { status: 401 });
	}

	try {
		const { rating } = await request.json();

		if (typeof rating !== 'number' || rating < 1 || rating > 10) {
			return json({ error: { message: 'Rating must be between 1 and 10' } }, { status: 400 });
		}

		const res = await fetch(`${BFF_URL}/v1/ratings/${animeId}`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: `Bearer ${accessToken}`
			},
			body: JSON.stringify({ rating })
		});

		const data = await res.json();
		return json(data, { status: res.status });
	} catch (err) {
		console.error('Rating submit error:', err);
		return json({ error: { message: 'Internal server error' } }, { status: 500 });
	}
};

// DELETE /api/ratings/[animeId] - delete a rating
export const DELETE: RequestHandler = async ({ params, cookies }) => {
	const accessToken = cookies.get('access_token');
	const { animeId } = params;

	if (!accessToken) {
		return json({ error: { message: 'Unauthorized' } }, { status: 401 });
	}

	try {
		const res = await fetch(`${BFF_URL}/v1/ratings/${animeId}`, {
			method: 'DELETE',
			headers: {
				Authorization: `Bearer ${accessToken}`
			}
		});

		if (res.status === 204) {
			return new Response(null, { status: 204 });
		}

		const data = await res.json();
		return json(data, { status: res.status });
	} catch (err) {
		console.error('Rating delete error:', err);
		return json({ error: { message: 'Internal server error' } }, { status: 500 });
	}
};
