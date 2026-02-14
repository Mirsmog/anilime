<script lang="ts">
	import { cn } from '$lib/utils';
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends HTMLInputAttributes {
		label?: string;
		error?: string;
		value?: string;
	}

	let { label, error, class: className, id, value = $bindable(''), ...rest }: Props = $props();

	const generatedId = crypto.randomUUID();
	const inputId = $derived(id || generatedId);
</script>

<div class="flex flex-col gap-1.5">
	{#if label}
		<label for={inputId} class="text-sm font-medium text-text-secondary">
			{label}
		</label>
	{/if}
	<input
		id={inputId}
		bind:value
		class={cn(
			'h-10 w-full rounded-lg border bg-surface px-3 text-sm text-text placeholder:text-text-muted',
			'transition-colors duration-200',
			'focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-bg',
			'disabled:cursor-not-allowed disabled:opacity-50',
			error ? 'border-error' : 'border-border hover:border-border-hover',
			className
		)}
		{...rest}
	/>
	{#if error}
		<p class="text-sm text-error">{error}</p>
	{/if}
</div>
