<script lang="ts">
	import { cn } from '$lib/utils';
	import { Play, Plus, Star, Info } from 'lucide-svelte';
	import { Button } from '$lib/components/ui';

	interface FeaturedAnime {
		id: string;
		title: string;
		description: string;
		backdrop: string;
		genres: string[];
		score: number;
		year: number;
	}

	interface Props {
		items: FeaturedAnime[];
		class?: string;
	}

	let { items, class: className }: Props = $props();

	let currentIndex = $state(0);
	let intervalId: ReturnType<typeof setInterval>;

	$effect(() => {
		if (items.length <= 1) return;
		intervalId = setInterval(() => {
			currentIndex = (currentIndex + 1) % items.length;
		}, 8000);
		return () => clearInterval(intervalId);
	});

	function goTo(index: number) {
		currentIndex = index;
		clearInterval(intervalId);
		intervalId = setInterval(() => {
			currentIndex = (currentIndex + 1) % items.length;
		}, 8000);
	}

	let current = $derived(items[currentIndex]);
</script>

<section class={cn('relative h-[70vh] min-h-[500px] w-full overflow-hidden', className)}>
	<!-- Background images -->
	{#each items as item, i}
		<div
			class={cn(
				'absolute inset-0 transition-opacity duration-1000',
				i === currentIndex ? 'opacity-100' : 'opacity-0'
			)}
		>
			<img
				src={item.backdrop}
				alt={item.title}
				class="h-full w-full object-cover"
			/>
			<!-- Gradient overlays -->
			<div class="absolute inset-0 bg-gradient-to-r from-bg via-bg/70 to-transparent"></div>
			<div class="absolute inset-0 bg-gradient-to-t from-bg via-transparent to-bg/30"></div>
		</div>
	{/each}

	<!-- Content -->
	<div class="absolute inset-0 flex items-center">
		<div class="mx-auto w-full max-w-[1920px] px-4 md:px-8">
			<div class="max-w-2xl space-y-4">
				{#key currentIndex}
					<h1 class="font-heading text-3xl font-bold text-text md:text-5xl lg:text-6xl animate-in fade-in slide-in-from-bottom-4 duration-500">
						{current.title}
					</h1>

					<div class="flex flex-wrap items-center gap-3 text-sm text-text-secondary animate-in fade-in slide-in-from-bottom-4 duration-500 delay-100">
						<span class="flex items-center gap-1">
							<Star class="h-4 w-4 fill-accent text-accent" />
							{current.score.toFixed(1)}
						</span>
						<span>{current.year}</span>
						<span>•</span>
						{#each current.genres.slice(0, 3) as genre}
							<span class="rounded-full bg-surface-hover px-2 py-0.5 text-xs">{genre}</span>
						{/each}
					</div>

					<p class="line-clamp-3 text-sm text-text-secondary md:text-base animate-in fade-in slide-in-from-bottom-4 duration-500 delay-200">
						{current.description}
					</p>

					<div class="flex flex-wrap gap-3 pt-2 animate-in fade-in slide-in-from-bottom-4 duration-500 delay-300">
						<Button size="lg" variant="primary">
							{#snippet children()}
								<Play class="h-5 w-5 fill-current" />
								Watch Now
							{/snippet}
						</Button>
						<Button size="lg" variant="secondary">
							{#snippet children()}
								<Info class="h-5 w-5" />
								More Info
							{/snippet}
						</Button>
					</div>
				{/key}
			</div>
		</div>
	</div>

	<!-- Dots indicator -->
	{#if items.length > 1}
		<div class="absolute bottom-16 left-1/2 flex -translate-x-1/2 gap-2">
			{#each items as _, i}
				<button
					onclick={() => goTo(i)}
					class={cn(
						'h-1.5 rounded-full transition-all duration-300',
						i === currentIndex ? 'w-8 bg-accent' : 'w-1.5 bg-text-muted hover:bg-text-secondary'
					)}
					aria-label="Go to slide {i + 1}"
				></button>
			{/each}
		</div>
	{/if}
</section>

<style>
	@keyframes fade-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}
	@keyframes slide-in-from-bottom-4 {
		from { transform: translateY(1rem); }
		to { transform: translateY(0); }
	}
	.animate-in {
		animation: fade-in 0.5s ease-out, slide-in-from-bottom-4 0.5s ease-out;
	}
	.delay-100 { animation-delay: 100ms; animation-fill-mode: both; }
	.delay-200 { animation-delay: 200ms; animation-fill-mode: both; }
	.delay-300 { animation-delay: 300ms; animation-fill-mode: both; }
</style>
