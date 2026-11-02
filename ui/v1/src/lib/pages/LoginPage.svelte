<script lang="ts">
	import { Label, Input, Button, Alert } from 'flowbite-svelte';
	import { ResponseError } from 'ragbag-frontend-sdk';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import type { InvitePreviewDto } from 'ragbag-frontend-sdk';
	import {
		errorMessage, login, register, fetchFeatureFlags, requestPasswordReset, pendingInvite, getRegistrationInvite,
	} from '../api/client';
	import { route } from '../router';
	import { continueAfterSignIn } from '../utils/signin';
	import { legalLinks, type LegalLink } from '../utils/legal';

	let registrationEnabled = $state(false);
	// The server sends mails: new accounts confirm their address, passwords can be reset by mail
	let emailEnabled = $state(false);
	let isRegister = $state(false);
	let isForgot = $state(false);
	// With terms of use, registering means accepting them (and the privacy policy is linked next to it)
	let terms = $state<LegalLink | undefined>();
	let privacy = $state<LegalLink | undefined>();
	let acceptTerms = $state(false);
	legalLinks().then((links) => {
		terms = links.find((link) => link.page === 'terms');
		privacy = links.find((link) => link.page === 'privacy');
	});

	// An invite opened while signed out; it may create an account even while registration is disabled
	const inviteToken = pendingInvite();
	let invite = $state<InvitePreviewDto | null>(null);
	const inviteRegisters = $derived(invite?.registration === true);
	if (inviteToken) {
		getRegistrationInvite(inviteToken).then((preview) => {
			invite = preview;
			// Mostly people new here: start with the registration
			if (preview.registration && !isForgot && !username) isRegister = true;
		}).catch(() => {});
	}

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
		acceptTerms = false;
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
				await register(username, email, password, acceptTerms, inviteRegisters ? inviteToken ?? undefined : undefined);
				const space = inviteRegisters ? invite?.targetName ?? 'the space' : '';
				success = emailEnabled
					? `Almost done: we sent a confirmation link to ${email}. Open it to start using Ragbag${space ? ` and join ${space}` : ''}.`
					: `Registration successful! You can now sign in${space ? ` and open ${space}` : ''}.`;
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

		{#if invite?.valid && !success}
			<p class="mb-4 text-center text-sm text-gray-500 dark:text-gray-400">
				{invite.invitedBy ?? 'Someone'} invited you to join
				<span class="font-semibold text-gray-900 dark:text-white">{invite.targetName ?? 'a space'}</span>.
				{inviteRegisters ? 'Create an account or sign in to accept.' : 'Sign in to accept.'}
			</p>
		{/if}
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
			{#if isRegister && terms}
				<label class="flex items-start gap-2 text-sm text-gray-300">
					<input
						type="checkbox"
						class="mt-0.5 rounded border-gray-600 bg-gray-700 text-primary-600 focus:ring-primary-500"
						bind:checked={acceptTerms}
						required
					/>
					<span>
						I accept the
						<a href={terms.href} target="_blank" rel="noopener noreferrer" class="text-primary-500 hover:underline">terms of use</a
						>{#if privacy}
							and have read the
							<a href={privacy.href} target="_blank" rel="noopener noreferrer" class="text-primary-500 hover:underline"
								>privacy policy</a
							>{/if}.
					</span>
				</label>
			{:else if isRegister && privacy}
				<p class="text-xs text-gray-400">
					How your data is used:
					<a href={privacy.href} target="_blank" rel="noopener noreferrer" class="text-primary-500 hover:underline">privacy policy</a>.
				</p>
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

		{#if (registrationEnabled || inviteRegisters) && !isForgot}
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
