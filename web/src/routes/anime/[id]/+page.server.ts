import type { PageServerLoad } from './$types';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

interface AnimeResponse {
	id: string;
	title: string;
	title_english?: string;
	title_japanese?: string;
	image?: string;
	description?: string;
	genres?: string[];
	score: number;
	status?: string;
	type?: string;
	total_episodes: number;
}

interface EpisodeResponse {
	id: string;
	anime_id: string;
	number: number;
	title: string;
	aired_at?: string;
}

interface RatingSummaryResponse {
	anime_id: string;
	average_rating: number;
	total_ratings: number;
	rating_distribution: Record<string, number>;
	user_rating?: number;
}

export const load: PageServerLoad = async ({ params, cookies, fetch: serverFetch }) => {
	const { id } = params;
	const accessToken = cookies.get('access_token');

	// Fetch anime details, episodes, and ratings in parallel
	const [animeRes, episodesRes, ratingsRes] = await Promise.all([
		serverFetch(`${BFF_URL}/v1/anime/${id}`),
		serverFetch(`${BFF_URL}/v1/anime/${id}/episodes`),
		serverFetch(`${BFF_URL}/v1/ratings/${id}${accessToken ? `?user_id=me` : ''}`, {
			headers: accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
		}).catch(() => null)
	]);

	if (!animeRes.ok) {
		return {
			anime: null,
			error: animeRes.status === 404 ? 'Anime not found' : 'Failed to load anime'
		};
	}

	const animeData: AnimeResponse = await animeRes.json();
	const episodesData: { episodes: EpisodeResponse[] } = episodesRes.ok
		? await episodesRes.json()
		: { episodes: [] };

	let ratings: RatingSummaryResponse | null = null;
	if (ratingsRes && ratingsRes.ok) {
		ratings = await ratingsRes.json();
	}

	// Transform to frontend format
	const anime = {
		id: animeData.id,
		title: animeData.title,
		title_english: animeData.title_english,
		title_japanese: animeData.title_japanese,
		poster: animeData.image || '/placeholder-anime.jpg',
		backdrop: animeData.image || '/placeholder-backdrop.jpg',
		synopsis: animeData.description || 'No description available.',
		score: animeData.score,
		scored_by: ratings?.total_ratings || 0,
		rank: 0,
		popularity: 0,
		year: new Date().getFullYear(),
		season: '',
		type: animeData.type || 'TV',
		status: animeData.status || 'Unknown',
		episodes_count: animeData.total_episodes,
		duration: '24 min per ep',
		rating: 'PG-13',
		genres: animeData.genres || [],
		studios: [],
		source: '',
		episodes: episodesData.episodes.map((ep) => ({
			id: ep.id,
			number: ep.number,
			title: ep.title || `Episode ${ep.number}`,
			duration: '24m',
			filler: false,
			watched: false
		})),
		related: [],
		userRating: ratings?.user_rating
	};

	return { anime };
};
