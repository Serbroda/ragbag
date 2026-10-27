<script lang="ts">
	import { Alert, Button, Input, Label } from 'flowbite-svelte';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { errorMessage, getPasswordReset, resetPassword } from '../api/client';
	import { navigate, p, route } from '../router';

	const MIN_PASSWORD_LENGTH = 8;
	const token = $derived(route.params.token ?? '');

	let linkState = $state<'loading' | 'valid' | 'invalid'>('loading');
	let username = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let saving = $state(false);

	$effect(() => {
		if (!token) return;
		getPasswordReset(token)
			.then((preview) => {
				linkState = preview.valid ? 'valid' : 'invalid';
				username = preview.username ?? '';
			})
			.catch(() => (linkState = 'invalid'));
	});

	async function submit(e: Event) {
		e.preventDefault();
		if (password !== confirmPassword) {
			error = 'Passwords do not match.';
			return;
		}
		saving = true;
		error = '';
		try {
			await resetPassword(token, password);
			navigate('/login', { search: { reset: '1' } });
		} catch (err) {
			error = await errorMessage(err, 'Failed to set the new password.');
		} finally {
			saving = false;
		}
	}
</script>

<CenteredScreen subtitle="Set a new password">
	{#if linkState === 'loading'}
		<p class="text-center text-gray-400">Checking link…</p>
	{:else if linkState === 'invalid'}
		<p class="mb-4 text-center text-gray-300">
			This reset link is invalid, expired or has already been used. Ask your administrator for a new one.
		</p>
		<a href={p('/login')} class="block text-center text-sm text-primary-500 hover:underline">Back to sign in</a>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={submit}>
			<p class="text-sm text-gray-300">
				Choose a new password for <strong class="text-white">{username}</strong>. You will be signed out on all
				devices.
			</p>
			<!-- Lets password managers update the right entry -->
			<input type="text" autocomplete="username" value={username} hidden readonly />
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
			<Button type="submit" disabled={saving || password.length < MIN_PASSWORD_LENGTH}>
				{saving ? 'Saving…' : 'Set new password'}
			</Button>
		</form>
	{/if}
</CenteredScreen>
