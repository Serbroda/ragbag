<script lang="ts">
	import { BookmarkApi, CollectionApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, CollectionDto, SpacePermissionEnum } from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import LinkViewToggle from '../components/LinkViewToggle.svelte';
	import { linkListClass, preferences } from '../stores/preferences.svelte';
	import LinkFormFields from '../components/LinkFormFields.svelte';
	import PinToDashboardModal from '../components/PinToDashboardModal.svelte';
	import CollectionAccessModal from '../components/CollectionAccessModal.svelte';
	import ImportCollectionModal from '../components/ImportCollectionModal.svelte';
	import { DotsVerticalOutline, LockOutline } from 'flowbite-svelte-icons';
	import { route, navigate } from '../router';
	import { Button, Input, Alert, Modal, Label, Select, Textarea } from 'flowbite-svelte';
	import { collectionsToTree, flattenTree, findNode, collectSubtreeIds } from '../components/tree';

	const bookmarkApi = new BookmarkApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());
	type SpacePermission = SpacePermissionEnum;

	const reloadSidebar = getContext<() => void>('reload-collections');
	const hasSpacePermission = getContext<((permission: SpacePermission) => boolean) | null>(
		'has-space-permission',
	);
	const canCreateBookmarks = $derived(hasSpacePermission?.('CREATE_BOOKMARKS') ?? false);
	const canEditBookmarks = $derived(hasSpacePermission?.('EDIT_BOOKMARKS') ?? false);
	const canDeleteBookmarks = $derived(hasSpacePermission?.('DELETE_BOOKMARKS') ?? false);
	const canEditCollections = $derived(hasSpacePermission?.('EDIT_COLLECTIONS') ?? false);
	const canDeleteCollections = $derived(hasSpacePermission?.('DELETE_COLLECTIONS') ?? false);
	const canImportExport = $derived(hasSpacePermission?.('IMPORT_EXPORT') ?? false);

	let collection = $state<CollectionDto | null>(null);
	let bookmarks = $state<BookmarkDto[]>([]);
	let loading = $state(true);
	let error = $state('');
	/** Errors of individual actions; unlike `error` they don't replace the page. */
	let actionError = $state('');
	let modalError = $state('');
	let bookmarkToDelete = $state<BookmarkDto | null>(null);
	let bookmarkToPin = $state<BookmarkDto | null>(null);

	// Add bookmark form
	let showAddModal = $state(false);
	let newUrl = $state('');
	let newTitle = $state('');
	let newDescription = $state('');
	let newTags = $state('');
	let saving = $state(false);

	// Edit bookmark
	let editingBookmark = $state<BookmarkDto | null>(null);
	let editUrl = $state('');
	let editTitle = $state('');
	let editDescription = $state('');
	let editTags = $state('');
	let openMenuId = $state<string | null>(null);
	let openHeaderMenu = $state(false);

	// Edit collection
	let showEditCollectionModal = $state(false);
	let editCollectionName = $state('');
	let editCollectionParentId = $state('');
	let editCollectionError = $state('');
	let editCollectionSaving = $state(false);
	let allCollectionOptions = $state<{ id: string; label: string; depth: number }[]>([]);

	let showAccessModal = $state(false);
	let showImportModal = $state(false);

	async function exportCollection() {
		if (!collection) return;
		actionError = '';
		try {
			const file = await collectionApi.exportCollection({ collectionId });
			const blob = new Blob([JSON.stringify(file, null, 2)], { type: 'application/json' });
			const slug = collection.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'collection';
			const link = document.createElement('a');
			link.href = URL.createObjectURL(blob);
			link.download = `ragbag-${slug}-${new Date().toISOString().slice(0, 10)}.json`;
			link.click();
			URL.revokeObjectURL(link.href);
		} catch (err) {
			actionError = await errorMessage(err, 'Export failed.');
		}
	}

	// Delete collection
	let showDeleteCollectionModal = $state(false);
	let deleteCollectionError = $state('');
	let deleteCollectionSaving = $state(false);

	const collectionId = $derived(route.params.collectionId ?? '');
	const spaceId = $derived(route.params.spaceId ?? '');

	$effect(() => {
		if (collectionId) {
			loadData(collectionId);
		}
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
		modalError = '';
		try {
			const created = await bookmarkApi.createBookmark({
				collectionId,
				createBookmarkDto: {
					url: newUrl,
					title: newTitle || undefined,
					description: newDescription || undefined,
					tags: parseTags(newTags),
				},
			});
			bookmarks = [created, ...bookmarks];
			newUrl = '';
			newTitle = '';
			newDescription = '';
			newTags = '';
			showAddModal = false;
		} catch (err) {
			modalError = await errorMessage(err, 'Failed to add link.');
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
		editTags = bookmark.tags?.join(', ') ?? '';
		modalError = '';
	}

	async function saveEdit() {
		if (!canEditBookmarks) return;
		if (!editingBookmark || !editUrl.trim()) return;
		saving = true;
		modalError = '';
		try {
			const updated = await bookmarkApi.updateBookmark({
				bookmarkId: editingBookmark.id,
				updateBookmarkDto: {
					url: editUrl,
					title: editTitle || undefined,
					description: editDescription || undefined,
					tags: parseTags(editTags),
				},
			});
			bookmarks = bookmarks.map((b) => (b.id === updated.id ? updated : b));
			editingBookmark = null;
		} catch (err) {
			modalError = await errorMessage(err, 'Failed to update link.');
		} finally {
			saving = false;
		}
	}

	async function deleteBookmark(bookmark: BookmarkDto) {
		if (!canDeleteBookmarks) return;
		actionError = '';
		try {
			await bookmarkApi.deleteBookmark({ bookmarkId: bookmark.id });
			bookmarks = bookmarks.filter((b) => b.id !== bookmark.id);
		} catch (err) {
			actionError = await errorMessage(err, 'Failed to delete link.');
		} finally {
			bookmarkToDelete = null;
		}
	}

	/** Shift-click deletes without confirmation, like deleting messages in Discord. */
	function requestDelete(bookmark: BookmarkDto, event: MouseEvent) {
		if (event.shiftKey) deleteBookmark(bookmark);
		else bookmarkToDelete = bookmark;
	}

	function openAddModal() {
		newUrl = '';
		newTitle = '';
		newDescription = '';
		newTags = '';
		modalError = '';
		showAddModal = true;
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
		} catch (err) {
			editCollectionError = await errorMessage(err, 'Failed to update collection.');
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
		} catch (err) {
			deleteCollectionError = await errorMessage(err, 'Failed to delete collection.');
		} finally {
			deleteCollectionSaving = false;
		}
	}

	function parseTags(value: string): string[] {
		return value
			.split(',')
			.map((tag) => tag.trim())
			.filter((tag) => tag.length > 0);
	}

	function toggleMenu(bookmarkId: string, event: MouseEvent) {
		event.stopPropagation();
		openHeaderMenu = false;
		openMenuId = openMenuId === bookmarkId ? null : bookmarkId;
	}
</script>

<svelte:window onclick={() => ((openMenuId = null), (openHeaderMenu = false))} />

<div>
	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if error}
		<Alert color="red" class="mb-4">{error}</Alert>
	{:else if collection}
		<!-- Header -->
		<div class="mb-6 flex items-center justify-between gap-3">
			<div class="min-w-0">
				<h1 class="flex items-center gap-2 text-2xl font-bold tracking-tight">
					<span class="truncate">{collection.name}</span>
					{#if collection.restricted}
						<button
							type="button"
							class="inline-flex shrink-0 items-center gap-1 rounded-md bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-500/30 hover:bg-amber-500/25 dark:text-amber-300"
							title="Only selected members can see this collection"
							onclick={() => (showAccessModal = true)}
						>
							<LockOutline class="h-3.5 w-3.5" /> Private
						</button>
					{/if}
				</h1>
				{#if collection.description}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{collection.description}</p>
				{/if}
			</div>
			<div class="flex shrink-0 items-center gap-2">
				<LinkViewToggle />
				{#if canCreateBookmarks}
					<Button size="sm" onclick={openAddModal}>+ Add Link</Button>
				{/if}
				<div class="relative">
						<Button
							size="sm"
							color="light"
							onclick={(e: MouseEvent) => {
								e.stopPropagation();
								openMenuId = null;
								openHeaderMenu = !openHeaderMenu;
							}}
							aria-label="Collection actions"
						>
							&#x22EE;
						</Button>
						{#if openHeaderMenu}
							<div
								class="absolute right-0 top-10 z-20 w-44 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg dark:border-gray-700 dark:bg-gray-800"
							>
								<button
									class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
									onclick={() => {
										showAccessModal = true;
										openHeaderMenu = false;
									}}
								>
									Who can see this…
								</button>
								{#if canEditCollections}
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											openEditCollectionModal();
											openHeaderMenu = false;
										}}
									>
										Edit Collection
									</button>
								{/if}
								{#if canImportExport}
									<div class="my-1 h-px bg-gray-100 dark:bg-gray-700"></div>
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											exportCollection();
											openHeaderMenu = false;
										}}
									>
										Export as JSON
									</button>
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											showImportModal = true;
											openHeaderMenu = false;
										}}
									>
										Import JSON…
									</button>
								{/if}
								{#if canDeleteCollections}
									<div class="my-1 h-px bg-gray-100 dark:bg-gray-700"></div>
									<button
										class="w-full px-3 py-1.5 text-left text-red-600 hover:bg-gray-100 dark:text-red-400 dark:hover:bg-gray-700"
										onclick={() => {
											openDeleteCollectionModal();
											openHeaderMenu = false;
										}}
									>
										Delete Collection
									</button>
								{/if}
							</div>
						{/if}
					</div>
			</div>
		</div>

		{#if actionError}
			<Alert color="red" dismissable class="mb-4" onclose={() => (actionError = '')}>{actionError}</Alert>
		{/if}

		<!-- Bookmark list -->
		{#if bookmarks.length === 0}
			<div
				class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-12 dark:border-gray-600"
			>
				<p class="mb-2 text-gray-500 dark:text-gray-400">No links yet</p>
				{#if canCreateBookmarks}
					<Button size="sm" color="light" onclick={openAddModal}
						>Add your first link</Button
					>
				{/if}
			</div>
		{:else}
			<div class={linkListClass(preferences.linkView)}>
				{#each bookmarks as bookmark (bookmark.id)}
					<BookmarkCard {bookmark}>
						{#snippet actions()}
							<button
								type="button"
								class="rounded-md p-1 text-gray-400 opacity-60 transition hover:bg-gray-100 hover:text-gray-700 group-hover:opacity-100 dark:hover:bg-gray-800 dark:hover:text-gray-200"
								onclick={(e: MouseEvent) => toggleMenu(bookmark.id, e)}
								aria-label="Link actions"
							>
								<DotsVerticalOutline class="h-4 w-4" />
							</button>
							{#if openMenuId === bookmark.id}
								<div
									class="absolute right-0 z-20 w-44 rounded-md {preferences.linkView === 'list'
										? 'top-8'
										: 'bottom-9'} border border-gray-200 bg-white py-1 text-sm shadow-lg dark:border-gray-700 dark:bg-gray-800"
								>
									<button
										class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
										onclick={() => {
											bookmarkToPin = bookmark;
											openMenuId = null;
										}}
									>
										Add to dashboard
									</button>
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
											title="Shift-click to delete without confirmation"
											onclick={(e: MouseEvent) => {
												requestDelete(bookmark, e);
												openMenuId = null;
											}}
										>
											Delete
										</button>
									{/if}
								</div>
							{/if}
						{/snippet}
					</BookmarkCard>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<!-- Add Bookmark Modal -->
<Modal title="Add Link" bind:open={showAddModal} size="sm" autoclose={false}>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			addBookmark();
		}}
		class="flex flex-col gap-4"
	>
		<LinkFormFields
			{collectionId}
			idPrefix="add-link"
			bind:url={newUrl}
			bind:title={newTitle}
			bind:description={newDescription}
			bind:tags={newTags}
		/>
		{#if modalError}
			<p class="text-sm text-red-500 dark:text-red-400">{modalError}</p>
		{/if}
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
		<LinkFormFields
			{collectionId}
			idPrefix="edit-link"
			bind:url={editUrl}
			bind:title={editTitle}
			bind:description={editDescription}
			bind:tags={editTags}
		/>
		{#if modalError}
			<p class="text-sm text-red-500 dark:text-red-400">{modalError}</p>
		{/if}
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
			<Input id="edit-col-name" type="text" bind:value={editCollectionName} required maxlength={80} />
		</div>
		<div>
			<Label for="edit-col-parent" class="mb-2">Parent Collection</Label>
			<Select id="edit-col-parent" placeholder="" bind:value={editCollectionParentId}>
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

<PinToDashboardModal bind:bookmark={bookmarkToPin} {spaceId} />

{#if collection && canImportExport}
	<ImportCollectionModal
		bind:open={showImportModal}
		{collectionId}
		collectionName={collection.name}
		ondone={() => {
			loadData(collectionId);
			reloadSidebar();
		}}
	/>
{/if}

{#if collection}
	<CollectionAccessModal
		bind:open={showAccessModal}
		{collectionId}
		collectionName={collection.name}
		{spaceId}
		onchanged={() => {
			loadData(collectionId);
			reloadSidebar();
		}}
	/>
{/if}

<!-- Delete Bookmark Modal -->
<Modal title="Delete link" open={bookmarkToDelete !== null} size="sm" onclose={() => (bookmarkToDelete = null)}>
	<p class="text-gray-600 dark:text-gray-400">
		Delete <strong>{bookmarkToDelete?.title || bookmarkToDelete?.url}</strong>?
	</p>
	<p class="text-xs text-gray-500 dark:text-gray-400">Tip: hold Shift while clicking Delete to skip this dialog.</p>
	<div class="flex justify-end gap-2">
		<Button color="light" onclick={() => (bookmarkToDelete = null)}>Cancel</Button>
		<Button color="red" onclick={() => bookmarkToDelete && deleteBookmark(bookmarkToDelete)}>Delete</Button>
	</div>
</Modal>
