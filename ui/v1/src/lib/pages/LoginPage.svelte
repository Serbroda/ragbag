<script lang="ts">
	import { Label, Input, Button, Alert } from 'flowbite-svelte';
	import { ResponseError } from 'ragbag-frontend-sdk';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import type { InvitePreviewDto } from 'ragbag-frontend-sdk';
	import {
		errorMessage, login, register, fetchFeatureFlags, requestPasswordReset, pendingInvite, getRegistrationInvite,
		needsTwoFactorCode, needsEmailConfirmation, resendVerification, retryAfter, startSso, finishSso,
	} from '../api/client';
	import SsoSignup from '../components/SsoSignup.svelte';
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

	// Single sign-on (OIDC_ISSUER): the provider's name for the button; with sso_only the password form is only
	// a fallback for instance admins
	let sso = $state('');
	let ssoOnly = $state(false);
	let ssoCreatesAccounts = $state(false);
	let showPasswordForm = $state(false);
	let leavingForSso = $state(false);

	$effect(() => {
		fetchFeatureFlags().then((flags) => {
			registrationEnabled = flags['registration_enabled'] === 'true';
			emailEnabled = flags['email'] === 'true';
			sso = flags['sso'] ?? '';
			ssoOnly = flags['sso_only'] === 'true';
			ssoCreatesAccounts = flags['sso_signup'] === 'true';
		}).catch(() => {
			registrationEnabled = false;
		});
	});

	async function signInWithSso() {
		error = '';
		leavingForSso = true;
		try {
			await startSso();
		} catch (err) {
			error = await errorMessage(err, 'The sign-in could not be started. Please try again.');
			leavingForSso = false;
		}
	}

	// Back from the provider: signed in (the refresh cookie is set), new here (the ticket is in the fragment, which
	// never reaches a server log) or with an error
	let ssoSignupTicket = $state<URLSearchParams | null>(null);
	let finishingSso = $state(route.search.sso === 'done');
	if (route.search.sso === 'done') {
		finishSso().then((signedIn) => {
			if (signedIn) continueAfterSignIn();
			else {
				finishingSso = false;
				error = 'The sign-in could not be finished. Please try again.';
			}
		});
	} else if (route.search.sso === 'signup' && location.hash.length > 1) {
		ssoSignupTicket = new URLSearchParams(location.hash.slice(1));
		history.replaceState(history.state, '', location.pathname);
	}

	function cancelSsoSignup() {
		ssoSignupTicket = null;
		history.replaceState(history.state, '', '/login');
	}
	let username = $state('');
	let displayName = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	// Second step of signing in to an account with two-factor authentication
	let codeStep = $state(false);
	let code = $state('');
	let error = $state(typeof route.search.sso_error === 'string' ? route.search.sso_error : '');
	// Set after a password reset, which sends the user here
	let success = $state(route.search.reset ? 'Your password has been changed. Sign in with the new one.' : '');
	let loading = $state(false);

	/** Pause between two confirmation mails, like the server's */
	const RESEND_PAUSE = 120;
	// An account that is not confirmed yet: its credentials (only in memory) to send the link again
	let unconfirmed = $state<{ login: string; password: string } | null>(null);
	let resendAt = $state(0);
	let now = $state(Date.now());
	let resending = $state(false);
	const resendIn = $derived(Math.max(0, Math.ceil((resendAt - now) / 1000)));
	$effect(() => {
		if (resendIn === 0) return;
		const timer = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(timer);
	});

	function waitBeforeResend(seconds: number) {
		now = Date.now();
		resendAt = now + seconds * 1000;
	}

	async function resend() {
		if (!unconfirmed) return;
		resending = true;
		error = '';
		success = '';
		try {
			await resendVerification(unconfirmed.login, unconfirmed.password);
			success = 'We sent a new confirmation link. Check your inbox (and the spam folder).';
			waitBeforeResend(RESEND_PAUSE);
		} catch (err) {
			const wait = retryAfter(err);
			if (wait) waitBeforeResend(wait);
			error = await errorMessage(err, 'The link could not be sent. Please try again.');
		} finally {
			resending = false;
		}
	}

	function resetForm() {
		username = '';
		displayName = '';
		email = '';
		password = '';
		confirmPassword = '';
		acceptTerms = false;
		codeStep = false;
		code = '';
		error = '';
		success = '';
		unconfirmed = null;
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
				const login = username;
				await register(
					username, email, password, acceptTerms, inviteRegisters ? inviteToken ?? undefined : undefined,
					displayName.trim() || undefined,
				);
				const space = inviteRegisters ? invite?.targetName ?? 'the space' : '';
				success = emailEnabled
					? `Almost done: we sent a confirmation link to ${email}. Open it to start using Ragbag${space ? ` and join ${space}` : ''}.`
					: `Registration successful! You can now sign in${space ? ` and open ${space}` : ''}.`;
				isRegister = false;
				if (emailEnabled) {
					unconfirmed = { login, password };
					waitBeforeResend(RESEND_PAUSE);
				}
				username = '';
				displayName = '';
				email = '';
				password = '';
				confirmPassword = '';
			} else {
				unconfirmed = null;
				await login(username, password, codeStep ? code : undefined);
				continueAfterSignIn();
			}
		} catch (err) {
			if (needsTwoFactorCode(err)) {
				// The password was right: ask for the code, or say that it was wrong
				error = codeStep ? 'That code is not right. Codes change every 30 seconds.' : '';
				codeStep = true;
				code = '';
				return;
			}
			if (needsEmailConfirmation(err)) unconfirmed = { login: username, password };
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
	subtitle={ssoSignupTicket
		? 'Create your account'
		: isForgot
		? 'Reset your password'
		: isRegister
			? 'Create your account'
			: codeStep
				? 'Enter the code of your authenticator app.'
				: 'Welcome back! Sign in to continue.'}
>

		{#if finishingSso}
			<p class="text-center text-sm text-gray-500 dark:text-gray-400">Signing in…</p>
		{:else if ssoSignupTicket}
			<SsoSignup
				ticket={ssoSignupTicket.get('ticket') ?? ''}
				email={ssoSignupTicket.get('email') ?? ''}
				proposedUsername={ssoSignupTicket.get('username') ?? ''}
				proposedName={ssoSignupTicket.get('name') ?? ''}
				provider={sso || 'single sign-on'}
				allowed={ssoCreatesAccounts || inviteRegisters}
				inviteToken={inviteRegisters ? (inviteToken ?? undefined) : undefined}
				{terms}
				{privacy}
				oncancel={cancelSsoSignup}
			/>
		{:else}
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
		{#if unconfirmed}
			<p class="-mt-2 mb-4 text-center text-sm text-gray-500 dark:text-gray-400">
				No email?
				{#if resendIn}
					You can ask for a new one in {Math.floor(resendIn / 60)}:{String(resendIn % 60).padStart(2, '0')}.
				{:else}
					<button type="button" class="font-medium text-primary-500 hover:underline disabled:opacity-50" disabled={resending} onclick={resend}>
						Send it again
					</button>
				{/if}
			</p>
		{/if}

		{#if sso && !isRegister && !isForgot && !codeStep}
			<Button color={ssoOnly ? 'primary' : 'alternative'} class="w-full" disabled={leavingForSso} onclick={signInWithSso}>
				{leavingForSso ? 'Opening…' : `Sign in with ${sso}`}
			</Button>
			{#if !ssoOnly}
				<div class="my-4 flex items-center gap-3 text-xs text-gray-500">
					<span class="h-px flex-1 bg-gray-200 dark:bg-gray-700"></span>or<span class="h-px flex-1 bg-gray-200 dark:bg-gray-700"></span>
				</div>
			{/if}
		{/if}

		{#if ssoOnly && !showPasswordForm && !codeStep}
			<div class="mt-4 text-center text-xs">
				<button type="button" class="text-gray-500 hover:underline dark:text-gray-400" onclick={() => (showPasswordForm = true)}>
					Administrators: sign in with a password
				</button>
			</div>
		{:else}
		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			{#if codeStep}
				<div>
					<Label for="code" class="mb-2">Code</Label>
					<!-- svelte-ignore a11y_autofocus -->
					<Input
						id="code"
						type="text"
						inputmode="numeric"
						autocomplete="one-time-code"
						placeholder="123 456"
						bind:value={code}
						required
						autofocus
					/>
					<p class="mt-1 text-xs text-gray-500">No access to the app? Enter one of your recovery codes instead.</p>
				</div>
			{/if}
			<div class={codeStep ? 'hidden' : ''}>
				<Label for="username" class="mb-2">{isForgot ? 'Username or email' : 'Username'}</Label>
				<Input id="username" type="text" autocomplete="username" bind:value={username} required />
				{#if isRegister}
					<p class="mt-1 text-xs text-gray-500">For signing in: stored in lower case, without spaces.</p>
				{/if}
			</div>
			{#if isRegister}
				<div>
					<Label for="displayName" class="mb-2">Display name <span class="font-normal text-gray-500">(optional)</span></Label>
					<Input id="displayName" type="text" autocomplete="nickname" maxlength={50} placeholder="e.g. Max Power" bind:value={displayName} />
				</div>
				<div>
					<Label for="email" class="mb-2">Email</Label>
					<Input id="email" type="email" bind:value={email} required />
				</div>
			{/if}
			{#if !isForgot}
				<div class={codeStep ? 'hidden' : ''}>
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
					<Label for="confirmPassword" class="mb-2">Confirm password</Label>
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

		{#if codeStep}
			<div class="mt-4 text-center text-sm">
				<button type="button" class="text-gray-500 hover:underline dark:text-gray-400" onclick={resetForm}>
					Back to sign in
				</button>
			</div>
		{/if}

		{#if codeStep || ssoOnly}
			<!-- Nothing else to choose in the second step; with single sign-on only, passwords are for admins -->
		{:else if isForgot}
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

		{/if}

		{#if (registrationEnabled || inviteRegisters) && !ssoOnly && !isForgot && !codeStep}
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
		{/if}
</CenteredScreen>
