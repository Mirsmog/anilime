<script lang="ts">
	import { HeroCarousel, AnimeCarousel, AnimeCard } from '$lib/components/anime';
	import { Play } from 'lucide-svelte';

	let { data } = $props();

	// Fallback data if API returns empty
	const defaultFeatured = [
		{
			id: '1',
			title: 'Solo Leveling',
			description: 'In a world where hunters with various magical abilities battle deadly monsters, Sung Jinwoo finds a hidden dungeon and gains the power to level up infinitely.',
			backdrop: 'https://image.tmdb.org/t/p/original/wNGAE0DiDvJPKvMQ7FQxJwjVdZm.jpg',
			genres: ['Action', 'Fantasy', 'Adventure'],
			score: 8.7,
			year: 2024
		}
	];

	const featured = data.featured?.length ? data.featured : defaultFeatured;
	const trending = data.trending || [];
	const topRated = data.topRated || [];
	const newEpisodes = data.newEpisodes || [];
	const continueWatching = data.continueWatching || [];
</script>

<svelte:head>
	<title>Amaral - Watch Anime Online</title>
	<meta name="description" content="Watch the latest anime episodes and explore a vast catalog of anime series and movies." />
</svelte:head>

<div class="flex flex-col">
	<HeroCarousel items={featured} />

	<div class="-mt-20 relative z-10 space-y-8 pb-16">
		{#if continueWatching.length > 0}
			<AnimeCarousel title="Continue Watching">
				{#snippet children()}
					{#each continueWatching as item}
						<a
							href="/watch/{item.episodeId}"
							class="group relative w-64 flex-shrink-0 overflow-hidden rounded-lg bg-surface md:w-80"
						>
							<div class="aspect-video w-full bg-surface-hover">
								{#if item.poster}
									<img src={item.poster} alt={item.title} class="h-full w-full object-cover" />
								{:else}
									<div class="flex h-full items-center justify-center text-text-muted">
										<Play class="h-12 w-12" />
									</div>
								{/if}
							</div>
							<!-- Progress bar -->
							<div class="absolute bottom-0 left-0 right-0 h-1 bg-surface-hover">
								<div class="h-full bg-accent" style="width: {item.progressPercent}%"></div>
							</div>
							<div class="p-3">
								<p class="truncate text-sm font-medium text-text">{item.title}</p>
								<p class="text-xs text-text-muted">Episode {item.episodeNumber}</p>
							</div>
							<!-- Play overlay -->
							<div class="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
								<div class="flex h-12 w-12 items-center justify-center rounded-full bg-accent">
									<Play class="h-6 w-6 fill-white text-white" />
								</div>
							</div>
						</a>
					{/each}
				{/snippet}
			</AnimeCarousel>
		{/if}

		{#if trending.length > 0}
			<AnimeCarousel title="Trending Now" href="/search?sort=popularity">
				{#snippet children()}
					{#each trending as anime}
						<AnimeCard {...anime} class="w-36 flex-shrink-0 md:w-44" />
					{/each}
				{/snippet}
			</AnimeCarousel>
		{/if}

		{#if newEpisodes.length > 0}
			<AnimeCarousel title="New Episodes" href="/search?status=airing">
				{#snippet children()}
					{#each newEpisodes as anime}
						<AnimeCard {...anime} class="w-36 flex-shrink-0 md:w-44" />
					{/each}
				{/snippet}
			</AnimeCarousel>
		{/if}

		{#if topRated.length > 0}
			<AnimeCarousel title="Top Rated" href="/search?sort=score">
				{#snippet children()}
					{#each topRated as anime}
						<AnimeCard {...anime} class="w-36 flex-shrink-0 md:w-44" />
					{/each}
				{/snippet}
			</AnimeCarousel>
		{/if}

		{#if !trending.length && !newEpisodes.length && !topRated.length}
			<div class="px-4 md:px-8">
				<div class="rounded-xl border border-border bg-surface p-8 text-center">
					<p class="text-lg text-text-secondary">No anime available yet.</p>
					<p class="mt-2 text-sm text-text-muted">Run the backfill to populate the catalog.</p>
				</div>
			</div>
		{/if}
	</div>
</div>
