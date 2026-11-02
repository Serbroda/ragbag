<script lang="ts">
	import { Alert, Button, Input, Label } from 'flowbite-svelte';
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { BackupSummaryDto } from 'ragbag-frontend-sdk';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { apiConfig, completeSetup, errorMessage, logout, restoreBackup, takeLoginPassword } from '../api/client';
	import { navigate, p } from '../router';
	import { loadMe, resetSession } from '../stores/spaces.svelte';

	const MIN_PASSWORD_LENGTH = 8;

	// choose: start fresh or restore a backup (empty instances only); password: replace the initial one when
	// starting fresh. A restore needs no own password: it replaces this account with those of the backup.
	let step = $state<'loading' | 'password' | 'choose' | 'restored'>('loading');
	let username = $state('');
	let mustChangePassword = $state(false);
	let canChoose = $state(false);
	// Prefilled from the login right before; asked again after a reload
	const remembered = takeLoginPassword();
	let initialPassword = $state(remembered ?? '');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let busy = $state(false);

	let file = $state<File | null>(null);
	let summary = $state<BackupSummaryDto | null>(null);

	async function init() {
		try {
			resetSession();
			const me = await loadMe();
			username = me.username;
			mustChangePassword = !!me.mustChangePassword;
			const instance = me.instanceAdmin ? await new AdminApi(apiConfig()).getInstance() : null;
			canChoose = !!instance?.empty;
			if (canChoose) {
				step = 'choose';
			} else if (mustChangePassword) {
				step = 'password';
			} else {
				navigate('/', { replace: true });
			}
		} catch (err) {
			error = await errorMessage(err, 'Could not load your account.');
			step = 'password';
		}
	}

	init();

	async function savePassword(e: Event) {
		e.preventDefault();
		if (password !== confirmPassword) {
			error = 'Passwords do not match.';
			return;
		}
		busy = true;
		error = '';
		try {
			await completeSetup(initialPassword, password, email);
			resetSession();
			navigate('/', { replace: true });
		} catch (err) {
			error = await errorMessage(err, 'Failed to save the password.');
		} finally {
			busy = false;
		}
	}

	function startFresh() {
		error = '';
		if (mustChangePassword) {
			step = 'password';
		} else {
			navigate('/');
		}
	}

	async function check(selected: File | undefined) {
		file = selected ?? null;
		summary = null;
		error = '';
		if (!file) return;
		busy = true;
		try {
			summary = await restoreBackup(file, true);
		} catch (err) {
			error = await errorMessage(err, 'This file could not be read as a backup.');
			file = null;
		} finally {
			busy = false;
		}
	}

	async function restore() {
		if (!file) return;
		busy = true;
		error = '';
		try {
			summary = await restoreBackup(file, false);
			resetSession();
			step = 'restored';
		} catch (err) {
			error = await errorMessage(err, 'Restoring the backup failed; nothing was changed.');
		} finally {
			busy = false;
		}
	}

	async function signOut() {
		await logout().catch(() => {});
		resetSession();
		navigate('/login');
	}

	function count(table: string): number {
		return summary?.rows[table] ?? 0;
	}
</script>

<CenteredScreen subtitle={step === 'password' ? 'Choose your password' : step === 'restored' ? 'Backup restored' : 'Welcome to Ragbag'}>
	{#if step === 'loading'}
		<p class="text-center text-gray-400">Loading…</p>
	{:else if step === 'password'}
		<form class="flex flex-col gap-4" onsubmit={savePassword}>
			<p class="text-sm text-gray-300">
				You signed in as <strong class="text-white">{username}</strong> with the initial password. Choose your own
				one; the file on the server is deleted then.
			</p>
			<input type="text" autocomplete="username" value={username} hidden readonly />
			{#if !remembered}
				<div>
					<Label for="initial-password" class="mb-2">Initial password (from the file)</Label>
					<Input id="initial-password" type="password" autocomplete="current-password" bind:value={initialPassword} required />
				</div>
			{/if}
			<div>
				<Label for="email" class="mb-2">Your email address</Label>
				<Input id="email" type="email" autocomplete="email" bind:value={email} required />
			</div>
			<div>
				<Label for="new-password" class="mb-2">New password</Label>
				<Input
					id="new-password"
					type="password"
					autocomplete="new-password"
					minlength={MIN_PASSWORD_LENGTH}
					placeholder="At least {MIN_PASSWORD_LENGTH} characters"
					bind:value={password}
					required
				/>
			</div>
			<div>
				<Label for="confirm-password" class="mb-2">Repeat new password</Label>
				<Input id="confirm-password" type="password" autocomplete="new-password" bind:value={confirmPassword} required />
			</div>
			{#if error}
				<Alert color="red">{error}</Alert>
			{/if}
			<Button type="submit" disabled={busy || password.length < MIN_PASSWORD_LENGTH}>
				{busy ? 'Saving…' : 'Save and continue'}
			</Button>
			{#if canChoose}
				<button type="button" class="text-sm text-gray-400 hover:underline" onclick={() => ((step = 'choose'), (error = ''))}>
					Back: restore a backup instead
				</button>
			{:else}
				<button type="button" class="text-sm text-gray-400 hover:underline" onclick={signOut}>Log out</button>
			{/if}
		</form>
	{:else if step === 'choose'}
		<div class="flex flex-col gap-4">
			<p class="text-sm text-gray-300">This Ragbag is still empty. Start fresh or move an existing one here.</p>
			<Button color="alternative" onclick={startFresh}>Start fresh</Button>

			<div class="rounded-lg p-4 ring-1 ring-inset ring-gray-700">
				<p class="mb-1 font-medium text-white">Restore a backup</p>
				<p class="mb-3 text-sm text-gray-400">
					Download it on the old Ragbag (account menu → Users → Download backup). Every account, space, link and
					image comes along; afterwards everybody signs in with their old password.
				</p>
				<input
					type="file"
					accept=".gz,.json,application/gzip,application/json"
					disabled={busy}
					class="block w-full text-sm text-gray-300 file:me-3 file:rounded-lg file:border-0 file:bg-gray-700 file:px-3 file:py-2 file:text-white hover:file:bg-gray-600"
					onchange={(e) => check((e.currentTarget as HTMLInputElement).files?.[0])}
				/>
				{#if busy && !summary}
					<p class="mt-3 text-sm text-gray-400">Checking the backup…</p>
				{/if}
				{#if summary}
					<div class="mt-3 rounded-lg bg-gray-800 p-3 text-sm text-gray-300">
						<p>
							Made by Ragbag {summary.appVersion} on {summary.createdAt.toLocaleString()}:
						</p>
						<p class="mt-1 text-white">
							{count('users')} accounts · {count('spaces')} spaces · {count('collections')} collections ·
							{count('bookmarks')} links · {count('images')} images
						</p>
					</div>
					<Alert color="yellow" class="mt-3">
						Restoring replaces this setup account ({username}), so it needs no password of its own. Sign in
						with your account from the backup afterwards; admins are the ones in ADMIN_USERS.
					</Alert>
					<Button class="mt-3 w-full" disabled={busy} onclick={restore}>
						{busy ? 'Restoring…' : 'Restore this backup'}
					</Button>
				{/if}
			</div>
			{#if error}
				<Alert color="red">{error}</Alert>
			{/if}
		</div>
	{:else}
		<div class="flex flex-col gap-4 text-center">
			<p class="text-sm text-gray-300">
				{count('users')} accounts, {count('spaces')} spaces and {count('bookmarks')} links are here now. Sign in
				with your account from the backup.
			</p>
			<Button href={p('/login')}>Sign in</Button>
		</div>
	{/if}
</CenteredScreen>
