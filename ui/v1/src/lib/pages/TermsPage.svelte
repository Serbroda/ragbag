<script lang="ts">
	import { Button, Input, Label } from 'flowbite-svelte';
	import type { LegalPageDto } from 'ragbag-frontend-sdk';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { acceptTerms, deleteAccount, errorMessage, getLegalPage, logout } from '../api/client';
	import { navigate } from '../router';
	import { loadMe, resetSession } from '../stores/spaces.svelte';
	import { showToast } from '../stores/toast.svelte';
	import { legalLinks, type LegalLink } from '../utils/legal';

	/**
	 * The terms of use changed: nothing else works until the user accepts them. Who does not agree signs out or
	 * deletes the account.
	 */
	let loading = $state(true);
	let link = $state<LegalLink | null>(null);
	let terms = $state<LegalPageDto | null>(null);
	let error = $state('');
	let busy = $state(false);
	let deleteOpen = $state(false);
	let deletePassword = $state('');

	async function init() {
		try {
			resetSession();
			const me = await loadMe();
			if (!me.termsPending) {
				navigate('/', { replace: true });
				return;
			}
			link = (await legalLinks()).find((l) => l.page === 'terms') ?? null;
			// Own pages are shown right here, addresses elsewhere are linked
			if (link && !link.external) terms = await getLegalPage('terms').catch(() => null);
		} catch (err) {
			error = await errorMessage(err, 'Could not load your account.');
		} finally {
			loading = false;
		}
	}

	init();

	async function accept() {
		busy = true;
		error = '';
		try {
			await acceptTerms();
			resetSession();
			navigate('/', { replace: true });
		} catch (err) {
			error = await errorMessage(err, 'Failed to accept the terms of use.');
		} finally {
			busy = false;
		}
	}

	async function signOut() {
		await logout().catch(() => {});
		resetSession();
		navigate('/login');
	}

	async function remove(e: Event) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await deleteAccount(deletePassword);
			resetSession();
			navigate('/login');
			showToast('Your account was deleted.');
		} catch (err) {
			error = await errorMessage(err, 'Failed to delete the account.');
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head>
	<title>Terms of use · Ragbag</title>
</svelte:head>

<CenteredScreen subtitle="The terms of use have changed">
	{#if loading}
		<p class="text-center text-gray-400">Loading…</p>
	{:else}
		<div class="flex flex-col gap-4">
			<p class="text-sm text-gray-300">
				To go on using Ragbag, please read the new terms of use and accept them.
			</p>
			{#if terms}
				<div class="max-h-80 overflow-y-auto rounded-lg border border-white/10 bg-gray-950/60 p-4 text-sm">
					<h2 class="mb-3 text-base font-semibold text-white">{terms.title}</h2>
					<!-- Rendered by the server from the operator's Markdown, without raw HTML -->
					<div class="markdown">{@html terms.html}</div>
				</div>
			{/if}
			{#if link}
				<a href={link.href} target="_blank" rel="noopener noreferrer" class="text-sm text-primary-400 hover:underline">
					{terms ? 'Open them in a new tab' : 'Read the terms of use'}
				</a>
			{/if}
			{#if error}
				<p class="text-sm text-red-400">{error}</p>
			{/if}
			<Button disabled={busy} onclick={accept}>Accept and continue</Button>

			<div class="border-t border-white/10 pt-4 text-sm text-gray-400">
				{#if deleteOpen}
					<form class="flex flex-col gap-3" onsubmit={remove}>
						<p>
							Deleting your account deletes the spaces only you are a member of, with their links. This cannot be
							undone.
						</p>
						<div>
							<Label for="delete-password" class="mb-2">Your password</Label>
							<Input id="delete-password" type="password" autocomplete="current-password" bind:value={deletePassword} required />
						</div>
						<div class="flex gap-2">
							<Button type="submit" color="red" disabled={busy || !deletePassword}>Delete my account</Button>
							<Button color="alternative" onclick={() => ((deleteOpen = false), (deletePassword = ''))}>Cancel</Button>
						</div>
					</form>
				{:else}
					<p class="mb-3">You do not agree? Sign out, or delete your account.</p>
					<div class="flex flex-wrap gap-2">
						<Button size="sm" color="alternative" onclick={signOut}>Sign out</Button>
						<Button size="sm" color="alternative" class="text-red-400" onclick={() => (deleteOpen = true)}>Delete account…</Button>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</CenteredScreen>
