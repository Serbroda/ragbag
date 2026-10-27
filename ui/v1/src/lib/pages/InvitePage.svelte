<script lang="ts">
	import { InviteApi } from 'ragbag-frontend-sdk';
	import type { InvitePreviewDto } from 'ragbag-frontend-sdk';
	import { Alert, Button } from 'flowbite-svelte';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { navigate, route } from '../router';
	import { loadSpaces } from '../stores/spaces.svelte';

	const inviteApi = new InviteApi(apiConfig());
	const ROLE_LABELS: Record<string, string> = { ADMIN: 'admin', CONTRIBUTOR: 'contributor', VIEWER: 'viewer' };

	const token = $derived(route.params.token ?? '');

	let preview = $state<InvitePreviewDto | null>(null);
	let loading = $state(true);
	let accepting = $state(false);
	let error = $state('');

	$effect(() => {
		if (token) load(token);
	});

	async function load(t: string) {
		loading = true;
		error = '';
		try {
			preview = await inviteApi.getInvite({ token: t });
		} catch (err) {
			error = await errorMessage(err, 'This invite link is invalid.');
		} finally {
			loading = false;
		}
	}

	function openSpace() {
		if (preview) navigate('/space/:spaceId', { params: { spaceId: preview.targetId } });
	}

	async function accept() {
		accepting = true;
		error = '';
		try {
			preview = await inviteApi.acceptInvite({ token });
			await loadSpaces();
			openSpace();
		} catch (err) {
			error = await errorMessage(err, 'Could not accept the invite.');
		} finally {
			accepting = false;
		}
	}
</script>

<CenteredScreen>
	<div class="text-center">

		{#if loading}
			<p class="text-gray-500 dark:text-gray-400">Loading invite...</p>
		{:else if preview}
			<p class="mb-1 text-sm text-gray-500 dark:text-gray-400">
				{preview.invitedBy ?? 'Someone'} invited you to join
			</p>
			<p class="mb-1 text-xl font-semibold text-gray-900 dark:text-white">{preview.targetName ?? 'a space'}</p>
			{#if preview.role}
				<p class="mb-6 text-sm text-gray-500 dark:text-gray-400">as {ROLE_LABELS[preview.role]}</p>
			{/if}

			{#if error}
				<Alert color="red" class="mb-4">{error}</Alert>
			{/if}

			{#if preview.alreadyMember}
				<p class="mb-4 text-gray-700 dark:text-gray-300">You are already a member of this space.</p>
				<Button class="w-full" onclick={openSpace}>Open space</Button>
			{:else if !preview.valid}
				<Alert color="yellow" class="mb-4">This invite has expired or has been used up.</Alert>
				<Button class="w-full" color="light" onclick={() => navigate('/')}>Go to my spaces</Button>
			{:else}
				<Button class="w-full" disabled={accepting} onclick={accept}>
					{accepting ? 'Joining...' : 'Accept invite'}
				</Button>
			{/if}
		{:else}
			<Alert color="red" class="mb-4">{error}</Alert>
			<Button class="w-full" color="light" onclick={() => navigate('/')}>Go to my spaces</Button>
		{/if}
	</div>
</CenteredScreen>
