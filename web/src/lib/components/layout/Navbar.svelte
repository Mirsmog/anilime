<script lang="ts">
	import { cn } from '$lib/utils';
	import { Search, Bell, User, Menu, LogOut } from 'lucide-svelte';
	import { Button } from '$lib/components/ui';

	interface UserInfo {
		user_id: string;
		email: string;
		username: string;
	}

	interface Props {
		user?: UserInfo | null;
		class?: string;
	}

	let { user = null, class: className }: Props = $props();

	let scrolled = $state(false);
	let searchOpen = $state(false);
	let searchQuery = $state('');
	let userMenuOpen = $state(false);

	$effect(() => {
		const handleScroll = () => {
			scrolled = window.scrollY > 10;
		};
		window.addEventListener('scroll', handleScroll);
		return () => window.removeEventListener('scroll', handleScroll);
	});

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		window.location.href = '/';
	}
</script>

<header
	class={cn(
		'fixed top-0 left-0 right-0 z-50 transition-all duration-300',
		scrolled ? 'bg-bg/95 backdrop-blur-md shadow-[inset_0_-1px_0_0_rgba(255,255,255,0.1)]' : 'bg-gradient-to-b from-bg/80 to-transparent',
		className
	)}
>
	<nav class="mx-auto flex h-16 max-w-[1920px] items-center justify-between px-4 md:px-8">
		<!-- Logo -->
		<a href="/" class="flex items-center gap-2 text-xl font-bold text-accent">
			<span class="font-heading">AMARAL</span>
		</a>

		<!-- Desktop Nav -->
		<div class="hidden items-center gap-6 md:flex">
			<a href="/" class="text-sm font-medium text-text hover:text-accent transition-colors">
				Home
			</a>
			<a href="/search" class="text-sm font-medium text-text-secondary hover:text-text transition-colors">
				Browse
			</a>
			<a href="/search?status=airing" class="text-sm font-medium text-text-secondary hover:text-text transition-colors">
				New
			</a>
		</div>

		<!-- Right side -->
		<div class="flex items-center gap-2">
			<!-- Search -->
			<div class="relative">
				{#if searchOpen}
					<div class="flex items-center gap-2">
						<input
							type="text"
							placeholder="Search anime..."
							bind:value={searchQuery}
							class="h-9 w-48 rounded-lg border border-border bg-surface px-3 text-sm text-text placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-accent md:w-64"
							onkeydown={(e) => {
								if (e.key === 'Enter' && searchQuery) {
									window.location.href = `/search?q=${encodeURIComponent(searchQuery)}`;
								}
								if (e.key === 'Escape') {
									searchOpen = false;
									searchQuery = '';
								}
							}}
						/>
						<Button variant="ghost" size="sm" onclick={() => { searchOpen = false; searchQuery = ''; }}>
							{#snippet children()}✕{/snippet}
						</Button>
					</div>
				{:else}
					<Button variant="ghost" size="sm" onclick={() => (searchOpen = true)}>
						{#snippet children()}
							<Search class="h-5 w-5" />
						{/snippet}
					</Button>
				{/if}
			</div>

			{#if user}
				<!-- Notifications -->
				<Button variant="ghost" size="sm" class="hidden md:flex">
					{#snippet children()}
						<Bell class="h-5 w-5" />
					{/snippet}
				</Button>

				<!-- User menu -->
				<div class="relative">
					<Button variant="ghost" size="sm" onclick={() => (userMenuOpen = !userMenuOpen)}>
						{#snippet children()}
							<div class="flex h-8 w-8 items-center justify-center rounded-full bg-accent text-sm font-medium text-white">
								{user.username.charAt(0).toUpperCase()}
							</div>
						{/snippet}
					</Button>

					{#if userMenuOpen}
						<div class="absolute right-0 top-full mt-2 w-48 rounded-lg border border-border bg-surface py-1 shadow-lg">
							<div class="border-b border-border px-4 py-2">
								<p class="text-sm font-medium text-text">{user.username}</p>
								<p class="text-xs text-text-muted">{user.email}</p>
							</div>
							<a href="/profile" class="block px-4 py-2 text-sm text-text-secondary hover:bg-surface-hover hover:text-text">
								Profile
							</a>
							<a href="/settings" class="block px-4 py-2 text-sm text-text-secondary hover:bg-surface-hover hover:text-text">
								Settings
							</a>
							<button
								onclick={handleLogout}
								class="flex w-full items-center gap-2 px-4 py-2 text-sm text-error hover:bg-surface-hover"
							>
								<LogOut class="h-4 w-4" />
								Sign out
							</button>
						</div>
					{/if}
				</div>
			{:else}
				<a href="/login">
					<Button variant="ghost" size="sm">
						{#snippet children()}Sign In{/snippet}
					</Button>
				</a>
				<a href="/register" class="hidden md:block">
					<Button size="sm">
						{#snippet children()}Sign Up{/snippet}
					</Button>
				</a>
			{/if}

			<!-- Mobile menu -->
			<Button variant="ghost" size="sm" class="md:hidden">
				{#snippet children()}
					<Menu class="h-5 w-5" />
				{/snippet}
			</Button>
		</div>
	</nav>
</header>
