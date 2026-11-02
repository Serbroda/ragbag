<script lang="ts">
	import { Label, Input, Button, Alert } from 'flowbite-svelte';
	import { ResponseError } from 'ragbag-frontend-sdk';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { errorMessage, login, register, fetchFeatureFlags, requestPasswordReset } from '../api/client';
	import { route } from '../router';
	import { continueAfterSignIn } from '../utils/signin';

	let registrationEnabled = $state(false);
	// The server sends mails: new accounts confirm their address, passwords can be reset by mail
	let emailEnabled = $state(false);
	let isRegister = $state(false);
	let isForgot = $state(false);

	$effect(() => {
		fetchFeatureFlags().then((flags) => {
			registrationEnabled = flags['registration_enabled'] === 'true';
			emailEnabled = flags['email'] === 'true';
		}).catch(() => {
			registrationEnabled = false;
		});
	});
	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	// Set after a password reset, which sends the user here
	let success = $state(route.search.reset ? 'Your password has been changed. Sign in with the new one.' : '');
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
		isForgot = false;
		resetForm();
	}

	function toggleForgot() {
		isForgot = !isForgot;
		isRegister = false;
		resetForm();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		loading = true;

		try {
			if (isForgot) {
				await requestPasswordReset(username);
				success = 'If an account exists for this, we sent it a link to reset the password. Check your inbox (and the spam folder).';
				isForgot = false;
				username = '';
			} else if (isRegister) {
				if (password !== confirmPassword) {
					error = 'Passwords do not match.';
					loading = false;
					return;
				}
				await register(username, email, password);
				success = emailEnabled
					? `Almost done: we sent a confirmation link to ${email}. Open it to start using Ragbag.`
					: 'Registration successful! You can now sign in.';
				isRegister = false;
				username = '';
				email = '';
				password = '';
				confirmPassword = '';
			} else {
				await login(username, password);
				continueAfterSignIn();
			}
		} catch (err) {
			// Wrong credentials get the friendly text; unconfirmed addresses and rate limits the server's
			error = isForgot
				? await errorMessage(err, 'The reset link could not be sent. Please try again.')
				: isRegister
					? await errorMessage(err, 'Registration failed. Please try again.')
					: err instanceof ResponseError && err.response.status !== 401
						? await errorMessage(err, 'Login failed. Please try again.')
						: 'Login failed. Please check your credentials.';
		} finally {
			loading = false;
		}
	}
</script>

<CenteredScreen
	subtitle={isForgot ? 'Reset your password' : isRegister ? 'Create your account' : 'Welcome back! Sign in to continue.'}
>

		{#if error}
			<Alert color="red" class="mb-4">{error}</Alert>
		{/if}
		{#if success}
			<Alert color="green" class="mb-4">{success}</Alert>
		{/if}

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<div>
				<Label for="username" class="mb-2">{isForgot ? 'Username or email' : 'Username'}</Label>
				<Input id="username" type="text" autocomplete="username" bind:value={username} required />
			</div>
			{#if isRegister}
				<div>
					<Label for="email" class="mb-2">Email</Label>
					<Input id="email" type="email" bind:value={email} required />
				</div>
			{/if}
			{#if !isForgot}
				<div>
					<Label for="password" class="mb-2">Password</Label>
					<Input
						id="password"
						type="password"
						autocomplete={isRegister ? 'new-password' : 'current-password'}
						minlength={isRegister ? 8 : undefined}
						placeholder={isRegister ? 'At least 8 characters' : undefined}
						bind:value={password}
						required
					/>
				</div>
			{/if}
			{#if isRegister}
				<div>
					<Label for="confirmPassword" class="mb-2">Confirm Password</Label>
					<Input id="confirmPassword" type="password" bind:value={confirmPassword} required />
				</div>
			{/if}
			<Button type="submit" class="mt-2" disabled={loading}>
				{#if loading}
					{isForgot ? 'Sending...' : isRegister ? 'Creating account...' : 'Signing in...'}
				{:else}
					{isForgot ? 'Send reset link' : isRegister ? 'Create account' : 'Sign in'}
				{/if}
			</Button>
		</form>

		{#if isForgot}
			<div class="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
				<button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-500" onclick={toggleForgot}>
					Back to sign in
				</button>
			</div>
		{:else if !isRegister}
			{#if emailEnabled}
				<div class="mt-4 text-center text-sm">
					<button type="button" class="text-gray-500 hover:underline dark:text-gray-400" onclick={toggleForgot}>
						Forgot your password?
					</button>
				</div>
			{:else}
				<p class="mt-4 text-center text-xs text-gray-500">
					Forgot your password? Ask the administrator of this Ragbag for a reset link.
				</p>
			{/if}
		{/if}

		{#if registrationEnabled && !isForgot}
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
