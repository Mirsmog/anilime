<script lang="ts">
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui';
	import { AnimeCard, AnimeCarousel } from '$lib/components/anime';
	import { Play, Plus, Star, Calendar, Clock, Tv, Check, AlertCircle } from 'lucide-svelte';

	let { data } = $props();
	const { anime, error } = data;

	let selectedCategory = $state<'sub' | 'dub' | 'raw'>('sub');
	let expandSynopsis = $state(false);
	let userRating = $state(anime?.userRating ?? 0);
	let isRating = $state(false);

	async function submitRating(rating: number) {
		if (!anime || isRating) return;
		isRating = true;

		try {
			const res = await fetch(`/api/ratings/${anime.id}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ rating })
			});

			if (res.ok) {
				userRating = rating;
			}
		} catch (err) {
			console.error('Failed to submit rating:', err);
		} finally {
			isRating = false;
		}
	}

	const firstEpisode = $derived(anime?.episodes?.[0]);
</script>

<svelte:head>
	<title>{anime?.title || 'Anime Not Found'} - Amaral</title>
	{#if anime}
		<meta name="description" content={anime.synopsis.slice(0, 160)} />
		<meta property="og:title" content={anime.title} />
		<meta property="og:description" content={anime.synopsis.slice(0, 160)} />
		<meta property="og:image" content={anime.poster} />
	{/if}
</svelte:head>

{#if error || !anime}
	<div class="flex min-h-[60vh] flex-col items-center justify-center gap-4 px-4">
		<AlertCircle class="h-16 w-16 text-text-muted" />
		<h1 class="text-2xl font-bold text-text">{error || 'Anime not found'}</h1>
		<p class="text-text-secondary">The anime you're looking for doesn't exist or has been removed.</p>
		<a href="/">
			<Button>Go Home</Button>
		</a>
	</div>
{:else}
	<!-- Hero backdrop -->
	<div class="relative h-[50vh] min-h-[400px] w-full">
		<img
			src={anime.backdrop}
			alt={anime.title}
			class="h-full w-full object-cover"
		/>
		<div class="absolute inset-0 bg-gradient-to-t from-bg via-bg/60 to-transparent"></div>
		<div class="absolute inset-0 bg-gradient-to-r from-bg/80 via-transparent to-transparent"></div>
	</div>

	<!-- Content -->
	<div class="relative -mt-48 pb-16">
		<div class="mx-auto max-w-[1920px] px-4 md:px-8">
			<div class="flex flex-col gap-8 lg:flex-row">
				<!-- Poster & Actions -->
				<div class="flex flex-shrink-0 flex-col items-center gap-4 lg:w-64">
					<img
						src={anime.poster}
						alt={anime.title}
						class="h-auto w-48 rounded-xl shadow-xl lg:w-full"
					/>
					<div class="flex w-full flex-col gap-2">
						{#if firstEpisode}
							<a href="/watch/{firstEpisode.id}">
								<Button size="lg" class="w-full">
									{#snippet children()}
										<Play class="h-5 w-5 fill-current" />
										Play Episode 1
									{/snippet}
								</Button>
							</a>
						{/if}
						<Button size="lg" variant="secondary" class="w-full">
							{#snippet children()}
								<Plus class="h-5 w-5" />
								Add to List
							{/snippet}
						</Button>
					</div>

					<!-- User Rating -->
					<div class="w-full rounded-lg border border-border bg-surface p-4">
						<p class="mb-2 text-sm text-text-muted">Your Rating</p>
						<div class="flex justify-center gap-1">
							{#each [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] as star}
								<button
									onclick={() => submitRating(star)}
									disabled={isRating}
									class={cn(
										'p-1 transition-colors',
										star <= userRating ? 'text-accent' : 'text-text-muted hover:text-accent/70'
									)}
								>
									<Star class={cn('h-4 w-4', star <= userRating && 'fill-current')} />
								</button>
							{/each}
						</div>
						{#if userRating > 0}
							<p class="mt-2 text-center text-sm text-accent">{userRating}/10</p>
						{/if}
					</div>
				</div>

				<!-- Info -->
				<div class="flex-1 space-y-6">
					<div>
						<h1 class="font-heading text-3xl font-bold text-text md:text-4xl">
							{anime.title}
						</h1>
						{#if anime.title_japanese}
							<p class="mt-1 text-lg text-text-secondary">{anime.title_japanese}</p>
						{/if}
					</div>

					<!-- Stats row -->
					<div class="flex flex-wrap items-center gap-4 text-sm">
						<div class="flex items-center gap-1.5 rounded-lg bg-accent/20 px-3 py-1.5 text-accent">
							<Star class="h-4 w-4 fill-current" />
							<span class="font-semibold">{anime.score.toFixed(1)}</span>
							{#if anime.scored_by > 0}
								<span class="text-text-muted">({anime.scored_by >= 1000 ? `${(anime.scored_by / 1000).toFixed(0)}K` : anime.scored_by})</span>
							{/if}
						</div>
						{#if anime.year}
							<div class="flex items-center gap-1.5 text-text-secondary">
								<Calendar class="h-4 w-4" />
								<span>{anime.year}{anime.season ? ` • ${anime.season}` : ''}</span>
							</div>
						{/if}
						<div class="flex items-center gap-1.5 text-text-secondary">
							<Tv class="h-4 w-4" />
							<span>{anime.type}{anime.episodes_count ? ` • ${anime.episodes_count} episodes` : ''}</span>
						</div>
						{#if anime.duration}
							<div class="flex items-center gap-1.5 text-text-secondary">
								<Clock class="h-4 w-4" />
								<span>{anime.duration}</span>
							</div>
						{/if}
					</div>

					<!-- Genres -->
					{#if anime.genres.length > 0}
						<div class="flex flex-wrap gap-2">
							{#each anime.genres as genre}
								<a
									href="/search?genres={genre}"
									class="rounded-full border border-border bg-surface px-3 py-1 text-sm text-text-secondary transition-colors hover:border-accent hover:text-accent"
								>
									{genre}
								</a>
							{/each}
						</div>
					{/if}

					<!-- Synopsis -->
					<div class="space-y-2">
						<h2 class="text-lg font-semibold text-text">Synopsis</h2>
						<p
							class={cn(
								'text-sm leading-relaxed text-text-secondary',
								!expandSynopsis && 'line-clamp-4'
							)}
						>
							{anime.synopsis}
						</p>
						{#if anime.synopsis.length > 300}
							<button
								onclick={() => (expandSynopsis = !expandSynopsis)}
								class="text-sm text-accent hover:underline"
							>
								{expandSynopsis ? 'Show less' : 'Show more'}
							</button>
						{/if}
					</div>

					<!-- Info grid -->
					<div class="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
						<div>
							<span class="text-text-muted">Status</span>
							<p class="font-medium text-text">{anime.status}</p>
						</div>
						{#if anime.studios.length > 0}
							<div>
								<span class="text-text-muted">Studios</span>
								<p class="font-medium text-text">{anime.studios.join(', ')}</p>
							</div>
						{/if}
						{#if anime.source}
							<div>
								<span class="text-text-muted">Source</span>
								<p class="font-medium text-text">{anime.source}</p>
							</div>
						{/if}
						{#if anime.rating}
							<div>
								<span class="text-text-muted">Rating</span>
								<p class="font-medium text-text">{anime.rating}</p>
							</div>
						{/if}
					</div>
				</div>
			</div>

			<!-- Episodes section -->
			{#if anime.episodes.length > 0}
				<section class="mt-12">
					<div class="mb-4 flex flex-wrap items-center justify-between gap-4">
						<h2 class="text-xl font-semibold text-text">Episodes ({anime.episodes.length})</h2>
						<div class="flex gap-2">
							{#each ['sub', 'dub', 'raw'] as cat}
								<button
									onclick={() => (selectedCategory = cat as 'sub' | 'dub' | 'raw')}
									class={cn(
										'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
										selectedCategory === cat
											? 'bg-accent text-white'
											: 'bg-surface text-text-secondary hover:bg-surface-hover'
									)}
								>
									{cat.toUpperCase()}
								</button>
							{/each}
						</div>
					</div>

					<div class="rounded-xl border border-border bg-surface">
						{#each anime.episodes as episode, i}
							<a
								href="/watch/{episode.id}?category={selectedCategory}"
								class={cn(
									'flex items-center gap-4 border-b border-border p-4 transition-colors hover:bg-surface-hover',
									i === anime.episodes.length - 1 && 'border-b-0'
								)}
							>
								<div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-surface-hover text-sm font-medium text-text-secondary">
									{episode.number}
								</div>
								<div class="flex-1 min-w-0">
									<p class="truncate font-medium text-text">{episode.title}</p>
									<p class="text-sm text-text-muted">{episode.duration}</p>
								</div>
								<div class="flex items-center gap-2">
									{#if episode.filler}
										<span class="rounded bg-warning/20 px-2 py-0.5 text-xs font-medium text-warning">
											FILLER
										</span>
									{/if}
									{#if episode.watched}
										<Check class="h-5 w-5 text-success" />
									{/if}
									<Play class="h-5 w-5 text-text-muted" />
								</div>
							</a>
						{/each}
					</div>
				</section>
			{:else}
				<section class="mt-12">
					<div class="rounded-xl border border-border bg-surface p-8 text-center">
						<p class="text-text-muted">No episodes available yet.</p>
					</div>
				</section>
			{/if}

			<!-- Related anime -->
			{#if anime.related.length > 0}
				<section class="mt-12">
					<AnimeCarousel title="You May Also Like">
						{#snippet children()}
							{#each anime.related as related}
								<AnimeCard {...related} class="w-36 flex-shrink-0 md:w-44" />
							{/each}
						{/snippet}
					</AnimeCarousel>
				</section>
			{/if}
		</div>
	</div>
{/if}
