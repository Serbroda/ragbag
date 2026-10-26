<script lang="ts">
	import { Label, Input, Button, Alert } from 'flowbite-svelte';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { login, register, fetchFeatureFlags, takePendingInvite } from '../api/client';
	import { navigate } from '../router';
	import { resetSession } from '../stores/spaces.svelte';

	let registrationEnabled = $state(false);
	let isRegister = $state(false);

	$effect(() => {
		fetchFeatureFlags().then((flags) => {
			registrationEnabled = flags['registration_enabled'] === 'true';
		}).catch(() => {
			registrationEnabled = false;
		});
	});
	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	function resetForm() {
		username = '';
		email = '';
		password = '';
		confirmPassword = '';
		error = '';
		success = '';
	}

	function toggleMode() {
		isRegister = !isRegister;
		resetForm();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		loading = true;

		try {
			if (isRegister) {
				if (password !== confirmPassword) {
					error = 'Passwords do not match.';
					loading = false;
					return;
				}
				await register(username, email, password);
				success = 'Registration successful! You can now sign in.';
				isRegister = false;
				username = '';
				email = '';
				password = '';
				confirmPassword = '';
			} else {
				await login(username, password);
				resetSession();
				const pendingInvite = takePendingInvite();
				if (pendingInvite) {
					navigate('/invite/:token', { params: { token: pendingInvite } });
				} else {
					navigate('/');
				}
			}
		} catch (err) {
			error = isRegister
				? 'Registration failed. Please try again.'
				: 'Login failed. Please check your credentials.';
		} finally {
			loading = false;
		}
	}
</script>

<CenteredScreen subtitle={isRegister ? 'Create your account' : 'Welcome back! Sign in to continue.'}>

		{#if error}
			<Alert color="red" class="mb-4">{error}</Alert>
		{/if}
		{#if success}
			<Alert color="green" class="mb-4">{success}</Alert>
		{/if}

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<div>
				<Label for="username" class="mb-2">Username</Label>
				<Input id="username" type="text" autocomplete="username" bind:value={username} required />
			</div>
			{#if isRegister}
				<div>
					<Label for="email" class="mb-2">Email</Label>
					<Input id="email" type="email" bind:value={email} required />
				</div>
			{/if}
			<div>
				<Label for="password" class="mb-2">Password</Label>
				<Input
					id="password"
					type="password"
					autocomplete={isRegister ? 'new-password' : 'current-password'}
					bind:value={password}
					required
				/>
			</div>
			{#if isRegister}
				<div>
					<Label for="confirmPassword" class="mb-2">Confirm Password</Label>
					<Input id="confirmPassword" type="password" bind:value={confirmPassword} required />
				</div>
			{/if}
			<Button type="submit" class="mt-2" disabled={loading}>
				{#if loading}
					{isRegister ? 'Creating account...' : 'Signing in...'}
				{:else}
					{isRegister ? 'Create account' : 'Sign in'}
				{/if}
			</Button>
		</form>

		{#if registrationEnabled}
			<div class="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
				{#if isRegister}
					Already have an account?
					<button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-500" onclick={toggleMode}>
						Sign in
					</button>
				{:else}
					Don't have an account?
					<button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-500" onclick={toggleMode}>
						Register
					</button>
				{/if}
			</div>
		{/if}
</CenteredScreen>
