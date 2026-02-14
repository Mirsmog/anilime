<script lang="ts">
	import { Button, Input } from '$lib/components/ui';
	import { Eye, EyeOff, Check, X } from 'lucide-svelte';

	let email = $state('');
	let username = $state('');
	let password = $state('');
	let showPassword = $state(false);
	let loading = $state(false);
	let error = $state('');

	// Password validation
	let passwordChecks = $derived({
		length: password.length >= 8,
		uppercase: /[A-Z]/.test(password),
		lowercase: /[a-z]/.test(password),
		number: /[0-9]/.test(password)
	});

	let passwordValid = $derived(
		passwordChecks.length && passwordChecks.uppercase && 
		passwordChecks.lowercase && passwordChecks.number
	);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!passwordValid) {
			error = 'Please meet all password requirements';
			return;
		}

		error = '';
		loading = true;

		try {
			const res = await fetch('/api/auth/register', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, username, password })
			});

			if (!res.ok) {
				const data = await res.json();
				throw new Error(data.error?.message || 'Registration failed');
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
	<title>Create Account - Amaral</title>
</svelte:head>

<div class="w-full max-w-md">
	<div class="rounded-xl border border-border bg-surface/80 p-8 backdrop-blur-md">
		<!-- Logo -->
		<a href="/" class="mb-8 block text-center">
			<span class="font-heading text-3xl font-bold text-accent">AMARAL</span>
		</a>

		<h1 class="mb-2 text-center text-2xl font-semibold text-text">Create account</h1>
		<p class="mb-8 text-center text-sm text-text-secondary">
			Join to start watching anime
		</p>

		{#if error}
			<div class="mb-6 rounded-lg border border-error/50 bg-error/10 p-3 text-sm text-error">
				{error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-5">
			<Input
				type="email"
				label="Email"
				placeholder="you@example.com"
				bind:value={email}
				required
				autocomplete="email"
			/>

			<Input
				type="text"
				label="Username"
				placeholder="Choose a username"
				bind:value={username}
				required
				autocomplete="username"
			/>

			<div class="relative">
				<Input
					type={showPassword ? 'text' : 'password'}
					label="Password"
					placeholder="Create a password"
					bind:value={password}
					required
					autocomplete="new-password"
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

			<!-- Password requirements -->
			{#if password.length > 0}
				<div class="space-y-2 text-xs">
					<p class="text-text-muted">Password must have:</p>
					<div class="grid grid-cols-2 gap-2">
						{#each [
							{ check: passwordChecks.length, label: '8+ characters' },
							{ check: passwordChecks.uppercase, label: 'Uppercase letter' },
							{ check: passwordChecks.lowercase, label: 'Lowercase letter' },
							{ check: passwordChecks.number, label: 'Number' }
						] as { check, label }}
							<div class="flex items-center gap-1.5">
								{#if check}
									<Check class="h-3.5 w-3.5 text-success" />
								{:else}
									<X class="h-3.5 w-3.5 text-text-muted" />
								{/if}
								<span class={check ? 'text-success' : 'text-text-muted'}>{label}</span>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<label class="flex items-start gap-2 text-sm text-text-secondary">
				<input type="checkbox" class="mt-0.5 rounded border-border bg-surface" required />
				<span>
					I agree to the 
					<a href="/terms" class="text-accent hover:underline">Terms of Service</a>
					and
					<a href="/privacy" class="text-accent hover:underline">Privacy Policy</a>
				</span>
			</label>

			<Button type="submit" class="w-full" size="lg" {loading} disabled={!passwordValid}>
				{#snippet children()}
					{loading ? 'Creating account...' : 'Create Account'}
				{/snippet}
			</Button>
		</form>

		<div class="mt-6 flex items-center gap-3">
			<div class="h-px flex-1 bg-border"></div>
			<span class="text-xs text-text-muted">or</span>
			<div class="h-px flex-1 bg-border"></div>
		</div>

		<p class="mt-6 text-center text-sm text-text-secondary">
			Already have an account?
			<a href="/login" class="font-medium text-accent hover:underline">Sign in</a>
		</p>
	</div>
</div>
