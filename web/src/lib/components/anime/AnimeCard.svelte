<script lang="ts">
	import { cn } from '$lib/utils';
	import { Play, Star } from 'lucide-svelte';

	interface Props {
		id: string;
		title: string;
		poster: string;
		score?: number;
		year?: number;
		episodeCount?: number;
		class?: string;
	}

	let { id, title, poster, score, year, episodeCount, class: className }: Props = $props();

	let hovered = $state(false);
</script>

<a
	href="/anime/{id}"
	class={cn(
		'group relative block overflow-hidden rounded-lg transition-transform duration-300',
		'hover:scale-105 hover:z-10',
		className
	)}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- Poster -->
	<div class="aspect-[2/3] w-full overflow-hidden rounded-lg bg-surface">
		<img
			src={poster}
			alt={title}
			class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-110"
			loading="lazy"
		/>
	</div>

	<!-- Overlay on hover -->
	<div
		class={cn(
			'absolute inset-0 flex flex-col justify-end rounded-lg bg-gradient-to-t from-bg via-bg/60 to-transparent p-3 opacity-0 transition-opacity duration-300',
			hovered && 'opacity-100'
		)}
	>
		<!-- Play button -->
		<div class="absolute inset-0 flex items-center justify-center">
			<div class="flex h-12 w-12 items-center justify-center rounded-full bg-accent text-white shadow-lg">
				<Play class="h-5 w-5 fill-current" />
			</div>
		</div>
	</div>

	<!-- Info below -->
	<div class="mt-2 space-y-1">
		<h3 class="line-clamp-2 text-sm font-medium text-text group-hover:text-accent transition-colors">
			{title}
		</h3>
		<div class="flex items-center gap-2 text-xs text-text-muted">
			{#if score}
				<span class="flex items-center gap-1">
					<Star class="h-3 w-3 fill-accent text-accent" />
					{score.toFixed(1)}
				</span>
			{/if}
			{#if year}
				<span>{year}</span>
			{/if}
			{#if episodeCount}
				<span>{episodeCount} eps</span>
			{/if}
		</div>
	</div>
</a>
