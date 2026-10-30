<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { TrashDto } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { Alert, Button, Modal } from 'flowbite-svelte';
	import { FolderOutline, LinkOutline, TrashBinOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { route } from '../router';
	import { showToast } from '../stores/toast.svelte';
	import { getDomain, safeUrl, timeAgo } from '../utils/format';

	const spaceApi = new SpaceApi(apiConfig());
	const reloadSidebar = getContext<(() => void) | undefined>('reload-collections');
	const hasSpacePermission = getContext<((permission: string) => boolean) | undefined>('has-space-permission');
	const canRestoreLinks = $derived(hasSpacePermission?.('DELETE_BOOKMARKS') ?? false);
	const canRestoreCollections = $derived(hasSpacePermission?.('DELETE_COLLECTIONS') ?? false);

	const spaceId = $derived(route.params.spaceId ?? '');
	let trash = $state<TrashDto | null>(null);
	let error = $state('');
	let busy = $state(false);
	let confirmEmpty = $state(false);

	$effect(() => {
		if (spaceId) load(spaceId);
	});

	async function load(id: string) {
		error = '';
		try {
			trash = await spaceApi.getTrash({ spaceId: id });
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the trash.');
		}
	}

	async function act(kind: 'restore' | 'purge', selection: { collectionIds?: string[]; bookmarkIds?: string[]; all?: boolean }) {
		busy = true;
		try {
			if (kind === 'restore') await spaceApi.restoreFromTrash({ spaceId, trashSelectionDto: selection });
			else await spaceApi.purgeTrash({ spaceId, trashSelectionDto: selection });
			showToast(kind === 'restore' ? 'Restored' : 'Deleted for good');
			reloadSidebar?.();
			await load(spaceId);
		} catch (err) {
			showToast(await errorMessage(err, 'That did not work.'));
		} finally {
			busy = false;
			confirmEmpty = false;
		}
	}

	const empty = $derived(!!trash && trash.collections.length === 0 && trash.bookmarks.length === 0);
</script>

<div>
	<div class="mb-6 flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-2xl font-bold tracking-tight">Trash</h1>
			<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
				Deleted links and collections stay here for {trash?.retentionDays ?? 30} days, then they are gone for good.
			</p>
		</div>
		{#if trash && !empty && (canRestoreLinks || canRestoreCollections)}
			<Button color="light" size="sm" disabled={busy} onclick={() => (confirmEmpty = true)}>
				<TrashBinOutline class="me-1.5 h-4 w-4" /> Empty trash
			</Button>
		{/if}
	</div>

	{#if error}
		<Alert color="red">{error}</Alert>
	{:else if !trash}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if empty}
		<div class="rounded-lg border-2 border-dashed border-gray-300 py-12 text-center text-gray-500 dark:border-gray-700 dark:text-gray-400">
			The trash is empty.
		</div>
	{:else}
		<ul class="divide-y divide-gray-200 rounded-xl border border-gray-200 dark:divide-gray-800 dark:border-gray-800">
			{#each trash.collections as collection (collection.id)}
				<li class="flex flex-wrap items-center gap-3 px-4 py-3">
					<FolderOutline class="h-5 w-5 shrink-0 text-gray-400" />
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-medium">{collection.name}</p>
						<p class="truncate text-xs text-gray-500">
							Collection{collection.parentName ? ` in ${collection.parentName}` : ''} · {collection.bookmarkCount}
							link{collection.bookmarkCount === 1 ? '' : 's'}{collection.childCount
								? ` · ${collection.childCount} collection${collection.childCount === 1 ? '' : 's'}`
								: ''} · deleted {timeAgo(collection.deletedAt)}{collection.deletedBy ? ` by ${collection.deletedBy}` : ''}
						</p>
					</div>
					{#if canRestoreCollections}
						<Button size="xs" color="light" disabled={busy} onclick={() => act('restore', { collectionIds: [collection.id] })}>
							Restore
						</Button>
						<Button size="xs" color="alternative" disabled={busy} onclick={() => act('purge', { collectionIds: [collection.id] })}>
							Delete for good
						</Button>
					{/if}
				</li>
			{/each}
			{#each trash.bookmarks as bookmark (bookmark.id)}
				{@const favicon = safeUrl(bookmark.favicon)}
				<li class="flex flex-wrap items-center gap-3 px-4 py-3">
					{#if favicon}
						<img src={favicon} alt="" class="h-5 w-5 shrink-0 rounded-sm" referrerpolicy="no-referrer" />
					{:else}
						<LinkOutline class="h-5 w-5 shrink-0 text-gray-400" />
					{/if}
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-medium">{bookmark.title || bookmark.url}</p>
						<p class="truncate text-xs text-gray-500">
							{getDomain(bookmark.url)} · from {bookmark.collectionName} · deleted {timeAgo(bookmark.deletedAt)}{bookmark.deletedBy
								? ` by ${bookmark.deletedBy}`
								: ''}
						</p>
					</div>
					{#if canRestoreLinks}
						<Button size="xs" color="light" disabled={busy} onclick={() => act('restore', { bookmarkIds: [bookmark.id] })}>
							Restore
						</Button>
						<Button size="xs" color="alternative" disabled={busy} onclick={() => act('purge', { bookmarkIds: [bookmark.id] })}>
							Delete for good
						</Button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Modal title="Empty trash" bind:open={confirmEmpty} size="sm" autoclose={false}>
	<p class="text-gray-600 dark:text-gray-300">
		Delete everything in the trash for good? This cannot be undone.
	</p>
	<div class="mt-4 flex justify-end gap-2">
		<Button color="light" onclick={() => (confirmEmpty = false)}>Cancel</Button>
		<Button color="red" disabled={busy} onclick={() => act('purge', { all: true })}>Delete for good</Button>
	</div>
</Modal>
