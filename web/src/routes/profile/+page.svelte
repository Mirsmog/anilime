<script lang="ts">
	import { Button } from '$lib/components/ui';
	import { User, Mail, Clock, Settings, LogOut, Play, Trash2 } from 'lucide-svelte';

	let { data } = $props();
	const { user, continueWatching } = data;

	async function handleLogout() {
		await fetch('/api/auth/logout', { method: 'POST' });
		window.location.href = '/';
	}

	function formatProgress(progress: number, total: number): string {
		if (!total) return `${Math.floor(progress / 60)}m`;
		const percent = Math.round((progress / total) * 100);
		return `${percent}%`;
	}
</script>

<svelte:head>
	<title>Profile - Amaral</title>
</svelte:head>

<div class="mx-auto max-w-4xl px-4 py-8 md:px-8 md:py-12">
	<!-- Profile Header -->
	<div class="mb-8 flex flex-col items-center gap-6 rounded-xl border border-border bg-surface p-6 md:flex-row md:p-8">
		<div class="flex h-24 w-24 items-center justify-center rounded-full bg-accent/20 text-accent">
			<User class="h-12 w-12" />
		</div>

		<div class="flex-1 text-center md:text-left">
			<h1 class="text-2xl font-bold text-text">{user.username || 'User'}</h1>
			{#if user.email}
				<div class="mt-1 flex items-center justify-center gap-2 text-text-secondary md:justify-start">
					<Mail class="h-4 w-4" />
					<span>{user.email}</span>
				</div>
			{/if}
			<p class="mt-2 text-sm text-text-muted">User ID: {user.user_id}</p>
		</div>

		<div class="flex gap-2">
			<a href="/settings">
				<Button variant="secondary">
					{#snippet children()}
						<Settings class="h-4 w-4" />
						Settings
					{/snippet}
				</Button>
			</a>
			<Button variant="secondary" onclick={handleLogout}>
				{#snippet children()}
					<LogOut class="h-4 w-4" />
					Logout
				{/snippet}
			</Button>
		</div>
	</div>

	<!-- Continue Watching -->
	<section>
		<div class="mb-4 flex items-center gap-2">
			<Clock class="h-5 w-5 text-accent" />
			<h2 class="text-xl font-semibold text-text">Continue Watching</h2>
		</div>

		{#if continueWatching.length > 0}
			<div class="space-y-3">
				{#each continueWatching as item}
					<div class="flex items-center gap-4 rounded-lg border border-border bg-surface p-4 transition-colors hover:border-accent/50">
						<div class="relative h-20 w-32 flex-shrink-0 overflow-hidden rounded-lg">
							{#if item.thumbnail}
								<img src={item.thumbnail} alt={item.anime_title} class="h-full w-full object-cover" />
							{:else}
								<div class="flex h-full w-full items-center justify-center bg-surface-hover">
									<Play class="h-8 w-8 text-text-muted" />
								</div>
							{/if}
							<div class="absolute inset-0 bg-black/40 opacity-0 transition-opacity hover:opacity-100 flex items-center justify-center">
								<Play class="h-10 w-10 text-white" />
							</div>
						</div>

						<div class="flex-1 min-w-0">
							<a href="/anime/{item.anime_id}" class="font-medium text-text hover:text-accent transition-colors">
								{item.anime_title || 'Unknown Anime'}
							</a>
							<p class="text-sm text-text-secondary">
								Episode {item.episode_number || '?'}
								{#if item.episode_title}
									: {item.episode_title}
								{/if}
							</p>
							<div class="mt-2 flex items-center gap-2">
								<div class="h-1 flex-1 overflow-hidden rounded-full bg-surface-hover">
									<div
										class="h-full bg-accent"
										style="width: {item.duration_seconds ? (item.position_seconds / item.duration_seconds) * 100 : 0}%"
									></div>
								</div>
								<span class="text-xs text-text-muted">
									{formatProgress(item.position_seconds, item.duration_seconds)}
								</span>
							</div>
						</div>

						<a
							href="/watch/{item.episode_id}"
							class="flex-shrink-0"
						>
							<Button size="sm">
								{#snippet children()}
									<Play class="h-4 w-4" />
									Resume
								{/snippet}
							</Button>
						</a>
					</div>
				{/each}
			</div>
		{:else}
			<div class="rounded-xl border border-border bg-surface p-8 text-center">
				<Clock class="mx-auto mb-4 h-12 w-12 text-text-muted" />
				<p class="text-text-secondary">No watch history yet.</p>
				<p class="mt-1 text-sm text-text-muted">Start watching anime to track your progress!</p>
				<a href="/" class="mt-4 inline-block">
					<Button>Browse Anime</Button>
				</a>
			</div>
		{/if}
	</section>
</div>
