<script lang="ts">
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui';
	import { AnimeCard } from '$lib/components/anime';
	import { Search, Filter, X } from 'lucide-svelte';
	import { goto } from '$app/navigation';

	let { data } = $props();

	let searchQuery = $state(data.query);
	let showFilters = $state(false);

	const genreOptions = [
		'Action', 'Adventure', 'Comedy', 'Drama', 'Fantasy', 'Horror',
		'Mystery', 'Romance', 'Sci-Fi', 'Slice of Life', 'Sports', 'Supernatural'
	];
	const statusOptions = ['Airing', 'Finished', 'Upcoming'];
	const typeOptions = ['TV', 'Movie', 'OVA', 'ONA', 'Special'];
	const sortOptions = [
		{ value: 'score', label: 'Top Rated' },
		{ value: 'popularity', label: 'Most Popular' },
		{ value: 'latest', label: 'Latest' },
		{ value: 'title', label: 'A-Z' }
	];

	let selectedGenres = $state<string[]>(data.filters.genres ? data.filters.genres.split(',') : []);
	let selectedStatus = $state(data.filters.status);
	let selectedType = $state(data.filters.type);
	let selectedSort = $state(data.filters.sort || 'score');

	function applyFilters() {
		const params = new URLSearchParams();
		if (searchQuery) params.set('q', searchQuery);
		if (selectedGenres.length) params.set('genres', selectedGenres.join(','));
		if (selectedStatus) params.set('status', selectedStatus);
		if (selectedType) params.set('type', selectedType);
		if (selectedSort) params.set('sort', selectedSort);
		goto(`/search?${params.toString()}`);
	}

	function clearFilters() {
		selectedGenres = [];
		selectedStatus = '';
		selectedType = '';
		selectedSort = 'score';
		searchQuery = '';
		goto('/search');
	}

	function toggleGenre(genre: string) {
		if (selectedGenres.includes(genre)) {
			selectedGenres = selectedGenres.filter((g) => g !== genre);
		} else {
			selectedGenres = [...selectedGenres, genre];
		}
	}

	const hasActiveFilters = $derived(
		selectedGenres.length > 0 || selectedStatus || selectedType || searchQuery
	);
</script>

<svelte:head>
	<title>{data.query ? `Search: ${data.query}` : 'Browse Anime'} - Amaral</title>
</svelte:head>

<div class="min-h-screen pt-20 pb-16">
	<div class="mx-auto max-w-[1920px] px-4 md:px-8">
		<div class="mb-8">
			<h1 class="mb-4 text-2xl font-bold text-text md:text-3xl">
				{data.query ? `Results for "${data.query}"` : 'Browse Anime'}
			</h1>

			<div class="flex gap-3">
				<div class="relative flex-1">
					<Search class="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-text-muted" />
					<input
						type="text"
						placeholder="Search anime..."
						bind:value={searchQuery}
						onkeydown={(e) => e.key === 'Enter' && applyFilters()}
						class="h-12 w-full rounded-xl border border-border bg-surface pl-11 pr-4 text-text placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-accent"
					/>
				</div>
				<Button onclick={() => (showFilters = !showFilters)} variant="secondary" size="lg">
					{#snippet children()}
						<Filter class="h-5 w-5" />
						Filters
						{#if hasActiveFilters}
							<span class="ml-1 flex h-5 w-5 items-center justify-center rounded-full bg-accent text-xs text-white">
								{selectedGenres.length + (selectedStatus ? 1 : 0) + (selectedType ? 1 : 0)}
							</span>
						{/if}
					{/snippet}
				</Button>
			</div>
		</div>

		{#if showFilters}
			<div class="mb-8 rounded-xl border border-border bg-surface p-6">
				<div class="mb-6 flex items-center justify-between">
					<h2 class="text-lg font-semibold text-text">Filters</h2>
					{#if hasActiveFilters}
						<button onclick={clearFilters} class="flex items-center gap-1 text-sm text-accent hover:underline">
							<X class="h-4 w-4" />
							Clear all
						</button>
					{/if}
				</div>

				<div class="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
					<div>
						<label class="mb-2 block text-sm font-medium text-text-secondary">Genres</label>
						<div class="flex flex-wrap gap-2">
							{#each genreOptions as genre}
								<button
									onclick={() => toggleGenre(genre)}
									class={cn(
										'rounded-full px-3 py-1 text-sm transition-colors',
										selectedGenres.includes(genre)
											? 'bg-accent text-white'
											: 'bg-surface-hover text-text-secondary hover:text-text'
									)}
								>
									{genre}
								</button>
							{/each}
						</div>
					</div>

					<div>
						<label class="mb-2 block text-sm font-medium text-text-secondary">Status</label>
						<select
							bind:value={selectedStatus}
							class="h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm text-text focus:outline-none focus:ring-2 focus:ring-accent"
						>
							<option value="">All</option>
							{#each statusOptions as status}
								<option value={status.toLowerCase()}>{status}</option>
							{/each}
						</select>
					</div>

					<div>
						<label class="mb-2 block text-sm font-medium text-text-secondary">Type</label>
						<select
							bind:value={selectedType}
							class="h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm text-text focus:outline-none focus:ring-2 focus:ring-accent"
						>
							<option value="">All</option>
							{#each typeOptions as type}
								<option value={type.toLowerCase()}>{type}</option>
							{/each}
						</select>
					</div>

					<div>
						<label class="mb-2 block text-sm font-medium text-text-secondary">Sort by</label>
						<select
							bind:value={selectedSort}
							class="h-10 w-full rounded-lg border border-border bg-surface px-3 text-sm text-text focus:outline-none focus:ring-2 focus:ring-accent"
						>
							{#each sortOptions as opt}
								<option value={opt.value}>{opt.label}</option>
							{/each}
						</select>
					</div>
				</div>

				<div class="mt-6 flex justify-end">
					<Button onclick={applyFilters}>
						{#snippet children()}Apply Filters{/snippet}
					</Button>
				</div>
			</div>
		{/if}

		<p class="mb-6 text-sm text-text-secondary">Found {data.total} results</p>

		{#if data.results.length > 0}
			<div class="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 2xl:grid-cols-8">
				{#each data.results as anime}
					<AnimeCard {...anime} />
				{/each}
			</div>
		{:else}
			<div class="flex flex-col items-center justify-center py-20 text-center">
				<Search class="mb-4 h-16 w-16 text-text-muted" />
				<h2 class="text-xl font-semibold text-text">No results found</h2>
				<p class="mt-2 text-text-secondary">Try adjusting your search or filters</p>
			</div>
		{/if}
	</div>
</div>
