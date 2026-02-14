<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	interface Props extends HTMLButtonAttributes {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		loading?: boolean;
		children: Snippet;
	}

	let {
		variant = 'primary',
		size = 'md',
		loading = false,
		class: className,
		disabled,
		children,
		...rest
	}: Props = $props();

	const baseStyles =
		'inline-flex items-center justify-center font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg disabled:pointer-events-none disabled:opacity-50';

	const variants = {
		primary: 'bg-accent text-white hover:bg-accent-hover active:scale-[0.98]',
		secondary:
			'bg-surface text-text border border-border hover:bg-surface-hover hover:border-border-hover',
		ghost: 'text-text-secondary hover:text-text hover:bg-surface-hover',
		danger: 'bg-error text-white hover:bg-error/90'
	};

	const sizes = {
		sm: 'h-8 px-3 text-sm rounded-md gap-1.5',
		md: 'h-10 px-4 text-sm rounded-lg gap-2',
		lg: 'h-12 px-6 text-base rounded-lg gap-2'
	};
</script>

<button
	class={cn(baseStyles, variants[variant], sizes[size], className)}
	disabled={disabled || loading}
	{...rest}
>
	{#if loading}
		<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
			></circle>
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
			></path>
		</svg>
	{/if}
	{@render children()}
</button>
