<script lang="ts">
	import { BookmarkApi, CollectionApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, CollectionDto } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { apiConfig } from '../api/client';
	import { route, navigate } from '../router';
	import { Card, Button, Input, Alert, Modal, Label, Select, Textarea } from 'flowbite-svelte';
	import { collectionsToTree, flattenTree, findNode, collectSubtreeIds } from '../components/tree';

	const bookmarkApi = new BookmarkApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());
	type SpacePermission =
		| 'READ'
		| 'EDIT_SPACE'
		| 'CREATE_COLLECTIONS'
		| 'EDIT_COLLECTIONS'
		| 'DELETE_COLLECTIONS'
		| 'CREATE_BOOKMARKS'
		| 'EDIT_BOOKMARKS'
		| 'DELETE_BOOKMARKS';

	const reloadSidebar = getContext<() => void>('reload-collections');
	const hasSpacePermission = getContext<((permission: SpacePermission) => boolean) | null>(
		'has-space-permission',
	);
	const canCreateBookmarks = $derived(hasSpacePermission?.('CREATE_BOOKMARKS') ?? false);
	const canEditBookmarks = $derived(hasSpacePermission?.('EDIT_BOOKMARKS') ?? false);
	const canDeleteBookmarks = $derived(hasSpacePermission?.('DELETE_BOOKMARKS') ?? false);
	const canEditCollections = $derived(hasSpacePermission?.('EDIT_COLLECTIONS') ?? false);
	const canDeleteCollections = $derived(hasSpacePermission?.('DELETE_COLLECTIONS') ?? false);

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
	let openMenuId = $state<string | null>(null);
	let openHeaderMenu = $state(false);

	// Edit collection
	let showEditCollectionModal = $state(false);
	let editCollectionName = $state('');
	let editCollectionParentId = $state('');
	let editCollectionError = $state('');
	let editCollectionSaving = $state(false);
	let allCollectionOptions = $state<{ id: string; label: string; depth: number }[]>([]);

	// Delete collection
	let showDeleteCollectionModal = $state(false);
	let deleteCollectionError = $state('');
	let deleteCollectionSaving = $state(false);

	const collectionId = $derived(
		route.getParams('/space/:spaceId/collections/:collectionId').collectionId,
	);

	const spaceId = $derived(route.getParams('/space/:spaceId/collections/:collectionId').spaceId);

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
		if (!canCreateBookmarks) return;
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
		if (!canEditBookmarks) return;
		editingBookmark = bookmark;
		editUrl = bookmark.url;
		editTitle = bookmark.title ?? '';
		editDescription = bookmark.description ?? '';
	}

	async function saveEdit() {
		if (!canEditBookmarks) return;
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
		if (!canDeleteBookmarks) return;
		try {
			await bookmarkApi.deleteBookmark({ bookmarkId: bookmark.id });
			bookmarks = bookmarks.filter((b) => b.id !== bookmark.id);
		} catch {
			error = 'Failed to delete bookmark.';
		}
	}

	// --- Collection Edit/Delete ---

	async function openEditCollectionModal() {
		if (!canEditCollections) return;
		if (!collection) return;
		editCollectionName = collection.name;
		editCollectionParentId = collection.parentId ?? '';
		editCollectionError = '';
		editCollectionSaving = false;

		// Fetch all collections to build the parent dropdown
		try {
			const allCollections = await collectionApi.getCollections({ spaceId });
			const tree = collectionsToTree(allCollections, spaceId);
			const flat = flattenTree(tree);
			// Exclude current collection and its descendants from the parent options
			const currentNode = findNode(tree, collectionId);
			const excludeIds = currentNode ? collectSubtreeIds(currentNode) : new Set([collectionId]);
			allCollectionOptions = flat.filter((opt) => !excludeIds.has(opt.id));
		} catch {
			allCollectionOptions = [];
		}

		showEditCollectionModal = true;
	}

	async function handleEditCollection() {
		if (!canEditCollections) return;
		if (!collection) return;
		const name = editCollectionName.trim();
		if (!name) return;
		editCollectionError = '';
		editCollectionSaving = true;

		try {
			// Update name if changed
			if (name !== collection.name) {
				await collectionApi.updateCollection({
					collectionId,
					updateCollectionDto: { name },
				});
			}

			// Move if parent changed
			const currentParentId = collection.parentId ?? '';
			if (editCollectionParentId !== currentParentId) {
				await collectionApi.moveCollection({
					collectionId,
					moveCollectionDto: {
						parentId: editCollectionParentId || undefined,
					},
				});
			}

			showEditCollectionModal = false;
			await loadData(collectionId);
			reloadSidebar();
		} catch {
			editCollectionError = 'Failed to update collection.';
		} finally {
			editCollectionSaving = false;
		}
	}

	function openDeleteCollectionModal() {
		if (!canDeleteCollections) return;
		deleteCollectionError = '';
		deleteCollectionSaving = false;
		showDeleteCollectionModal = true;
	}

	async function handleDeleteCollection() {
		if (!canDeleteCollections) return;
		deleteCollectionError = '';
		deleteCollectionSaving = true;
		try {
			await collectionApi.deleteCollection({ collectionId });
			showDeleteCollectionModal = false;
			reloadSidebar();
			navigate('/space/:spaceId', { params: { spaceId } });
		} catch {
			deleteCollectionError = 'Failed to delete collection.';
		} finally {
			deleteCollectionSaving = false;
		}
	}

	function getDomain(url: string): string {
		try {
			return new URL(url).hostname;
		} catch {
			return url;
		}
	}

	function toggleMenu(bookmarkId: string) {
		openMenuId = openMenuId === bookmarkId ? null : bookmarkId;
	}
</script>

<main class="mx-auto p-6">
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
			<div class="flex items-center gap-2">
				<div class="hidden gap-2 md:flex">
					{#if canEditCollections}
						<Button size="sm" color="light" onclick={openEditCollectionModal}>Edit</Button>
					{/if}
					{#if canDeleteCollections}
						<Button size="sm" color="red" outline onclick={openDeleteCollectionModal}>Delete</Button>
					{/if}
					{#if canCreateBookmarks}
						<Button size="sm" onclick={() => (showAddModal = true)}>+ Add Link</Button>
					{/if}
				</div>
				{#if canEditCollections || canDeleteCollections || canCreateBookmarks}
					<div class="relative md:hidden">
						<Button
							size="sm"
							color="light"
							onclick={() => (openHeaderMenu = !openHeaderMenu)}
							aria-label="Collection actions"
						>
							&#x22EE;
						</Button>
						{#if openHeaderMenu}
							<div
								class="absolute right-0 top-10 z-20 w-40 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg dark:border-gray-700 dark:bg-gray-800"
							>
								{#if canCreateBookmarks}
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											showAddModal = true;
											openHeaderMenu = false;
										}}
									>
										Add Link
									</button>
								{/if}
								{#if canEditCollections}
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											openEditCollectionModal();
											openHeaderMenu = false;
										}}
									>
										Edit
									</button>
								{/if}
								{#if canDeleteCollections}
									<button
										class="w-full px-3 py-1.5 text-left text-red-600 hover:bg-gray-100 dark:text-red-400 dark:hover:bg-gray-700"
										onclick={() => {
											openDeleteCollectionModal();
											openHeaderMenu = false;
										}}
									>
										Delete
									</button>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			</div>
		</div>

		<!-- Bookmark list -->
		{#if bookmarks.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-gray-600"
			>
				<p class="mb-2 text-gray-500 dark:text-gray-400">No links yet</p>
				{#if canCreateBookmarks}
					<Button size="sm" color="light" onclick={() => (showAddModal = true)}
						>Add your first link</Button
					>
				{/if}
			</div>
		{:else}
			<div class="flex flex-col gap-4 md:flex-row md:flex-wrap">
				{#each bookmarks as bookmark (bookmark.id)}
					<Card class="relative w-full p-4 pb-12 hover:border-gray-400 md:w-[calc(50%-0.5rem)] lg:w-[calc(33.333%-0.75rem)]">
						<div class="flex flex-col gap-2">
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
										class="block break-words font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400 line-clamp-2"
									>
										{bookmark.title || bookmark.url}
									</a>
									<p class="truncate text-xs text-gray-400 dark:text-gray-500">
										{getDomain(bookmark.url)}
									</p>

								</div>



							</div>

							{#if bookmark.description}
								<div class="text-sm text-gray-500 dark:text-gray-400 break-words line-clamp-2">
									{bookmark.description}
								</div>
							{/if}
						</div>

						{#if canEditBookmarks || canDeleteBookmarks}
							<div class="absolute bottom-3 right-3">
								<Button
									size="xs"
									color="light"
									onclick={() => toggleMenu(bookmark.id)}
									aria-label="Bookmark actions"
								>
									&#x22EE;
								</Button>
								{#if openMenuId === bookmark.id}
									<div
										class="absolute bottom-9 right-0 z-10 w-32 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg dark:border-gray-700 dark:bg-gray-800"
									>
										{#if canEditBookmarks}
											<button
												class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
												onclick={() => {
													startEdit(bookmark);
													openMenuId = null;
												}}
											>
												Edit
											</button>
										{/if}
										{#if canDeleteBookmarks}
											<button
												class="w-full px-3 py-1.5 text-left text-red-600 hover:bg-gray-100 dark:text-red-400 dark:hover:bg-gray-700"
												onclick={() => {
													deleteBookmark(bookmark);
													openMenuId = null;
												}}
											>
												Delete
											</button>
										{/if}
									</div>
								{/if}
							</div>
						{/if}
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
		<Textarea placeholder="Description (optional)" rows={4} class="w-full" bind:value={newDescription} />
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
		<Textarea placeholder="Description (optional)" rows={4} class="w-full" bind:value={editDescription} />
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (editingBookmark = null)}>Cancel</Button>
			<Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Save'}</Button>
		</div>
	</form>
</Modal>

<!-- Edit Collection Modal -->
<Modal title="Edit Collection" bind:open={showEditCollectionModal} size="sm" autoclose={false}>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleEditCollection();
		}}
		class="flex flex-col gap-4"
	>
		<div>
			<Label for="edit-col-name" class="mb-2">Name</Label>
			<Input id="edit-col-name" type="text" bind:value={editCollectionName} required />
		</div>
		<div>
			<Label for="edit-col-parent" class="mb-2">Parent Collection</Label>
			<Select id="edit-col-parent" bind:value={editCollectionParentId}>
				<option value="">None (top level)</option>
				{#each allCollectionOptions as opt (opt.id)}
					<option value={opt.id}>{'─'.repeat(opt.depth)} {opt.label}</option>
				{/each}
			</Select>
		</div>
		{#if editCollectionError}
			<p class="text-sm text-red-500 dark:text-red-400">{editCollectionError}</p>
		{/if}
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (showEditCollectionModal = false)}>Cancel</Button>
			<Button type="submit" disabled={editCollectionSaving}>
				{editCollectionSaving ? 'Saving...' : 'Save'}
			</Button>
		</div>
	</form>
</Modal>

<!-- Delete Collection Modal -->
<Modal title="Delete Collection" bind:open={showDeleteCollectionModal} size="sm" autoclose={false}>
	<p class="text-gray-600 dark:text-gray-400">
		Are you sure you want to delete <strong>{collection?.name}</strong>? This will also delete all
		bookmarks and sub-collections within it. This action cannot be undone.
	</p>
	{#if deleteCollectionError}
		<p class="mt-2 text-sm text-red-500 dark:text-red-400">{deleteCollectionError}</p>
	{/if}
	<div class="mt-4 flex justify-end gap-2">
		<Button color="light" onclick={() => (showDeleteCollectionModal = false)}>Cancel</Button>
		<Button color="red" onclick={handleDeleteCollection} disabled={deleteCollectionSaving}>
			{deleteCollectionSaving ? 'Deleting...' : 'Delete'}
		</Button>
	</div>
</Modal>
