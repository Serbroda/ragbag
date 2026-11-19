<script lang="ts">
	import { Alert, Button, Input, Label } from 'flowbite-svelte';
	import { untrack } from 'svelte';
	import { errorMessage, ssoSignup } from '../api/client';
	import { continueAfterSignIn } from '../utils/signin';
	import type { LegalLink } from '../utils/legal';

	/**
	 * Back from the provider without an account here: choose a username (proposed by the provider) and create it.
	 * Only when the instance lets the provider create accounts (OIDC_AUTO_REGISTER) or an invite does.
	 */
	let {
		ticket,
		email,
		proposedUsername,
		proposedName,
		provider,
		allowed,
		inviteToken,
		terms,
		privacy,
		oncancel,
	}: {
		ticket: string;
		email: string;
		proposedUsername: string;
		proposedName: string;
		/** OIDC_NAME, e.g. "Authentik" */
		provider: string;
		allowed: boolean;
		/** An invite that may create the account (and is joined with it) */
		inviteToken?: string;
		terms?: LegalLink;
		privacy?: LegalLink;
		oncancel: () => void;
	} = $props();

	// Prefilled once with the provider's proposal; usernames are lower case without spaces, it may be neither
	let username = $state(untrack(() => proposedUsername.toLowerCase().replace(/\s+/g, '')));
	let displayName = $state(untrack(() => proposedName.trim().slice(0, 50)));
	let acceptTerms = $state(false);
	let error = $state('');
	let confirmationSent = $state(false);
	let loading = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			const signedIn = await ssoSignup(ticket, username.trim(), displayName.trim() || undefined, acceptTerms, inviteToken);
			if (signedIn) await continueAfterSignIn();
			else confirmationSent = true;
		} catch (err) {
			error = await errorMessage(err, 'The account could not be created. Please try again.');
		} finally {
			loading = false;
		}
	}
</script>

{#if confirmationSent}
	<Alert color="green">Almost done: we sent a confirmation link to {email}. Open it to start using Ragbag.</Alert>
{:else if !allowed}
	<Alert color="yellow" class="mb-4">
		No Ragbag account is connected to this {provider} login yet. Sign in with your password and connect it under
		Settings → Security.
	</Alert>
	<Button color="alternative" class="w-full" onclick={oncancel}>Back to sign in</Button>
{:else}
	<p class="mb-4 text-center text-sm text-gray-500 dark:text-gray-400">
		Signed in with {provider} as <span class="font-medium text-gray-900 dark:text-white">{email}</span>. Choose how
		others see you here.
	</p>
	{#if error}
		<Alert color="red" class="mb-4">{error}</Alert>
	{/if}
	<form onsubmit={submit} class="flex flex-col gap-4">
		<div>
			<Label for="sso-username" class="mb-2">Username</Label>
			<Input id="sso-username" type="text" autocomplete="username" bind:value={username} required />
			<p class="mt-1 text-xs text-gray-500">Stored in lower case, without spaces.</p>
		</div>
		<div>
			<Label for="sso-name" class="mb-2">Display name <span class="font-normal text-gray-500">(optional)</span></Label>
			<Input id="sso-name" type="text" autocomplete="nickname" maxlength={50} bind:value={displayName} />
		</div>
		{#if terms}
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
		{/if}
		<Button type="submit" class="mt-2" disabled={loading}>{loading ? 'Creating account...' : 'Create account'}</Button>
	</form>
	<div class="mt-4 text-center text-sm">
		<button type="button" class="text-gray-500 hover:underline dark:text-gray-400" onclick={oncancel}>Cancel</button>
	</div>
{/if}
