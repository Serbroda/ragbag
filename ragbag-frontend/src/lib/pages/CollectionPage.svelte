<script lang="ts">
	import { BookmarkApi, CollectionApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, CollectionDto } from 'ragbag-frontend-sdk';
	import { apiConfig } from '../api/client';
	import { route } from '../router';
	import { Card, Button, Input, Alert, Modal } from 'flowbite-svelte';

	const bookmarkApi = new BookmarkApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());

	let collection = $state<CollectionDto | null>(null);
	let bookmarks = $state<BookmarkDto[]>([]);
	let loading = $state(true);
	let error = $state('');

	// Add bookmark form
	let showAddModal = $state(false);
	let newUrl = $state('');
	let newTitle = $state('');
	let newDescription = $state('');
	let saving = $state(false);

	// Edit bookmark
	let editingBookmark = $state<BookmarkDto | null>(null);
	let editUrl = $state('');
	let editTitle = $state('');
	let editDescription = $state('');

	const collectionId = $derived(
		route.getParams('/space/:spaceId/collections/:collectionId').collectionId,
	);

	$effect(() => {
		loadData(collectionId);
	});

	async function loadData(id: string) {
		loading = true;
		error = '';
		try {
			const [col, bm] = await Promise.all([
				collectionApi.getCollection({ collectionId: id }),
				bookmarkApi.getBookmarks({ collectionId: id }),
			]);
			collection = col;
			bookmarks = bm;
		} catch {
			error = 'Failed to load collection.';
		} finally {
			loading = false;
		}
	}

	async function addBookmark() {
		if (!newUrl.trim()) return;
		saving = true;
		try {
			const created = await bookmarkApi.createBookmark({
				collectionId,
				createBookmarkDto: {
					url: newUrl,
					title: newTitle || undefined,
					description: newDescription || undefined,
				},
			});
			bookmarks = [...bookmarks, created];
			newUrl = '';
			newTitle = '';
			newDescription = '';
			showAddModal = false;
		} catch {
			error = 'Failed to add bookmark.';
		} finally {
			saving = false;
		}
	}

	function startEdit(bookmark: BookmarkDto) {
		editingBookmark = bookmark;
		editUrl = bookmark.url;
		editTitle = bookmark.title ?? '';
		editDescription = bookmark.description ?? '';
	}

	async function saveEdit() {
		if (!editingBookmark || !editUrl.trim()) return;
		saving = true;
		try {
			const updated = await bookmarkApi.updateBookmark({
				bookmarkId: editingBookmark.id,
				updateBookmarkDto: {
					url: editUrl,
					title: editTitle || undefined,
					description: editDescription || undefined,
				},
			});
			bookmarks = bookmarks.map((b) => (b.id === updated.id ? updated : b));
			editingBookmark = null;
		} catch {
			error = 'Failed to update bookmark.';
		} finally {
			saving = false;
		}
	}

	async function deleteBookmark(bookmark: BookmarkDto) {
		try {
			await bookmarkApi.deleteBookmark({ bookmarkId: bookmark.id });
			bookmarks = bookmarks.filter((b) => b.id !== bookmark.id);
		} catch {
			error = 'Failed to delete bookmark.';
		}
	}

	function getDomain(url: string): string {
		try {
			return new URL(url).hostname;
		} catch {
			return url;
		}
	}
</script>

<main class="mx-auto max-w-4xl p-6">
	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if error}
		<Alert color="red" class="mb-4">{error}</Alert>
	{:else if collection}
		<!-- Header -->
		<div class="mb-6 flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{collection.name}</h1>
				{#if collection.description}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{collection.description}</p>
				{/if}
			</div>
			<Button size="sm" onclick={() => (showAddModal = true)}>+ Add Link</Button>
		</div>

		<!-- Bookmark list -->
		{#if bookmarks.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-gray-600"
			>
				<p class="mb-2 text-gray-500 dark:text-gray-400">No links yet</p>
				<Button size="sm" color="light" onclick={() => (showAddModal = true)}
					>Add your first link</Button
				>
			</div>
		{:else}
			<div class="space-y-2">
				{#each bookmarks as bookmark (bookmark.id)}
					<Card class="p-4">
						<div class="flex items-start gap-4">
							<!-- Favicon -->
							<img
								src="https://www.google.com/s2/favicons?domain={getDomain(bookmark.url)}&sz=32"
								alt=""
								class="mt-1 h-6 w-6 shrink-0 rounded"
							/>

							<!-- Content -->
							<div class="min-w-0 flex-1">
								<a
									href={bookmark.url}
									target="_blank"
									rel="noopener noreferrer"
									class="font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
								>
									{bookmark.title || bookmark.url}
								</a>
								<p class="truncate text-xs text-gray-400 dark:text-gray-500">
									{getDomain(bookmark.url)}
								</p>
								{#if bookmark.description}
									<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
										{bookmark.description}
									</p>
								{/if}
							</div>

							<!-- Actions -->
							<div class="flex shrink-0 gap-1">
								<Button size="xs" color="light" onclick={() => startEdit(bookmark)}>Edit</Button>
								<Button size="xs" color="red" outline onclick={() => deleteBookmark(bookmark)}
									>Delete</Button
								>
							</div>
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	{/if}
</main>

<!-- Add Bookmark Modal -->
<Modal title="Add Link" bind:open={showAddModal} size="sm" autoclose={false}>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			addBookmark();
		}}
		class="flex flex-col gap-4"
	>
		<Input type="url" placeholder="https://..." bind:value={newUrl} required />
		<Input type="text" placeholder="Title (optional)" bind:value={newTitle} />
		<Input type="text" placeholder="Description (optional)" bind:value={newDescription} />
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (showAddModal = false)}>Cancel</Button>
			<Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Add'}</Button>
		</div>
	</form>
</Modal>

<!-- Edit Bookmark Modal -->
<Modal
	title="Edit Link"
	open={!!editingBookmark}
	size="sm"
	autoclose={false}
	onclose={() => (editingBookmark = null)}
>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			saveEdit();
		}}
		class="flex flex-col gap-4"
	>
		<Input type="url" placeholder="https://..." bind:value={editUrl} required />
		<Input type="text" placeholder="Title (optional)" bind:value={editTitle} />
		<Input type="text" placeholder="Description (optional)" bind:value={editDescription} />
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (editingBookmark = null)}>Cancel</Button>
			<Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save'}</Button>
		</div>
	</form>
</Modal>
