import type { PageServerLoad } from './$types';
import { redirect } from '@sveltejs/kit';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const load: PageServerLoad = async ({ cookies, locals }) => {
	if (!locals.user) {
		throw redirect(302, '/login?return=/profile');
	}

	const accessToken = cookies.get('access_token');

	// Fetch continue watching for the user
	let continueWatching: any[] = [];

	try {
		const res = await fetch(`${BFF_URL}/v1/activity/continue?limit=20`, {
			headers: {
				Authorization: `Bearer ${accessToken}`
			}
		});

		if (res.ok) {
			const data = await res.json();
			continueWatching = data.items || [];
		}
	} catch (err) {
		console.error('Failed to fetch continue watching:', err);
	}

	return {
		user: locals.user,
		continueWatching
	};
};
