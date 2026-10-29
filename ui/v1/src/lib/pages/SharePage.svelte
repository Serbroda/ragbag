<script lang="ts">
	import type { SharedCollectionDto } from 'ragbag-frontend-sdk';
	import { ChevronRightOutline, FolderOutline, SearchOutline } from 'flowbite-svelte-icons';
	import { errorMessage, getSharedCollection } from '../api/client';
	import { p, route } from '../router';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import LinkViewToggle from '../components/LinkViewToggle.svelte';
	import LinkSortSelect from '../components/LinkSortSelect.svelte';
	import { filterByTags, filterLinks, sortLinks } from '../utils/links';
	import TagFilter from '../components/TagFilter.svelte';
	import Logo from '../components/Logo.svelte';
	import { linkListClass, loadPreferences, preferences } from '../stores/preferences.svelte';

	/**
	 * What a public link shows: the shared collection like in the app, without sidebar, spaces or anything
	 * personal. Sub-collections open within the shared view.
	 */
	// Visitors have no account: their display and sort choices are kept for this browser
	loadPreferences('public');

	const token = $derived(route.params.token ?? '');
	const collectionId = $derived(route.params.collectionId);

	let shared = $state<SharedCollectionDto | null>(null);
	let loading = $state(true);
	let error = $state('');
	let filter = $state('');

	$effect(() => {
		load(token, collectionId);
	});

	async function load(t: string, id: string | undefined) {
		loading = true;
		error = '';
		filter = '';
		selectedTags = [];
		try {
			const result = await getSharedCollection(t, id);
			if (t === token && id === collectionId) shared = result;
		} catch (err) {
			shared = null;
			error = await errorMessage(err, 'This link does not exist or was revoked.');
		} finally {
			loading = false;
		}
	}

	function href(id: string): string {
		return id === shared?.path[0]?.id
			? p('/share/:token', { params: { token } })
			: p('/share/:token/:collectionId', { params: { token, collectionId: id } });
	}

	let selectedTags = $state<string[]>([]);
	const bookmarks = $derived(
		sortLinks(filterByTags(filterLinks(shared?.bookmarks ?? [], filter), selectedTags), preferences.linkSort),
	);
</script>

<svelte:head>
	<title>{shared ? `${shared.name} · Ragbag` : 'Ragbag'}</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<div class="min-h-dvh bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100">
	<header class="sticky top-0 z-20 border-b border-gray-200 bg-white/90 backdrop-blur dark:border-gray-950/60 dark:bg-gray-850/90">
		<div class="mx-auto flex h-12 w-full max-w-6xl items-center gap-3 px-4 md:px-8">
			<Logo size={28} />
			<nav aria-label="Breadcrumb" class="min-w-0 flex-1">
				<ol class="flex min-w-0 items-center gap-1 text-sm">
					{#each shared?.path ?? [] as crumb, i (crumb.id)}
						{@const last = i === (shared?.path.length ?? 0) - 1}
						<li class="items-center gap-1 {last ? 'flex min-w-0' : 'hidden shrink-0 sm:flex'}">
							{#if i > 0}
								<ChevronRightOutline class="hidden h-3.5 w-3.5 shrink-0 text-gray-400 sm:block" />
							{/if}
							{#if last}
								<span class="truncate px-1 font-semibold" aria-current="page">{crumb.name}</span>
							{:else}
								<a
									href={href(crumb.id)}
									class="max-w-48 truncate rounded px-1 py-0.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
								>
									{crumb.name}
								</a>
							{/if}
						</li>
					{/each}
				</ol>
			</nav>
			<span class="hidden shrink-0 text-xs text-gray-500 sm:inline">Shared with Ragbag</span>
		</div>
	</header>

	<main class="mx-auto w-full max-w-6xl px-4 py-6 md:px-8">
		{#if loading && !shared}
			<p class="text-gray-500 dark:text-gray-400">Loading...</p>
		{:else if error}
			<div class="mx-auto mt-16 max-w-md text-center">
				<p class="text-lg font-semibold">Nothing to see here</p>
				<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{error}</p>
			</div>
		{:else if shared}
			<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div class="min-w-0">
					<h1 class="break-words text-xl font-bold tracking-tight sm:text-2xl">{shared.name}</h1>
					{#if shared.description}
						<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{shared.description}</p>
					{/if}
				</div>
				<div class="flex shrink-0 items-center gap-2">
					{#if shared.bookmarks.length > 0}
						<div class="relative">
							<input
								type="search"
								placeholder="Filter links"
								class="w-44 rounded-md border-0 bg-gray-100 py-1.5 pl-8 pr-3 text-sm placeholder:text-gray-500 focus:ring-2 focus:ring-primary-500 dark:bg-gray-950/70"
								bind:value={filter}
							/>
							<SearchOutline class="absolute left-2.5 top-2 h-4 w-4 text-gray-500" />
						</div>
					{/if}
					<LinkSortSelect />
					<LinkViewToggle />
				</div>
			</div>

			{#if shared.children.length > 0}
				<div class="mb-6 grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-2">
					{#each shared.children as child (child.id)}
						<a
							href={href(child.id)}
							class="flex items-center gap-3 rounded-lg border border-gray-200 px-3 py-2.5 transition-colors hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:hover:border-gray-700 dark:hover:bg-gray-800/60"
						>
							<FolderOutline class="h-5 w-5 shrink-0 text-gray-400" />
							<span class="min-w-0">
								<span class="block truncate text-sm font-medium">{child.name}</span>
								<span class="block truncate text-xs text-gray-500">
									{[
										`${child.bookmarkCount} link${child.bookmarkCount === 1 ? '' : 's'}`,
										child.childCount > 0 ? `${child.childCount} collection${child.childCount === 1 ? '' : 's'}` : '',
									]
										.filter(Boolean)
										.join(' · ')}
								</span>
							</span>
						</a>
					{/each}
				</div>
			{/if}

			<div class="mb-4"><TagFilter links={shared.bookmarks} bind:selected={selectedTags} /></div>

			{#if bookmarks.length > 0}
				<div class={linkListClass(preferences.linkView)}>
					{#each bookmarks as bookmark (bookmark.id)}
						<BookmarkCard {bookmark} publicView />
					{/each}
				</div>
			{:else if filter.trim() || selectedTags.length > 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">No links match the filter.</p>
			{:else if shared.children.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">No links here yet.</p>
			{/if}
		{/if}
	</main>
</div>
