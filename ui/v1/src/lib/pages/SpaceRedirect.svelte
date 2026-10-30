<script lang="ts">
	import { Button } from 'flowbite-svelte';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { logout } from '../api/client';
	import { navigate } from '../router';
	import CreateOrJoinSpaceModal from '../components/CreateOrJoinSpaceModal.svelte';
	import { UserApi } from 'ragbag-frontend-sdk';
	import { apiConfig } from '../api/client';
	import { loadMe, loadSpaces, resetSession, session, startSpaceId } from '../stores/spaces.svelte';

	let failed = $state(false);
	let showCreateOrJoin = $state(false);

	async function init() {
		failed = false;
		let bar: string[] = [];
		try {
			const [, , layout] = await Promise.all([
				loadSpaces(),
				loadMe(),
				new UserApi(apiConfig()).getSpaceBar().catch(() => ({ items: [] })),
			]);
			// The order of the space bar, folders included
			bar = layout.items.flatMap((item) => (item.spaceId ? [item.spaceId] : (item.spaceIds ?? [])));
		} catch {
			failed = true;
			return;
		}
		// The chosen start space, else the one the user was in last (like Discord), else the first in their bar
		const target = startSpaceId(bar);
		if (target) {
			navigate('/space/:spaceId', { params: { spaceId: target }, replace: true });
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

{#if failed}
	<CenteredScreen>
		<div class="text-center">
			<p class="mb-4 text-gray-300">Could not load your spaces.</p>
			<div class="flex justify-center gap-2">
				<Button onclick={init}>Try again</Button>
				<Button color="alternative" onclick={handleLogout}>Log out</Button>
			</div>
		</div>
	</CenteredScreen>
{:else if session.loaded && session.spaces.length === 0}
	<CenteredScreen subtitle="Collect and share links with your people.">
		<div class="text-center">
			<p class="mb-1 text-lg font-semibold text-white">
				Welcome{session.me ? `, ${session.me.username}` : ''}!
			</p>
			<p class="mb-6 text-sm text-gray-400">
				You are not part of any space yet. Create your own or join one with an invite link.
			</p>
			<Button class="w-full" onclick={() => (showCreateOrJoin = true)}>Create or join a space</Button>
			<button type="button" class="mt-4 text-sm text-gray-400 hover:underline" onclick={handleLogout}>
				Log out
			</button>
		</div>
	</CenteredScreen>
{:else}
	<div class="flex min-h-dvh items-center justify-center bg-gray-950 text-gray-500">Loading...</div>
{/if}

<CreateOrJoinSpaceModal bind:open={showCreateOrJoin} ondone={openSpace} />
