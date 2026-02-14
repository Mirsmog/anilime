import type { PageServerLoad } from './$types';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

export const load: PageServerLoad = async ({ params, locals, fetch }) => {
	const { episodeId } = params;
	const accessToken = locals.accessToken;

	let sources = null;
	let episodeInfo = null;

	// Fetch streaming sources
	if (accessToken) {
		try {
			const res = await fetch(`${BFF_URL}/v1/watch/${episodeId}?category=sub`, {
				headers: { Authorization: `Bearer ${accessToken}` }
			});
			if (res.ok) {
				sources = await res.json();
			}
		} catch (err) {
			console.error('Fetch sources error:', err);
		}
	}

	// TODO: Fetch episode info from catalog when endpoint is available
	// For now, we'll handle this client-side or pass minimal data

	return {
		episodeId,
		sources,
		episodeInfo
	};
};
