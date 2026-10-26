<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import { Button, Checkbox, Input, Modal } from 'flowbite-svelte';
	import { LinkOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage, searchBookmarks } from '../api/client';
	import { getDomain, safeUrl } from '../utils/format';

	let {
		open = $bindable(false),
		spaceId,
		excludeIds = new Set<string>(),
		onpick,
	}: {
		open?: boolean;
		spaceId: string;
		/** Links already in the section; shown but not selectable. */
		excludeIds?: Set<string>;
		onpick: (bookmarks: BookmarkDto[]) => void;
	} = $props();

	let query = $state('');
	let results = $state<BookmarkDto[]>([]);
	let selected = $state<Map<string, BookmarkDto>>(new Map());
	let loading = $state(false);
	let error = $state('');
	let timer: ReturnType<typeof setTimeout> | undefined;

	$effect(() => {
		if (open) {
			query = '';
			selected = new Map();
			load('');
		}
	});

	async function load(q: string) {
		loading = true;
		error = '';
		try {
			const found = q.trim()
				? await searchBookmarks(spaceId, q.trim())
				: await new BookmarkApi(apiConfig()).getRecentBookmarks({ spaceId, limit: 100 });
			if (q === query) results = found;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load links.');
		} finally {
			loading = false;
		}
	}

	function oninput() {
		clearTimeout(timer);
		timer = setTimeout(() => load(query), 250);
	}

	function toggle(bookmark: BookmarkDto) {
		const next = new Map(selected);
		if (next.has(bookmark.id)) next.delete(bookmark.id);
		else next.set(bookmark.id, bookmark);
		selected = next;
	}

	function confirm() {
		onpick([...selected.values()]);
		open = false;
	}
</script>

<Modal bind:open size="md" title="Add links">
	<Input type="search" placeholder="Search links in this space" bind:value={query} {oninput} />
	<ul class="-mx-2 max-h-96 overflow-y-auto">
		{#each results as bookmark (bookmark.id)}
			{@const already = excludeIds.has(bookmark.id)}
			{@const favicon = safeUrl(bookmark.favicon)}
			<li>
				<label
					class="flex cursor-pointer items-center gap-3 rounded-lg px-2 py-2 hover:bg-gray-100 dark:hover:bg-gray-700/60 {already
						? 'cursor-default opacity-50'
						: ''}"
				>
					<Checkbox checked={already || selected.has(bookmark.id)} disabled={already} onchange={() => toggle(bookmark)} />
					<span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-gray-100 dark:bg-gray-800">
						{#if favicon}
							<img src={favicon} alt="" class="h-5 w-5 rounded" referrerpolicy="no-referrer" />
						{:else}
							<LinkOutline class="h-4 w-4 text-gray-400" />
						{/if}
					</span>
					<span class="min-w-0 flex-1">
						<span class="block truncate text-sm font-medium text-gray-900 dark:text-white">
							{bookmark.title || bookmark.url}
						</span>
						<span class="block truncate text-xs text-gray-500">
							{getDomain(bookmark.url)}{already ? ' · already added' : ''}
						</span>
					</span>
				</label>
			</li>
		{:else}
			<li class="px-2 py-8 text-center text-sm text-gray-500">
				{loading ? 'Loading…' : query ? 'No links match your search.' : 'This space has no links yet.'}
			</li>
		{/each}
	</ul>
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			<Button color="alternative" onclick={() => (open = false)}>Cancel</Button>
			<Button disabled={selected.size === 0} onclick={confirm}>
				Add {selected.size > 0 ? selected.size : ''} link{selected.size === 1 ? '' : 's'}
			</Button>
		</div>
	{/snippet}
</Modal>
