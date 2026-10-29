<script lang="ts">
	import { BookmarkApi, CollectionApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, CollectionDto, SpacePermissionEnum } from 'ragbag-frontend-sdk';
	import { getContext, tick } from 'svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import LinkViewToggle from '../components/LinkViewToggle.svelte';
	import { linkListClass, preferences } from '../stores/preferences.svelte';
	import LinkFormFields from '../components/LinkFormFields.svelte';
	import PinToDashboardModal from '../components/PinToDashboardModal.svelte';
	import CollectionAccessModal from '../components/CollectionAccessModal.svelte';
	import ImportCollectionModal from '../components/ImportCollectionModal.svelte';
	import { DotsVerticalOutline, FolderOutline, LockOutline } from 'flowbite-svelte-icons';
	import { route, navigate } from '../router';
	import { Button, Input, Alert, Modal, Label, Select, Textarea } from 'flowbite-svelte';
	import {
		collectionsToTree,
		flattenTree,
		findNode,
		collectSubtreeIds,
		type TreeNode,
	} from '../components/tree';
	import { bookmarkDrag } from '../stores/bookmarkDrag.svelte';
	import { pendingCollectionAction } from '../stores/collectionAction.svelte';
	import { showToast } from '../stores/toast.svelte';

	const bookmarkApi = new BookmarkApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());
	type SpacePermission = SpacePermissionEnum;

	const reloadSidebar = getContext<() => void>('reload-collections');
	const reloadSpaces = getContext<(() => Promise<void>) | undefined>('reload-spaces');
	const hasSpacePermission = getContext<((permission: SpacePermission) => boolean) | null>(
		'has-space-permission',
	);
	const canCreateBookmarks = $derived(hasSpacePermission?.('CREATE_BOOKMARKS') ?? false);
	const canEditBookmarks = $derived(hasSpacePermission?.('EDIT_BOOKMARKS') ?? false);
	const canDeleteBookmarks = $derived(hasSpacePermission?.('DELETE_BOOKMARKS') ?? false);
	const canEditCollections = $derived(hasSpacePermission?.('EDIT_COLLECTIONS') ?? false);
	const canDeleteCollections = $derived(hasSpacePermission?.('DELETE_COLLECTIONS') ?? false);
	const canImportExport = $derived(hasSpacePermission?.('IMPORT_EXPORT') ?? false);
	const privateAncestor = getContext<((collectionId: string) => string | undefined) | undefined>('private-ancestor');
	const collectionTree = getContext<(() => TreeNode[]) | undefined>('collection-tree');

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
	let addLinkForm = $state<LinkFormFields | null>(null);
	let bookmarkToMove = $state<BookmarkDto | null>(null);
	let moveTargetId = $state('');
	let openMenuId = $state<string | null>(null);
	let openHeaderMenu = $state(false);

	// Edit collection
	let showEditCollectionModal = $state(false);
	let editCollectionName = $state('');
	let editCollectionDescription = $state('');
	let editCollectionParentId = $state('');
	let editCollectionError = $state('');
	let editCollectionSaving = $state(false);
	let allCollectionOptions = $state<{ id: string; label: string; depth: number }[]>([]);

	let showAccessModal = $state(false);
	/** Sub-collection tile a dragged link is over. */
	let dropTargetId = $state<string | null>(null);
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
	/** Name of the nearest private collection above this one: its members are the only ones who see this one. */
	const privateVia = $derived(collectionId ? privateAncestor?.(collectionId) : undefined);

	const tree = $derived(collectionTree?.() ?? []);
	const subcollections = $derived(findNode(tree, collectionId)?.children ?? []);
	/** Every other collection of the space, for "Move to…". */
	const moveTargets = $derived(flattenTree(tree).filter((option) => option.id !== collectionId));

	// Dialogs requested from the sidebar's context menu
	$effect(() => {
		const pending = pendingCollectionAction.current;
		if (!pending || pending.collectionId !== collectionId || collection?.id !== collectionId) return;
		pendingCollectionAction.current = null;
		if (pending.action === 'edit') openEditCollectionModal();
		else if (pending.action === 'delete') openDeleteCollectionModal();
		else showAccessModal = true;
	});

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
			// The links keep their "New" badge until the page is left; the counters go away right now
			if (bm.some((b) => b.isNew)) markSeen(id);
		} catch {
			error = 'Failed to load collection.';
		} finally {
			loading = false;
		}
	}

	async function markSeen(id: string) {
		try {
			await collectionApi.markCollectionSeen({ collectionId: id });
			reloadSidebar();
			await reloadSpaces?.();
		} catch {
			// Only the counters stay; the next visit tries again
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
			reloadSidebar();
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
			reloadSidebar();
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

	/** Moves a link to another collection of the space; the toast offers to move it back. */
	async function moveBookmark(bookmark: BookmarkDto, targetId: string, undo = false) {
		if (!canEditBookmarks || targetId === bookmark.collectionId) return;
		actionError = '';
		try {
			const moved = await bookmarkApi.updateBookmark({
				bookmarkId: bookmark.id,
				updateBookmarkDto: {
					url: bookmark.url,
					title: bookmark.title,
					description: bookmark.description,
					tags: bookmark.tags,
					collectionId: targetId,
				},
			});
			if (targetId === collectionId) {
				bookmarks = [moved, ...bookmarks.filter((b) => b.id !== moved.id)].sort(
					(a, b) => (b.createdAt?.getTime() ?? 0) - (a.createdAt?.getTime() ?? 0),
				);
			} else {
				bookmarks = bookmarks.filter((b) => b.id !== bookmark.id);
			}
			reloadSidebar();
			if (!undo) {
				const name = findNode(tree, targetId)?.label ?? 'collection';
				const from = bookmark.collectionId!;
				showToast(`Moved to ${name}`, { label: 'Undo', run: () => moveBookmark(moved, from, true) });
			}
		} catch (err) {
			actionError = await errorMessage(err, 'Failed to move link.');
		}
	}

	function startBookmarkDrag(bookmark: BookmarkDto, event: DragEvent) {
		if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copyMove';
		bookmarkDrag.current = {
			collectionId: bookmark.collectionId ?? collectionId,
			drop: (targetId) => moveBookmark(bookmark, targetId),
		};
	}

	function endBookmarkDrag() {
		bookmarkDrag.current = null;
	}

	/** Like dropping a link into a chat: pasting a URL anywhere on the page (outside fields) starts adding it. */
	async function handlePaste(event: ClipboardEvent) {
		if (!canCreateBookmarks || !collection) return;
		const target = event.target instanceof Element ? event.target : null;
		if (target?.closest('input, textarea, select, [contenteditable]')) return;
		if (document.querySelector('dialog[open], [role=dialog]')) return;
		const text = event.clipboardData?.getData('text/plain').trim() ?? '';
		if (!/^https?:\/\/\S+$/i.test(text)) return;
		event.preventDefault();
		openAddModal();
		newUrl = text;
		await tick();
		addLinkForm?.fetchMetadata();
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
		editCollectionDescription = collection.description ?? '';
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
			const description = editCollectionDescription.trim();
			if (name !== collection.name || description !== (collection.description ?? '')) {
				await collectionApi.updateCollection({
					collectionId,
					updateCollectionDto: { name, description },
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

<svelte:window onclick={() => ((openMenuId = null), (openHeaderMenu = false))} onpaste={handlePaste} />

<div>
	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if error}
		<Alert color="red" class="mb-4">{error}</Alert>
	{:else if collection}
		<!-- Header -->
		<!-- Stacked on phones so the title keeps the full width -->
		<div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<div class="min-w-0">
				<h1 class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xl font-bold tracking-tight sm:text-2xl">
					<span class="min-w-0 break-words">{collection.name}</span>
					{#if collection.restricted || privateVia}
						<button
							type="button"
							class="inline-flex max-w-full shrink-0 items-center gap-1 rounded-md bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-500/30 hover:bg-amber-500/25 dark:text-amber-300"
							title={collection.restricted
								? 'Only selected members can see this collection'
								: `Inside the private collection “${privateVia}”: only its members can see this one`}
							onclick={() => (showAccessModal = true)}
						>
							<LockOutline class="h-3.5 w-3.5 shrink-0" />
							<span class="truncate">{collection.restricted ? 'Private' : `Private via ${privateVia}`}</span>
						</button>
					{/if}
				</h1>
				{#if collection.description}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{collection.description}</p>
				{/if}
			</div>
			<div class="flex shrink-0 items-center gap-2">
				<div class="me-auto sm:me-0"><LinkViewToggle /></div>
				{#if canCreateBookmarks}
					<Button size="sm" onclick={openAddModal} title="Or paste a URL anywhere on this page">+ Add Link</Button>
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
							<DotsVerticalOutline class="h-4 w-4" />
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

		{#if subcollections.length > 0}
			<div class="mb-6 grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] gap-2">
				{#each subcollections as sub (sub.id)}
					{@const Icon = sub.restricted ? LockOutline : FolderOutline}
					{@const childCount = sub.children?.length ?? 0}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<a
						href={sub.href}
						class="flex items-center gap-3 rounded-lg border px-3 py-2.5 transition-colors {dropTargetId === sub.id
							? 'border-primary-500 bg-primary-500/10'
							: 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:hover:border-gray-700 dark:hover:bg-gray-800/60'}"
						ondragover={(e) => {
							if (!bookmarkDrag.current) return;
							e.preventDefault();
							dropTargetId = sub.id!;
						}}
						ondragleave={() => (dropTargetId = null)}
						ondrop={(e) => {
							e.preventDefault();
							dropTargetId = null;
							bookmarkDrag.current?.drop(sub.id!);
						}}
					>
						<Icon class="h-5 w-5 shrink-0 text-gray-400" />
						<span class="min-w-0">
							<span class="block truncate text-sm font-medium">{sub.label}</span>
							<span class="block truncate text-xs text-gray-500">
								{[
									`${sub.bookmarkCount ?? 0} link${sub.bookmarkCount === 1 ? '' : 's'}`,
									childCount > 0 ? `${childCount} collection${childCount === 1 ? '' : 's'}` : '',
								]
									.filter(Boolean)
									.join(' · ')}
							</span>
						</span>
					</a>
				{/each}
			</div>
		{/if}

		<!-- Bookmark list -->
		{#if bookmarks.length === 0}
			{#if subcollections.length === 0}
				<div
					class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 px-4 py-12 text-center dark:border-gray-600"
				>
					<p class="mb-2 text-gray-500 dark:text-gray-400">No links yet</p>
					{#if canCreateBookmarks}
						<Button size="sm" color="light" onclick={openAddModal}>Add your first link</Button>
						<p class="mt-3 text-xs text-gray-400 dark:text-gray-500">Tip: paste a URL anywhere on this page</p>
					{/if}
				</div>
			{/if}
		{:else}
			<div class={linkListClass(preferences.linkView)}>
				{#each bookmarks as bookmark (bookmark.id)}
					<BookmarkCard
						{bookmark}
						ondragstart={canEditBookmarks ? (e) => startBookmarkDrag(bookmark, e) : undefined}
						ondragend={endBookmarkDrag}
					>
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
										<button
											class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
											title="Or drag the link onto a collection in the sidebar"
											onclick={() => {
												bookmarkToMove = bookmark;
												moveTargetId = '';
												openMenuId = null;
											}}
										>
											Move to…
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
			bind:this={addLinkForm}
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
			<Label for="edit-col-description" class="mb-2">Description</Label>
			<Textarea
				id="edit-col-description"
				rows={2}
				class="w-full"
				maxlength={500}
				placeholder="Optional – shown below the name"
				bind:value={editCollectionDescription}
			/>
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
		{privateVia}
		onchanged={() => {
			loadData(collectionId);
			reloadSidebar();
		}}
	/>
{/if}

<!-- Move Bookmark Modal -->
<Modal title="Move link" open={bookmarkToMove !== null} size="sm" autoclose={false} onclose={() => (bookmarkToMove = null)}>
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			if (bookmarkToMove && moveTargetId) moveBookmark(bookmarkToMove, moveTargetId);
			bookmarkToMove = null;
		}}
	>
		<p class="truncate text-sm text-gray-600 dark:text-gray-400">
			<strong>{bookmarkToMove?.title || bookmarkToMove?.url}</strong>
		</p>
		<div>
			<Label for="move-target" class="mb-2">Collection</Label>
			<Select id="move-target" placeholder="Choose a collection" bind:value={moveTargetId} required>
				{#each moveTargets as option (option.id)}
					<option value={option.id}>{'─'.repeat(option.depth)} {option.label}</option>
				{/each}
			</Select>
		</div>
		<p class="text-xs text-gray-500 dark:text-gray-400">Tip: you can also drag links onto a collection in the sidebar.</p>
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (bookmarkToMove = null)}>Cancel</Button>
			<Button type="submit" disabled={!moveTargetId}>Move</Button>
		</div>
	</form>
</Modal>

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
