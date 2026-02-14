<script lang="ts">
	import { cn } from '$lib/utils';
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';
	import { Button } from '$lib/components/ui';
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		href?: string;
		children: Snippet;
		class?: string;
	}

	let { title, href, children, class: className }: Props = $props();

	let scrollContainer: HTMLDivElement;

	function scroll(direction: 'left' | 'right') {
		if (!scrollContainer) return;
		const scrollAmount = scrollContainer.clientWidth * 0.8;
		scrollContainer.scrollBy({
			left: direction === 'left' ? -scrollAmount : scrollAmount,
			behavior: 'smooth'
		});
	}
</script>

<section class={cn('relative', className)}>
	<!-- Header -->
	<div class="mb-4 flex items-center justify-between px-4 md:px-8">
		<h2 class="text-lg font-semibold text-text md:text-xl">{title}</h2>
		<div class="flex items-center gap-2">
			{#if href}
				<a href={href} class="text-sm text-text-secondary hover:text-accent transition-colors">
					See All
				</a>
			{/if}
			<div class="hidden gap-1 md:flex">
				<Button variant="ghost" size="sm" onclick={() => scroll('left')}>
					{#snippet children()}
						<ChevronLeft class="h-5 w-5" />
					{/snippet}
				</Button>
				<Button variant="ghost" size="sm" onclick={() => scroll('right')}>
					{#snippet children()}
						<ChevronRight class="h-5 w-5" />
					{/snippet}
				</Button>
			</div>
		</div>
	</div>

	<!-- Scroll container -->
	<div
		bind:this={scrollContainer}
		class="scrollbar-hide flex gap-3 overflow-x-auto px-4 pb-4 md:gap-4 md:px-8"
	>
		{@render children()}
	</div>
</section>

<style>
	.scrollbar-hide {
		-ms-overflow-style: none;
		scrollbar-width: none;
	}
	.scrollbar-hide::-webkit-scrollbar {
		display: none;
	}
</style>
