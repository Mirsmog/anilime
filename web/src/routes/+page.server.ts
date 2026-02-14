import type { PageServerLoad } from './$types';

const BFF_URL = process.env.BFF_URL || 'http://localhost:8080';

interface SearchHit {
	anime_id: string;
	title: string;
	title_english?: string;
	title_japanese?: string;
	image?: string;
	description?: string;
	genres?: string[];
	score?: number;
	status?: string;
	type?: string;
	total_episodes?: number;
}

export const load: PageServerLoad = async ({ locals, fetch }) => {
	const accessToken = locals.accessToken;

	// Fetch trending/top anime from search
	async function fetchAnimeList(params: Record<string, string>) {
		try {
			const searchParams = new URLSearchParams(params);
			const res = await fetch(`${BFF_URL}/v1/search?${searchParams.toString()}`);
			if (res.ok) {
				const data = await res.json();
				return (data.hits || []).map((hit: SearchHit) => ({
					id: hit.anime_id,
					title: hit.title_english || hit.title || 'Unknown',
					poster: hit.image || '/placeholder-anime.jpg',
					backdrop: hit.image || '/placeholder-backdrop.jpg',
					synopsis: hit.description || '',
					score: hit.score || 0,
					year: null,
					episodeCount: hit.total_episodes || null,
					genres: hit.genres || []
				}));
			}
		} catch (err) {
			console.error('Fetch anime list error:', err);
		}
		return [];
	}

	// Fetch continue watching if logged in
	async function fetchContinueWatching() {
		if (!accessToken) return [];
		try {
			const res = await fetch(`${BFF_URL}/v1/activity/continue?limit=10`, {
				headers: { Authorization: `Bearer ${accessToken}` }
			});
			if (res.ok) {
				const data = await res.json();
				return (data.items || []).map((item: Record<string, unknown>) => {
					const episode = item.episode as Record<string, unknown>;
					const progress = item.progress as Record<string, unknown>;
					return {
						id: episode?.anime_id,
						episodeId: episode?.episode_id,
						title: episode?.title || `Episode ${episode?.number}`,
						poster: (item.image as string) || '/placeholder-anime.jpg',
						episodeNumber: episode?.number,
						progress: progress?.position_seconds,
						duration: progress?.duration_seconds,
						progressPercent: progress?.duration_seconds 
							? Math.round(((progress?.position_seconds as number) / (progress?.duration_seconds as number)) * 100)
							: 0
					};
				});
			}
		} catch (err) {
			console.error('Fetch continue watching error:', err);
		}
		return [];
	}

	// Parallel fetch
	const [trending, topRated, newEpisodes, continueWatching] = await Promise.all([
		fetchAnimeList({ limit: '12', sort: 'popularity' }),
		fetchAnimeList({ limit: '12', sort: 'score' }),
		fetchAnimeList({ limit: '12', status: 'airing' }),
		fetchContinueWatching()
	]);

	// Pick featured from trending for hero
	const featured = trending.slice(0, 3).map((anime: Record<string, unknown>) => ({
		id: anime.id,
		title: anime.title,
		description: (anime.synopsis as string) || 'No description available.',
		backdrop: anime.backdrop || anime.poster,
		genres: Array.isArray(anime.genres) ? anime.genres.map((g: unknown) => typeof g === 'string' ? g : (g as Record<string, unknown>).name) : [],
		score: anime.score,
		year: anime.year
	}));

	return {
		featured,
		trending,
		topRated,
		newEpisodes,
		continueWatching
	};
};
