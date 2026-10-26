<script lang="ts">
	import { Button, Card } from 'flowbite-svelte';
	import { logout } from '../api/client';
	import { navigate } from '../router';
	import CreateOrJoinSpaceModal from '../components/CreateOrJoinSpaceModal.svelte';
	import { lastSpaceId, loadMe, loadSpaces, resetSession, session } from '../stores/spaces.svelte';

	let failed = $state(false);
	let showCreateOrJoin = $state(false);

	async function init() {
		failed = false;
		try {
			await Promise.all([loadSpaces(), loadMe()]);
		} catch {
			failed = true;
			return;
		}
		// Reopen the space the user was in last, like Discord does
		const target = session.spaces.find((s) => s.id === lastSpaceId()) ?? session.spaces[0];
		if (target) {
			navigate('/space/:spaceId', { params: { spaceId: target.id }, replace: true });
		}
	}

	init();

	async function openSpace(spaceId: string) {
		await loadSpaces();
		navigate('/space/:spaceId', { params: { spaceId } });
	}

	async function handleLogout() {
		await logout().catch(() => {});
		resetSession();
		navigate('/login');
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-50 p-4 dark:bg-gray-900">
	{#if failed}
		<Card class="w-full max-w-md p-8 text-center">
			<p class="mb-4 text-gray-700 dark:text-gray-300">Could not load your spaces.</p>
			<div class="flex justify-center gap-2">
				<Button onclick={init}>Try again</Button>
				<Button color="light" onclick={handleLogout}>Log out</Button>
			</div>
		</Card>
	{:else if session.loaded && session.spaces.length === 0}
		<Card class="w-full max-w-md p-10 text-center">
			<h2 class="brand-glow mb-2 text-3xl font-bold tracking-tight">Ragbag</h2>
			<p class="mb-1 text-lg font-semibold text-gray-900 dark:text-white">
				Welcome{session.me ? `, ${session.me.username}` : ''}!
			</p>
			<p class="mb-6 text-sm text-gray-500 dark:text-gray-400">
				You are not part of any space yet. Create your own or join one with an invite link.
			</p>
			<Button class="w-full" onclick={() => (showCreateOrJoin = true)}>Create or join a space</Button>
			<button
				type="button"
				class="mt-4 text-sm text-gray-500 hover:underline dark:text-gray-400"
				onclick={handleLogout}
			>
				Log out
			</button>
		</Card>
	{:else}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{/if}
</div>

<CreateOrJoinSpaceModal bind:open={showCreateOrJoin} ondone={openSpace} />
