import type { PageServerLoad } from './$types';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

interface SearchHit {
	anime_id: string;
	title: string;
	title_english?: string;
	image?: string;
	score?: number;
	total_episodes?: number;
}

export const load: PageServerLoad = async ({ url, fetch }) => {
	const q = url.searchParams.get('q') || '';
	const genres = url.searchParams.get('genres') || '';
	const status = url.searchParams.get('status') || '';
	const type = url.searchParams.get('type') || '';
	const sort = url.searchParams.get('sort') || 'score';
	const page = parseInt(url.searchParams.get('page') || '1');
	const limit = 24;
	const offset = (page - 1) * limit;

	try {
		const params = new URLSearchParams();
		if (q) params.set('q', q);
		if (genres) params.set('genres', genres);
		if (status) params.set('status', status);
		if (type) params.set('type', type);
		params.set('limit', String(limit));
		params.set('offset', String(offset));

		const res = await fetch(`${BFF_URL}/v1/search?${params.toString()}`);

		if (res.ok) {
			const data = await res.json();
			// Transform search hits to match our AnimeCard props
			const results = (data.hits || []).map((hit: SearchHit) => ({
				id: hit.anime_id,
				title: hit.title_english || hit.title || 'Unknown',
				poster: hit.image || '/placeholder-anime.jpg',
				score: hit.score || 0,
				year: null,
				episodeCount: hit.total_episodes || null
			}));

			return {
				results,
				total: data.total || results.length,
				query: q,
				filters: { genres, status, type, sort },
				page
			};
		}
	} catch (err) {
		console.error('Search error:', err);
	}

	// Fallback to empty results
	return {
		results: [],
		total: 0,
		query: q,
		filters: { genres, status, type, sort },
		page
	};
};
