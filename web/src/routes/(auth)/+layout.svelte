<script lang="ts">
	let { children } = $props();
</script>

<!-- Auth layout - no navbar, centered content with background -->
<div class="relative min-h-screen w-full overflow-hidden bg-bg">
	<!-- Background image with blur -->
	<div class="absolute inset-0">
		<img
			src="https://image.tmdb.org/t/p/original/wNGAE0DiDvJPKvMQ7FQxJwjVdZm.jpg"
			alt=""
			class="h-full w-full object-cover opacity-30 blur-sm"
		/>
		<div class="absolute inset-0 bg-gradient-to-br from-bg via-bg/90 to-bg"></div>
	</div>

	<!-- Content -->
	<div class="relative flex min-h-screen items-center justify-center p-4">
		{@render children()}
	</div>
</div>
