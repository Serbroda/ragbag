<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { Alert } from 'flowbite-svelte';
	import { StarOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { p, route } from '../router';
	import BookmarkCard from '../components/BookmarkCard.svelte';

	const collectionName = getContext<((id: string) => string | undefined) | undefined>('collection-name');

	const spaceId = $derived(route.params.spaceId ?? '');

	let favorites = $state<BookmarkDto[]>([]);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		if (spaceId) load(spaceId);
	});

	async function load(id: string) {
		loading = true;
		error = '';
		try {
			const result = await new BookmarkApi(apiConfig()).getFavorites({ spaceId: id });
			if (id === spaceId) favorites = result;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load favorites.');
		} finally {
			loading = false;
		}
	}
</script>

<div>
	<header class="mb-6">
		<h1 class="text-2xl font-bold tracking-tight">Favorites</h1>
		<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Your starred links in this space. Only you can see them.</p>
	</header>

	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if error}
		<Alert color="red">{error}</Alert>
	{:else if favorites.length === 0}
		<div
			class="flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-gray-300 py-14 text-center dark:border-gray-700"
		>
			<StarOutline class="h-8 w-8 text-gray-400" />
			<p class="font-medium text-gray-700 dark:text-gray-200">No favorites yet</p>
			<p class="max-w-sm text-sm text-gray-500 dark:text-gray-400">
				Hover over a link and click the star to keep it here.
			</p>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
			{#each favorites as bookmark (bookmark.id)}
				<BookmarkCard
					{bookmark}
					collectionName={bookmark.collectionId ? collectionName?.(bookmark.collectionId) : undefined}
					collectionHref={bookmark.collectionId
						? p('/space/:spaceId/collections/:collectionId', {
								params: { spaceId, collectionId: bookmark.collectionId },
							})
						: undefined}
					onfavoritechange={(favorite) => {
						if (!favorite) favorites = favorites.filter((b) => b.id !== bookmark.id);
					}}
				/>
			{/each}
		</div>
	{/if}
</div>
