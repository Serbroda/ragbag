<script lang="ts">
	import { BookmarkApi, CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, CollectionDto, SpacePermissionEnum } from 'ragbag-frontend-sdk';
	import { getContext, tick } from 'svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import BookmarkCard from '../components/BookmarkCard.svelte';
	import LinkViewToggle from '../components/LinkViewToggle.svelte';
	import LinkSortSelect from '../components/LinkSortSelect.svelte';
	import { filterByTags, sortLinks } from '../utils/links';
	import TagFilter from '../components/TagFilter.svelte';
	import TagInput from '../components/TagInput.svelte';
	import PreviewImageField from '../components/PreviewImageField.svelte';
	import { linkListClass, preferences } from '../stores/preferences.svelte';
	import LinkFormFields from '../components/LinkFormFields.svelte';
	import PinToDashboardModal from '../components/PinToDashboardModal.svelte';
	import CollectionAccessModal from '../components/CollectionAccessModal.svelte';
	import ImportCollectionModal from '../components/ImportCollectionModal.svelte';
	import ShareCollectionModal from '../components/ShareCollectionModal.svelte';
	import { DotsVerticalOutline, FolderOutline, GlobeOutline, LockOutline } from 'flowbite-svelte-icons';
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
	type LockAction = (collectionId: string, label: string) => Promise<void>;
	const lockForMe = getContext<LockAction | undefined>('lock-collection');
	const unlockForMe = getContext<LockAction | undefined>('unlock-collection');

	let collection = $state<CollectionDto | null>(null);
	let bookmarks = $state<BookmarkDto[]>([]);
	/** Tags picked above the links; a link has to have all of them. */
	let selectedTags = $state<string[]>([]);
	const visibleBookmarks = $derived(sortLinks(filterByTags(bookmarks, selectedTags), preferences.linkSort));
	let loading = $state(true);
	let error = $state('');
	/** Errors of individual actions; unlike `error` they don't replace the page. */
	let actionError = $state('');
	let modalError = $state('');
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
	let editImageUrl = $state('');
	let editImageHidden = $state(false);
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
	let showShareModal = $state(false);
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

	/** Private collections (or those inside one) cannot be shared by a public link. */
	const privateReason = $derived(
		collection?.restricted
			? 'Private collections cannot be shared publicly.'
			: privateVia
				? `This collection is inside the private collection “${privateVia}”, so it cannot be shared publicly.`
				: undefined,
	);

	const tree = $derived(collectionTree?.() ?? []);
	const subcollections = $derived(findNode(tree, collectionId)?.children ?? []);
	const lockedForMe = $derived(findNode(tree, collectionId)?.locked ?? false);
	/** Every other collection of the space, for "Move to…". */
	const moveTargets = $derived(flattenTree(tree).filter((option) => option.id !== collectionId));

	// Dialogs requested from the sidebar's context menu
	$effect(() => {
		const pending = pendingCollectionAction.current;
		if (!pending || pending.collectionId !== collectionId || collection?.id !== collectionId) return;
		pendingCollectionAction.current = null;
		if (pending.action === 'edit') openEditCollectionModal();
		else if (pending.action === 'delete') openDeleteCollectionModal();
		else if (pending.action === 'share') showShareModal = true;
		else showAccessModal = true;
	});

	$effect(() => {
		if (collectionId) {
			loadData(collectionId);
		}
	});

	/** Plain variable on purpose: loadData runs inside an effect, which must not depend on what it loads. */
	let loadedId = '';

	async function loadData(id: string) {
		if (id !== loadedId) {
			selectedTags = [];
			clearSelection();
		}
		loadedId = id;
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
		editImageUrl = bookmark.customImageUrl ?? '';
		editImageHidden = bookmark.imageHidden ?? false;
		modalError = '';
	}

	/** After an upload or a reset in the edit dialog, which apply right away. */
	function imageChanged(updated: BookmarkDto) {
		editingBookmark = updated;
		bookmarks = bookmarks.map((b) => (b.id === updated.id ? { ...updated, isNew: b.isNew } : b));
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
					// "" keeps an upload and removes an own address
					customImageUrl: editImageUrl.trim(),
					imageHidden: editImageHidden,
				},
			});
			bookmarks = bookmarks.map((b) => (b.id === updated.id ? { ...updated, isNew: b.isNew } : b));
			editingBookmark = null;
		} catch (err) {
			modalError = await errorMessage(err, 'Failed to update link.');
		} finally {
			saving = false;
		}
	}

	/** Deleting moves links to the trash, so there is no question first: the toast offers to undo it. */
	async function deleteBookmarks(targets: BookmarkDto[]) {
		if (!canDeleteBookmarks || targets.length === 0) return;
		actionError = '';
		const ids = targets.map((b) => b.id);
		try {
			for (const id of ids) await bookmarkApi.deleteBookmark({ bookmarkId: id });
		} catch (err) {
			actionError = await errorMessage(err, 'Failed to delete link.');
		}
		bookmarks = bookmarks.filter((b) => !ids.includes(b.id));
		reloadSidebar();
		showToast(targets.length === 1 ? 'Link moved to the trash' : `${targets.length} links moved to the trash`, {
			label: 'Undo',
			run: () => restoreFromTrash({ bookmarkIds: ids }),
		});
	}

	async function restoreFromTrash(selection: { bookmarkIds?: string[]; collectionIds?: string[] }) {
		try {
			await new SpaceApi(apiConfig()).restoreFromTrash({ spaceId, trashSelectionDto: selection });
			reloadSidebar();
			if (selection.bookmarkIds) await loadData(collectionId);
		} catch (err) {
			showToast(await errorMessage(err, 'Could not restore it.'));
		}
	}

	/** Moves a link to another collection of the space; the toast offers to move it back. */
	// --- Multi-select: move, tag or delete several links at once ---
	let selectedIds = $state<string[]>([]);
	let lastSelectedId: string | null = null;
	/** Selecting without hover (phones): checkboxes on every card, a tap selects instead of opening. */
	let selectionMode = $state(false);
	const selecting = $derived(selectionMode || selectedIds.length > 0);
	const canSelect = $derived(canEditBookmarks || canDeleteBookmarks);
	const selectedBookmarks = $derived(bookmarks.filter((b) => selectedIds.includes(b.id)));
	let bulkMoveOpen = $state(false);
	let bulkMoveTarget = $state('');
	let bulkTagsOpen = $state(false);
	let bulkTags = $state('');
	let bulkBusy = $state(false);

	/** Shift-click selects everything between the last clicked link and this one, as shown. */
	function toggleSelected(bookmark: BookmarkDto, event: MouseEvent) {
		const order = visibleBookmarks.map((b) => b.id);
		if (event.shiftKey && lastSelectedId && order.includes(lastSelectedId)) {
			const [from, to] = [order.indexOf(lastSelectedId), order.indexOf(bookmark.id)].sort((a, b) => a - b);
			selectedIds = [...new Set([...selectedIds, ...order.slice(from, to + 1)])];
		} else {
			selectedIds = selectedIds.includes(bookmark.id)
				? selectedIds.filter((id) => id !== bookmark.id)
				: [...selectedIds, bookmark.id];
		}
		lastSelectedId = bookmark.id;
		selectionMode = true;
	}

	function clearSelection() {
		selectedIds = [];
		lastSelectedId = null;
		selectionMode = false;
	}

	function updateDto(b: BookmarkDto, changes: { tags?: string[]; collectionId?: string }) {
		return {
			url: b.url,
			title: b.title,
			description: b.description,
			tags: changes.tags ?? b.tags,
			collectionId: changes.collectionId,
		};
	}

	async function moveSelected(targetId: string) {
		const moving = selectedBookmarks.filter((b) => b.collectionId !== targetId);
		bulkBusy = true;
		try {
			for (const b of moving) {
				await bookmarkApi.updateBookmark({ bookmarkId: b.id, updateBookmarkDto: updateDto(b, { collectionId: targetId }) });
			}
		} catch (err) {
			actionError = await errorMessage(err, 'Failed to move links.');
		} finally {
			bulkBusy = false;
		}
		const name = findNode(tree, targetId)?.label ?? 'collection';
		const from = collectionId;
		clearSelection();
		await loadData(collectionId);
		reloadSidebar();
		showToast(`Moved ${moving.length} link${moving.length === 1 ? '' : 's'} to ${name}`, {
			label: 'Undo',
			run: async () => {
				for (const b of moving) {
					await bookmarkApi.updateBookmark({ bookmarkId: b.id, updateBookmarkDto: updateDto(b, { collectionId: from }) });
				}
				await loadData(collectionId);
				reloadSidebar();
			},
		});
	}

	async function tagSelected() {
		const added = parseTags(bulkTags);
		if (added.length === 0) return;
		bulkBusy = true;
		try {
			for (const b of selectedBookmarks) {
				const own = b.tags ?? [];
				const tags = [...own, ...added.filter((tag) => !own.some((t) => t.toLowerCase() === tag.toLowerCase()))];
				const updated = await bookmarkApi.updateBookmark({ bookmarkId: b.id, updateBookmarkDto: updateDto(b, { tags }) });
				bookmarks = bookmarks.map((x) => (x.id === updated.id ? { ...updated, isNew: x.isNew } : x));
			}
			showToast(`Tagged ${selectedBookmarks.length} link${selectedBookmarks.length === 1 ? '' : 's'}`);
			bulkTagsOpen = false;
			bulkTags = '';
			clearSelection();
		} catch (err) {
			actionError = await errorMessage(err, 'Failed to tag links.');
		} finally {
			bulkBusy = false;
		}
	}

	async function deleteSelected() {
		const targets = selectedBookmarks;
		clearSelection();
		await deleteBookmarks(targets);
	}

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
			const deleted = { id: collectionId, name: collection?.name ?? 'Collection' };
			const space = spaceId;
			navigate('/space/:spaceId', { params: { spaceId } });
			showToast(`“${deleted.name}” moved to the trash`, {
				label: 'Undo',
				run: async () => {
					await new SpaceApi(apiConfig()).restoreFromTrash({
						spaceId: space,
						trashSelectionDto: { collectionIds: [deleted.id] },
					});
					reloadSidebar();
					navigate('/space/:spaceId/collections/:collectionId', {
						params: { spaceId: space, collectionId: deleted.id },
					});
				},
			});
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

<svelte:window
	onclick={() => ((openMenuId = null), (openHeaderMenu = false))}
	onpaste={handlePaste}
	onkeydown={(e) => e.key === 'Escape' && selecting && !document.querySelector('dialog[open]') && clearSelection()}
/>

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
					{:else if collection.shared}
						<button
							type="button"
							class="inline-flex shrink-0 items-center gap-1 rounded-md bg-sky-500/15 px-2 py-0.5 text-xs font-medium text-sky-700 ring-1 ring-inset ring-sky-500/30 hover:bg-sky-500/25 dark:text-sky-300"
							title="Anybody with the link can see this collection"
							onclick={() => (showShareModal = true)}
						>
							<GlobeOutline class="h-3.5 w-3.5 shrink-0" /> Shared publicly
						</button>
					{/if}
				</h1>
				{#if collection.description}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{collection.description}</p>
				{/if}
			</div>
			<div class="flex shrink-0 items-center gap-2">
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
								<button
									class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
									onclick={() => {
										showShareModal = true;
										openHeaderMenu = false;
									}}
								>
									Share publicly…
								</button>
								<button
									class="w-full px-3 py-1.5 text-left text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
									title="Hide it from your view until you unlock with your PIN; nobody else is affected"
									onclick={() => {
										openHeaderMenu = false;
										if (collection) (lockedForMe ? unlockForMe : lockForMe)?.(collectionId, collection.name);
									}}
								>
									{lockedForMe ? "Don't lock for me" : 'Lock for me'}
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

		{#if bookmarks.length > 0}
			<div class="mb-4 flex flex-wrap items-start gap-2">
				<div class="min-w-0 flex-1 basis-full sm:basis-0">
					<TagFilter links={bookmarks} bind:selected={selectedTags} />
				</div>
				<div class="ms-auto flex items-center gap-2">
					{#if canSelect && !selecting}
						<button
							type="button"
							class="rounded-lg bg-gray-100 px-2.5 py-1.5 text-sm text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700"
							title="Select several links (or long-press a card)"
							onclick={() => (selectionMode = true)}>Select</button
						>
					{/if}
					<LinkSortSelect />
					<LinkViewToggle />
				</div>
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
			{#if visibleBookmarks.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">No links have all of the selected tags.</p>
			{/if}
			<div class={linkListClass(preferences.linkView)}>
				{#each visibleBookmarks as bookmark (bookmark.id)}
					<BookmarkCard
						{bookmark}
						ondragstart={canEditBookmarks ? (e) => startBookmarkDrag(bookmark, e) : undefined}
						ondragend={endBookmarkDrag}
						selectable={canSelect}
						selected={selectedIds.includes(bookmark.id)}
						{selecting}
						onselect={(e) => toggleSelected(bookmark, e)}
						raised={openMenuId === bookmark.id}
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
											onclick={() => {
												deleteBookmarks([bookmark]);
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
			excludeId={editingBookmark?.id}
			bind:url={editUrl}
			bind:title={editTitle}
			bind:description={editDescription}
			bind:tags={editTags}
		/>
		{#if editingBookmark}
			<PreviewImageField
				bookmark={editingBookmark}
				bind:imageUrl={editImageUrl}
				bind:hidden={editImageHidden}
				onchange={imageChanged}
			/>
		{/if}
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
		Move <strong>{collection?.name}</strong> with all its links and sub-collections to the trash? You can restore
		it from there for 30 days.
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
	<ShareCollectionModal
		bind:open={showShareModal}
		{collectionId}
		collectionName={collection.name}
		canManage={canEditCollections}
		{privateReason}
		onchanged={() => {
			loadData(collectionId);
			reloadSidebar();
		}}
	/>
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

{#if selecting}
	<!-- Actions for the selected links -->
	<div
		class="fixed inset-x-3 bottom-4 z-40 flex items-center justify-between gap-1 rounded-xl bg-gray-950 px-2 py-1.5 text-sm text-gray-100 shadow-2xl ring-1 ring-white/10 sm:inset-x-auto sm:bottom-6 sm:left-1/2 sm:-translate-x-1/2 sm:justify-start"
		role="toolbar"
		aria-label="Selected links"
	>
		<span class="whitespace-nowrap px-2 font-semibold tabular-nums"
			>{selectedIds.length === 0 ? 'Tap links' : selectedIds.length}<span class="hidden sm:inline"
				>{selectedIds.length === 0 ? ' to select' : ' selected'}</span
			></span
		>
		{#if selectedIds.length < visibleBookmarks.length}
			<button
				type="button"
				class="rounded-md px-2 py-1 hover:bg-white/10 disabled:pointer-events-none disabled:opacity-40"
				onclick={() => (selectedIds = visibleBookmarks.map((b) => b.id))}><span class="sm:hidden">All</span><span class="hidden sm:inline">Select all</span></button
			>
		{/if}
		{#if canEditBookmarks}
			<button
				type="button"
				class="rounded-md px-2 py-1 hover:bg-white/10 disabled:pointer-events-none disabled:opacity-40"
				disabled={bulkBusy || selectedIds.length === 0}
				onclick={() => ((bulkMoveTarget = ''), (bulkMoveOpen = true))}><span class="sm:hidden">Move</span><span class="hidden sm:inline">Move to…</span></button
			>
			<button
				type="button"
				class="rounded-md px-2 py-1 hover:bg-white/10 disabled:pointer-events-none disabled:opacity-40"
				disabled={bulkBusy || selectedIds.length === 0}
				onclick={() => ((bulkTags = ''), (bulkTagsOpen = true))}><span class="sm:hidden">Tags</span><span class="hidden sm:inline">Add tags…</span></button
			>
		{/if}
		{#if canDeleteBookmarks}
			<button type="button" class="rounded-md px-2 py-1 text-red-400 hover:bg-red-600 hover:text-white disabled:pointer-events-none disabled:opacity-40" disabled={bulkBusy || selectedIds.length === 0} onclick={deleteSelected}
				>Delete</button
			>
		{/if}
		<button
			type="button"
			class="rounded-md p-1 text-gray-400 hover:bg-white/10 hover:text-gray-100"
			aria-label="Clear selection"
			title="Clear selection (Esc)"
			onclick={clearSelection}
		>
			<svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"
				><path stroke-linecap="round" d="M6 6l12 12M18 6L6 18" /></svg
			>
		</button>
	</div>
{/if}

<Modal title="Move {selectedIds.length} links" bind:open={bulkMoveOpen} size="sm" autoclose={false}>
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			if (!bulkMoveTarget) return;
			bulkMoveOpen = false;
			moveSelected(bulkMoveTarget);
		}}
	>
		<div>
			<Label for="bulk-move-target" class="mb-2">Collection</Label>
			<Select id="bulk-move-target" placeholder="Choose a collection" bind:value={bulkMoveTarget} required>
				{#each moveTargets as option (option.id)}
					<option value={option.id}>{'─'.repeat(option.depth)} {option.label}</option>
				{/each}
			</Select>
		</div>
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (bulkMoveOpen = false)}>Cancel</Button>
			<Button type="submit" disabled={!bulkMoveTarget}>Move</Button>
		</div>
	</form>
</Modal>

<Modal title="Add tags to {selectedIds.length} links" bind:open={bulkTagsOpen} size="sm" autoclose={false}>
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			tagSelected();
		}}
	>
		<div>
			<Label for="bulk-tags" class="mb-2">Tags</Label>
			<TagInput id="bulk-tags" {spaceId} bind:value={bulkTags} />
			<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">Added to the tags the links already have.</p>
		</div>
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (bulkTagsOpen = false)}>Cancel</Button>
			<Button type="submit" disabled={bulkBusy || parseTags(bulkTags).length === 0}>Add tags</Button>
		</div>
	</form>
</Modal>
