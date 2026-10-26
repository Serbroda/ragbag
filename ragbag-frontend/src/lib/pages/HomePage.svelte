<script lang="ts">
	import { BookmarkApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { Alert, Button } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { navigate, route } from '../router';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import { session } from '../stores/spaces.svelte';

	const RECENT_LIMIT = 24;

	const collectionName = getContext<((id: string) => string | undefined) | undefined>('collection-name');
	const openInvite = getContext<(() => void) | undefined>('open-invite');

	const spaceId = $derived(route.params.spaceId ?? '');
	const space = $derived(session.spaces.find((s) => s.id === spaceId) ?? null);

	let recent = $state<BookmarkDto[]>([]);
	let memberCount = $state<number | null>(null);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		if (spaceId) load(spaceId);
	});

	async function load(id: string) {
		loading = true;
		error = '';
		try {
			const [bookmarks, members] = await Promise.all([
				new BookmarkApi(apiConfig()).getRecentBookmarks({ spaceId: id, limit: RECENT_LIMIT }),
				new SpaceApi(apiConfig()).getSpaceMembers({ spaceId: id }),
			]);
			if (id !== spaceId) return;
			recent = bookmarks;
			memberCount = members.length;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the dashboard.');
		} finally {
			loading = false;
		}
	}

	const greeting = $derived.by(() => {
		const hour = new Date().getHours();
		if (hour < 12) return 'Good morning';
		if (hour < 18) return 'Good afternoon';
		return 'Good evening';
	});
</script>

<main class="mx-auto max-w-6xl p-2 md:p-6">
	<header class="mb-6">
		<p class="text-sm text-gray-500 dark:text-gray-400">
			{greeting}{session.me ? `, ${session.me.username}` : ''} 👋
		</p>
		<h1 class="text-3xl font-bold text-gray-900 dark:text-white">{space?.name ?? 'Dashboard'}</h1>
		{#if space?.description}
			<p class="mt-1 text-gray-600 dark:text-gray-300">{space.description}</p>
		{/if}
		<div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
			{#if memberCount !== null}
				<span>{memberCount} member{memberCount === 1 ? '' : 's'}</span>
			{/if}
			{#if space?.role}
				<span>You are <span class="font-medium capitalize">{space.role.toLowerCase()}</span></span>
			{/if}
			<button
				type="button"
				class="text-primary-600 hover:underline dark:text-primary-400"
				onclick={() => navigate('/space/:spaceId/settings', { params: { spaceId } })}
			>
				Space settings
			</button>
		</div>
	</header>

	<section>
		<h2 class="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
			Recently added
		</h2>
		{#if loading}
			<p class="text-gray-500 dark:text-gray-400">Loading...</p>
		{:else if error}
			<Alert color="red">{error}</Alert>
		{:else if recent.length === 0}
			<div
				class="flex flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 py-12 text-center dark:border-gray-600"
			>
				<p class="font-medium text-gray-700 dark:text-gray-200">Nothing here yet</p>
				<p class="max-w-sm text-sm text-gray-500 dark:text-gray-400">
					Create a collection in the sidebar and start adding links. Everything new in this space shows up here.
				</p>
				{#if space?.permissions?.includes('MANAGE_MEMBERS')}
					<Button size="sm" color="light" class="mt-2" onclick={() => openInvite?.()}>Invite people</Button>
				{/if}
			</div>
		{:else}
			<div class="flex flex-col gap-4 md:flex-row md:flex-wrap">
				{#each recent as bookmark (bookmark.id)}
					<BookmarkCard
						{bookmark}
						collectionName={bookmark.collectionId ? collectionName?.(bookmark.collectionId) : undefined}
						collectionHref={bookmark.collectionId
							? `/space/${spaceId}/collections/${bookmark.collectionId}`
							: undefined}
					/>
				{/each}
			</div>
		{/if}
	</section>
</main>
