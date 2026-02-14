<script lang="ts">
	import { Button, Input } from '$lib/components/ui';
	import { Eye, EyeOff } from 'lucide-svelte';

	let login = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let loading = $state(false);
	let error = $state('');

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;

		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ login, password })
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error?.message || 'Login failed');
			}

			// Redirect to home on success
			window.location.href = '/';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Something went wrong';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign In - Amaral</title>
</svelte:head>

<div class="w-full max-w-md">
	<div class="rounded-xl border border-border bg-surface/80 p-8 backdrop-blur-md">
		<!-- Logo -->
		<a href="/" class="mb-8 block text-center">
			<span class="font-heading text-3xl font-bold text-accent">AMARAL</span>
		</a>

		<h1 class="mb-2 text-center text-2xl font-semibold text-text">Welcome back</h1>
		<p class="mb-8 text-center text-sm text-text-secondary">
			Sign in to continue watching
		</p>

		{#if error}
			<div class="mb-6 rounded-lg border border-error/50 bg-error/10 p-3 text-sm text-error">
				{error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-5">
			<Input
				type="text"
				label="Email or Username"
				placeholder="Enter your email or username"
				bind:value={login}
				required
				autocomplete="username"
			/>

			<div class="relative">
				<Input
					type={showPassword ? 'text' : 'password'}
					label="Password"
					placeholder="Enter your password"
					bind:value={password}
					required
					autocomplete="current-password"
				/>
				<button
					type="button"
					onclick={() => (showPassword = !showPassword)}
					class="absolute right-3 top-[38px] text-text-muted hover:text-text transition-colors"
				>
					{#if showPassword}
						<EyeOff class="h-4 w-4" />
					{:else}
						<Eye class="h-4 w-4" />
					{/if}
				</button>
			</div>

			<div class="flex items-center justify-between">
				<label class="flex items-center gap-2 text-sm text-text-secondary">
					<input type="checkbox" class="rounded border-border bg-surface" />
					Remember me
				</label>
				<a href="/forgot-password" class="text-sm text-accent hover:underline">
					Forgot password?
				</a>
			</div>

			<Button type="submit" class="w-full" size="lg" {loading}>
				{#snippet children()}
					{loading ? 'Signing in...' : 'Sign In'}
				{/snippet}
			</Button>
		</form>

		<div class="mt-6 flex items-center gap-3">
			<div class="h-px flex-1 bg-border"></div>
			<span class="text-xs text-text-muted">or</span>
			<div class="h-px flex-1 bg-border"></div>
		</div>

		<p class="mt-6 text-center text-sm text-text-secondary">
			Don't have an account?
			<a href="/register" class="font-medium text-accent hover:underline">Create one</a>
		</p>
	</div>
</div>
