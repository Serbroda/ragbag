<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { Alert } from 'flowbite-svelte';
	import { errorMessage, searchBookmarks } from '../api/client';
	import { p, route } from '../router';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import LinkViewToggle from '../components/LinkViewToggle.svelte';
	import { linkListClass, preferences } from '../stores/preferences.svelte';

	const collectionName = getContext<((id: string) => string | undefined) | undefined>('collection-name');

	let bookmarks = $state<BookmarkDto[]>([]);
	let loading = $state(false);
	let error = $state('');

	const spaceId = $derived(route.params.spaceId ?? '');
	const query = $derived(typeof route.search.q === 'string' ? route.search.q : String(route.search.q ?? ''));
	/** "#tag" (from clicking a tag) only matches that exact tag. */
	const isTag = $derived(/^#\S+$/.test(query.trim()));

	$effect(() => {
		if (spaceId && query.trim()) {
			doSearch(spaceId, query);
		} else {
			bookmarks = [];
			loading = false;
		}
	});

	async function doSearch(sid: string, q: string) {
		loading = true;
		error = '';
		try {
			const result = await searchBookmarks(sid, q);
			// Ignore responses for queries the user has already typed past
			if (q === query && sid === spaceId) bookmarks = result;
		} catch (err) {
			error = await errorMessage(err, 'Failed to search bookmarks.');
		} finally {
			loading = false;
		}
	}
</script>

<div>
	<div class="mb-6 flex items-start justify-between gap-3">
		<div class="min-w-0">
			<h1 class="truncate text-2xl font-bold tracking-tight">
				{isTag ? `Links tagged ${query}` : `Results for “${query}”`}
			</h1>
			{#if !loading}
				<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
					{bookmarks.length} result{bookmarks.length !== 1 ? 's' : ''} found
				</p>
			{/if}
		</div>
		<LinkViewToggle />
	</div>

	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Searching...</p>
	{:else if error}
		<Alert color="red" class="mb-4">{error}</Alert>
	{:else if bookmarks.length === 0}
		<div
			class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-gray-600"
		>
			<p class="text-gray-500 dark:text-gray-400">No bookmarks found matching your search.</p>
		</div>
	{:else}
		<div class={linkListClass(preferences.linkView)}>
			{#each bookmarks as bookmark (bookmark.id)}
				<BookmarkCard
					{bookmark}
					collectionName={bookmark.collectionId ? collectionName?.(bookmark.collectionId) : undefined}
					collectionHref={bookmark.collectionId
						? p('/space/:spaceId/collections/:collectionId', {
								params: { spaceId, collectionId: bookmark.collectionId },
							})
						: undefined}
				/>
			{/each}
		</div>
	{/if}
</div>
