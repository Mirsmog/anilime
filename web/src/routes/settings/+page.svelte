<script lang="ts">
	import { Button, Input } from '$lib/components/ui';
	import { User, Mail, Lock, Bell, Monitor, Moon, Sun, ChevronLeft } from 'lucide-svelte';
	import { cn } from '$lib/utils';

	let { data } = $props();
	const { user } = data;

	let theme = $state<'dark' | 'light' | 'system'>('dark');
	let autoplay = $state(true);
	let autoNext = $state(true);
	let skipIntro = $state(true);
	let notifications = $state(true);
	let emailNotifications = $state(false);

	// Password change
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let isChangingPassword = $state(false);
	let passwordError = $state('');

	async function handlePasswordChange() {
		if (newPassword !== confirmPassword) {
			passwordError = 'Passwords do not match';
			return;
		}

		if (newPassword.length < 8) {
			passwordError = 'Password must be at least 8 characters';
			return;
		}

		isChangingPassword = true;
		passwordError = '';

		try {
			// TODO: Implement password change API
			await new Promise((resolve) => setTimeout(resolve, 1000));
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
		} catch (err) {
			passwordError = 'Failed to change password';
		} finally {
			isChangingPassword = false;
		}
	}
</script>

<svelte:head>
	<title>Settings - Amaral</title>
</svelte:head>

<div class="mx-auto max-w-2xl px-4 py-8 md:px-8 md:py-12">
	<div class="mb-8 flex items-center gap-4">
		<a href="/profile" class="rounded-lg p-2 transition-colors hover:bg-surface">
			<ChevronLeft class="h-5 w-5 text-text-secondary" />
		</a>
		<h1 class="text-2xl font-bold text-text">Settings</h1>
	</div>

	<!-- Account Section -->
	<section class="mb-8 rounded-xl border border-border bg-surface p-6">
		<div class="mb-4 flex items-center gap-2">
			<User class="h-5 w-5 text-accent" />
			<h2 class="text-lg font-semibold text-text">Account</h2>
		</div>

		<div class="space-y-4">
			<div>
				<label class="mb-1 block text-sm text-text-muted">Username</label>
				<Input value={user.username || 'Not set'} disabled />
			</div>
			<div>
				<label class="mb-1 block text-sm text-text-muted">Email</label>
				<Input value={user.email || 'Not set'} disabled />
			</div>
		</div>
	</section>

	<!-- Password Section -->
	<section class="mb-8 rounded-xl border border-border bg-surface p-6">
		<div class="mb-4 flex items-center gap-2">
			<Lock class="h-5 w-5 text-accent" />
			<h2 class="text-lg font-semibold text-text">Change Password</h2>
		</div>

		<form class="space-y-4" onsubmit={(e) => { e.preventDefault(); handlePasswordChange(); }}>
			<div>
				<label class="mb-1 block text-sm text-text-muted">Current Password</label>
				<Input type="password" bind:value={currentPassword} placeholder="••••••••" />
			</div>
			<div>
				<label class="mb-1 block text-sm text-text-muted">New Password</label>
				<Input type="password" bind:value={newPassword} placeholder="••••••••" />
			</div>
			<div>
				<label class="mb-1 block text-sm text-text-muted">Confirm New Password</label>
				<Input type="password" bind:value={confirmPassword} placeholder="••••••••" />
			</div>

			{#if passwordError}
				<p class="text-sm text-danger">{passwordError}</p>
			{/if}

			<Button type="submit" disabled={isChangingPassword || !currentPassword || !newPassword || !confirmPassword}>
				{#snippet children()}
					{isChangingPassword ? 'Changing...' : 'Change Password'}
				{/snippet}
			</Button>
		</form>
	</section>

	<!-- Playback Section -->
	<section class="mb-8 rounded-xl border border-border bg-surface p-6">
		<div class="mb-4 flex items-center gap-2">
			<Monitor class="h-5 w-5 text-accent" />
			<h2 class="text-lg font-semibold text-text">Playback</h2>
		</div>

		<div class="space-y-4">
			<label class="flex cursor-pointer items-center justify-between">
				<span class="text-text">Auto-play videos</span>
				<button
					type="button"
					onclick={() => (autoplay = !autoplay)}
					class={cn(
						'relative h-6 w-11 rounded-full transition-colors',
						autoplay ? 'bg-accent' : 'bg-surface-hover'
					)}
				>
					<span
						class={cn(
							'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
							autoplay ? 'translate-x-6' : 'translate-x-1'
						)}
					></span>
				</button>
			</label>

			<label class="flex cursor-pointer items-center justify-between">
				<span class="text-text">Auto-play next episode</span>
				<button
					type="button"
					onclick={() => (autoNext = !autoNext)}
					class={cn(
						'relative h-6 w-11 rounded-full transition-colors',
						autoNext ? 'bg-accent' : 'bg-surface-hover'
					)}
				>
					<span
						class={cn(
							'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
							autoNext ? 'translate-x-6' : 'translate-x-1'
						)}
					></span>
				</button>
			</label>

			<label class="flex cursor-pointer items-center justify-between">
				<span class="text-text">Skip intro automatically</span>
				<button
					type="button"
					onclick={() => (skipIntro = !skipIntro)}
					class={cn(
						'relative h-6 w-11 rounded-full transition-colors',
						skipIntro ? 'bg-accent' : 'bg-surface-hover'
					)}
				>
					<span
						class={cn(
							'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
							skipIntro ? 'translate-x-6' : 'translate-x-1'
						)}
					></span>
				</button>
			</label>
		</div>
	</section>

	<!-- Theme Section -->
	<section class="mb-8 rounded-xl border border-border bg-surface p-6">
		<div class="mb-4 flex items-center gap-2">
			<Moon class="h-5 w-5 text-accent" />
			<h2 class="text-lg font-semibold text-text">Appearance</h2>
		</div>

		<div class="flex gap-3">
			{#each [{ id: 'dark', icon: Moon, label: 'Dark' }, { id: 'light', icon: Sun, label: 'Light' }, { id: 'system', icon: Monitor, label: 'System' }] as option}
				<button
					onclick={() => (theme = option.id as 'dark' | 'light' | 'system')}
					class={cn(
						'flex flex-1 flex-col items-center gap-2 rounded-lg border p-4 transition-colors',
						theme === option.id
							? 'border-accent bg-accent/10 text-accent'
							: 'border-border text-text-secondary hover:border-accent/50'
					)}
				>
					<option.icon class="h-6 w-6" />
					<span class="text-sm font-medium">{option.label}</span>
				</button>
			{/each}
		</div>
	</section>

	<!-- Notifications Section -->
	<section class="rounded-xl border border-border bg-surface p-6">
		<div class="mb-4 flex items-center gap-2">
			<Bell class="h-5 w-5 text-accent" />
			<h2 class="text-lg font-semibold text-text">Notifications</h2>
		</div>

		<div class="space-y-4">
			<label class="flex cursor-pointer items-center justify-between">
				<div>
					<span class="block text-text">Push notifications</span>
					<span class="text-sm text-text-muted">Get notified about new episodes</span>
				</div>
				<button
					type="button"
					onclick={() => (notifications = !notifications)}
					class={cn(
						'relative h-6 w-11 rounded-full transition-colors',
						notifications ? 'bg-accent' : 'bg-surface-hover'
					)}
				>
					<span
						class={cn(
							'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
							notifications ? 'translate-x-6' : 'translate-x-1'
						)}
					></span>
				</button>
			</label>

			<label class="flex cursor-pointer items-center justify-between">
				<div>
					<span class="block text-text">Email notifications</span>
					<span class="text-sm text-text-muted">Receive updates via email</span>
				</div>
				<button
					type="button"
					onclick={() => (emailNotifications = !emailNotifications)}
					class={cn(
						'relative h-6 w-11 rounded-full transition-colors',
						emailNotifications ? 'bg-accent' : 'bg-surface-hover'
					)}
				>
					<span
						class={cn(
							'absolute top-1 h-4 w-4 rounded-full bg-white transition-transform',
							emailNotifications ? 'translate-x-6' : 'translate-x-1'
						)}
					></span>
				</button>
			</label>
		</div>
	</section>
</div>
