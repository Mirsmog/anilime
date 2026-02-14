<script lang="ts">
	import { cn } from '$lib/utils';
	import { Button } from '$lib/components/ui';
	import {
		Play, Pause, Volume2, VolumeX, Maximize, Minimize,
		SkipForward, SkipBack, Settings, ChevronLeft, ChevronRight
	} from 'lucide-svelte';
	import { onMount } from 'svelte';

	let { data } = $props();

	// Use signed URL from API (proxied through our HLS proxy to avoid CORS)
	const signedPlaybackUrl = data.sources?.signed_playback_url || null;
	const intro = data.sources?.intro || { start: 0, end: 0 };
	const outro = data.sources?.outro || { start: 0, end: 0 };
	const tracks = data.sources?.tracks || [];

	// Episode info (would come from catalog API)
	const episodeInfo = {
		id: data.episodeId,
		animeId: '1',
		animeTitle: 'Loading...',
		number: 1,
		title: 'Loading...',
		nextEpisode: null as { id: string; number: number; title: string } | null,
		prevEpisode: null as { id: string; number: number; title: string } | null
	};

	let videoEl: HTMLVideoElement;
	let containerEl: HTMLDivElement;

	let playing = $state(false);
	let currentTime = $state(0);
	let duration = $state(0);
	let volume = $state(1);
	let muted = $state(false);
	let fullscreen = $state(false);
	let showControls = $state(true);
	let showSkipIntro = $state(false);
	let showSkipOutro = $state(false);
	let controlsTimeout: ReturnType<typeof setTimeout>;
	let progressSaveTimeout: ReturnType<typeof setTimeout>;

	$effect(() => {
		const inIntro = intro.end > 0 && currentTime >= intro.start && currentTime < intro.end;
		const inOutro = outro.end > 0 && currentTime >= outro.start && currentTime < outro.end;
		showSkipIntro = inIntro;
		showSkipOutro = inOutro;
	});

	// Save progress periodically
	$effect(() => {
		if (currentTime > 0 && duration > 0) {
			clearTimeout(progressSaveTimeout);
			progressSaveTimeout = setTimeout(() => {
				saveProgress();
			}, 10000); // Save every 10 seconds
		}
	});

	async function saveProgress() {
		try {
			await fetch('/api/activity/progress', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					episode_id: data.episodeId,
					position_seconds: Math.floor(currentTime),
					duration_seconds: Math.floor(duration),
					client_ts_ms: Date.now()
				})
			});
		} catch (err) {
			console.error('Save progress error:', err);
		}
	}

	function togglePlay() {
		if (videoEl.paused) {
			videoEl.play();
		} else {
			videoEl.pause();
		}
	}

	function toggleMute() {
		muted = !muted;
		videoEl.muted = muted;
	}

	function toggleFullscreen() {
		if (document.fullscreenElement) {
			document.exitFullscreen();
		} else {
			containerEl.requestFullscreen();
		}
	}

	function seek(time: number) {
		videoEl.currentTime = Math.max(0, Math.min(time, duration));
	}

	function skipIntro() {
		seek(intro.end);
	}

	function skipOutro() {
		if (episodeInfo.nextEpisode) {
			saveProgress();
			window.location.href = `/watch/${episodeInfo.nextEpisode.id}`;
		}
	}

	function formatTime(seconds: number): string {
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}

	function handleMouseMove() {
		showControls = true;
		clearTimeout(controlsTimeout);
		controlsTimeout = setTimeout(() => {
			if (playing) showControls = false;
		}, 3000);
	}

	function handleProgressClick(e: MouseEvent) {
		const target = e.currentTarget as HTMLDivElement;
		const rect = target.getBoundingClientRect();
		const percent = (e.clientX - rect.left) / rect.width;
		seek(percent * duration);
	}

	onMount(() => {
		import('hls.js').then(({ default: Hls }) => {
			if (Hls.isSupported() && signedPlaybackUrl) {
				const hls = new Hls();
				hls.loadSource(signedPlaybackUrl);
				hls.attachMedia(videoEl);
			}
		});

		const handleFullscreen = () => {
			fullscreen = !!document.fullscreenElement;
		};
		document.addEventListener('fullscreenchange', handleFullscreen);

		// Save progress on page leave
		const handleBeforeUnload = () => {
			saveProgress();
		};
		window.addEventListener('beforeunload', handleBeforeUnload);

		return () => {
			clearTimeout(controlsTimeout);
			clearTimeout(progressSaveTimeout);
			document.removeEventListener('fullscreenchange', handleFullscreen);
			window.removeEventListener('beforeunload', handleBeforeUnload);
			saveProgress();
		};
	});
</script>

<svelte:head>
	<title>E{episodeInfo.number}: {episodeInfo.title} - {episodeInfo.animeTitle}</title>
</svelte:head>

<div
	bind:this={containerEl}
	class="relative flex h-screen w-full flex-col bg-black"
	onmousemove={handleMouseMove}
	role="application"
>
	<div class="relative flex-1">
		<video
			bind:this={videoEl}
			class="h-full w-full"
			onclick={togglePlay}
			onplay={() => (playing = true)}
			onpause={() => (playing = false)}
			ontimeupdate={() => (currentTime = videoEl.currentTime)}
			ondurationchange={() => (duration = videoEl.duration)}
			onvolumechange={() => (volume = videoEl.volume)}
		></video>

		<div
			class={cn(
				'absolute inset-0 flex flex-col justify-between bg-gradient-to-t from-black/80 via-transparent to-black/40 transition-opacity duration-300',
				showControls ? 'opacity-100' : 'opacity-0 pointer-events-none'
			)}
		>
			<div class="flex items-center justify-between p-4">
				<a href="/anime/{episodeInfo.animeId}" class="flex items-center gap-2 text-white hover:text-accent transition-colors">
					<ChevronLeft class="h-6 w-6" />
					<span class="text-sm font-medium">Back</span>
				</a>
				<div class="text-center">
					<p class="text-sm text-white/80">{episodeInfo.animeTitle}</p>
					<p class="text-xs text-white/60">Episode {episodeInfo.number}: {episodeInfo.title}</p>
				</div>
				<div class="w-20"></div>
			</div>

			<div class="absolute inset-0 flex items-center justify-center pointer-events-none">
				{#if !playing}
					<button
						onclick={togglePlay}
						class="pointer-events-auto flex h-20 w-20 items-center justify-center rounded-full bg-accent/90 text-white shadow-xl transition-transform hover:scale-110"
					>
						<Play class="h-10 w-10 fill-current ml-1" />
					</button>
				{/if}
			</div>

			{#if showSkipIntro}
				<button
					onclick={skipIntro}
					class="absolute bottom-24 right-4 rounded-lg border border-white/30 bg-black/60 px-6 py-3 text-sm font-medium text-white backdrop-blur-sm transition-colors hover:bg-white/20"
				>
					Skip Intro
				</button>
			{/if}
			{#if showSkipOutro}
				<button
					onclick={skipOutro}
					class="absolute bottom-24 right-4 rounded-lg border border-white/30 bg-black/60 px-6 py-3 text-sm font-medium text-white backdrop-blur-sm transition-colors hover:bg-white/20"
				>
					Next Episode
				</button>
			{/if}

			<div class="space-y-2 p-4">
				<div
					class="group relative h-1 cursor-pointer rounded-full bg-white/30"
					onclick={handleProgressClick}
					role="slider"
					aria-valuenow={currentTime}
					aria-valuemin={0}
					aria-valuemax={duration}
					tabindex="0"
				>
					<div class="absolute inset-y-0 left-0 rounded-full bg-white/30" style="width: 50%"></div>
					<div
						class="absolute inset-y-0 left-0 rounded-full bg-accent"
						style="width: {(currentTime / duration) * 100}%"
					></div>
					<div
						class="absolute top-1/2 h-3 w-3 -translate-y-1/2 rounded-full bg-accent opacity-0 transition-opacity group-hover:opacity-100"
						style="left: {(currentTime / duration) * 100}%"
					></div>
				</div>

				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<button onclick={togglePlay} class="p-2 text-white hover:text-accent transition-colors">
							{#if playing}
								<Pause class="h-6 w-6" />
							{:else}
								<Play class="h-6 w-6" />
							{/if}
						</button>

						{#if episodeInfo.prevEpisode}
							<a href="/watch/{episodeInfo.prevEpisode.id}" class="p-2 text-white hover:text-accent transition-colors">
								<SkipBack class="h-5 w-5" />
							</a>
						{/if}

						{#if episodeInfo.nextEpisode}
							<a href="/watch/{episodeInfo.nextEpisode.id}" class="p-2 text-white hover:text-accent transition-colors">
								<SkipForward class="h-5 w-5" />
							</a>
						{/if}

						<div class="flex items-center gap-2">
							<button onclick={toggleMute} class="p-2 text-white hover:text-accent transition-colors">
								{#if muted || volume === 0}
									<VolumeX class="h-5 w-5" />
								{:else}
									<Volume2 class="h-5 w-5" />
								{/if}
							</button>
							<input
								type="range"
								min="0"
								max="1"
								step="0.1"
								bind:value={volume}
								oninput={() => {
									videoEl.volume = volume;
									if (volume > 0) muted = false;
								}}
								class="h-1 w-20 cursor-pointer appearance-none rounded-full bg-white/30 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white"
							/>
						</div>

						<span class="ml-2 text-sm text-white/80">
							{formatTime(currentTime)} / {formatTime(duration)}
						</span>
					</div>

					<div class="flex items-center gap-2">
						<button class="p-2 text-white hover:text-accent transition-colors">
							<Settings class="h-5 w-5" />
						</button>
						<button onclick={toggleFullscreen} class="p-2 text-white hover:text-accent transition-colors">
							{#if fullscreen}
								<Minimize class="h-5 w-5" />
							{:else}
								<Maximize class="h-5 w-5" />
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>

	{#if !fullscreen}
		<div class="border-t border-border bg-surface p-4">
			<div class="mx-auto flex max-w-[1920px] items-center justify-between">
				<div>
					<h1 class="text-lg font-semibold text-text">{episodeInfo.animeTitle}</h1>
					<p class="text-sm text-text-secondary">Episode {episodeInfo.number}: {episodeInfo.title}</p>
				</div>
				<div class="flex gap-2">
					{#if episodeInfo.prevEpisode}
						<a href="/watch/{episodeInfo.prevEpisode.id}">
							<Button variant="secondary">
								{#snippet children()}
									<ChevronLeft class="h-4 w-4" />
									Previous
								{/snippet}
							</Button>
						</a>
					{/if}
					{#if episodeInfo.nextEpisode}
						<a href="/watch/{episodeInfo.nextEpisode.id}">
							<Button>
								{#snippet children()}
									Next
									<ChevronRight class="h-4 w-4" />
								{/snippet}
							</Button>
						</a>
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>
