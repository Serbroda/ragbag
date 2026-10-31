<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import type { SpacePermissionEnum } from 'ragbag-frontend-sdk';
	import { setContext } from 'svelte';
	import { Button, Modal, Label, Input, Select, Toggle } from 'flowbite-svelte';
	import {
		ArrowRightToBracketOutline,
		BarsOutline,
		ChevronDownOutline,
		ChevronRightOutline,
		CloseOutline,
		CogSolid,
		EyeOutline,
		EyeSlashOutline,
		LockOpenOutline,
		FolderOutline,
		FolderPlusSolid,
		GlobeOutline,
		HomeSolid,
		LinkOutline,
		LockOutline,
		PenOutline,
		PlusOutline,
		SearchOutline,
		StarSolid,
		TrashBinOutline,
		UserAddSolid,
		UserCircleOutline,
		UsersOutline,
	} from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage, fetchFeatureFlags, logout, logoutEverywhere } from '../api/client';
	import { navigate, p, route } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import DraggableTreeList from '../components/DraggableTreeList.svelte';
	import CreateOrJoinSpaceModal from '../components/CreateOrJoinSpaceModal.svelte';
	import InvitePeopleModal from '../components/InvitePeopleModal.svelte';
	import QuickSwitcher, { type SwitcherItem } from '../components/QuickSwitcher.svelte';
	import SpaceRail from '../components/SpaceRail.svelte';
	import NewsBell from '../components/NewsBell.svelte';
	import Avatar from '../components/Avatar.svelte';
	import MemberChecklist from '../components/MemberChecklist.svelte';
	import ProfileModal from '../components/ProfileModal.svelte';
	import AdminUsersModal from '../components/AdminUsersModal.svelte';
	import { collectionsToTree, flattenTree, pathToNode, type TreeNode } from '../components/tree';
	import { CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { SpaceMemberDto } from 'ragbag-frontend-sdk';
	import {
		loadPreferences,
		preferences,
		saveSidebarWidth,
		setExpanded,
		setSidebarWidth,
	} from '../stores/preferences.svelte';
	import { pendingCollectionAction, type CollectionAction } from '../stores/collectionAction.svelte';
	import { showToast } from '../stores/toast.svelte';
	import { checkExpiry, keepUnlocked, lock, lockState, resetLock } from '../stores/lock.svelte';
	import LockDialog from '../components/LockDialog.svelte';
	import Toasts from '../components/Toasts.svelte';
	import PullToRefresh from '../components/PullToRefresh.svelte';
	import { copyText } from '../utils/clipboard';
	import { loadMe, loadSpaces, rememberSpace, resetSession, session } from '../stores/spaces.svelte';

	let { children }: { children: Snippet } = $props();

	let sidebarOpen = $state(false);

	// --- Spaces & Collections ---
	const spaceApi = new SpaceApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());

	let menuTree = $state<TreeNode[]>([]);

	const activeSpaceId = $derived(route.params.spaceId ?? null);
	const activeCollectionId = $derived(route.params.collectionId ?? null);
	const activeSpace = $derived(session.spaces.find((space) => space.id === activeSpaceId) ?? null);
	const activePermissions = $derived(new Set<SpacePermissionEnum>(activeSpace?.permissions ?? []));
	const canCreateCollections = $derived(activePermissions.has('CREATE_COLLECTIONS'));
	const canEditCollections = $derived(activePermissions.has('EDIT_COLLECTIONS'));
	const canManageMembers = $derived(activePermissions.has('MANAGE_MEMBERS'));

	async function loadCollections(spaceId: string) {
		try {
			const collections = await collectionApi.getCollections({ spaceId });
			if (spaceId !== activeSpaceId) return;
			menuTree = collectionsToTree(collections, spaceId);
		} catch {
			// Access was revoked or the space was deleted meanwhile
			menuTree = [];
			await loadSpaces();
		}
	}

	function selectSpace(spaceId: string) {
		closeMenus();
		navigate('/space/:spaceId', { params: { spaceId } });
	}

	// The membership check below only trusts a list loaded by this layout, not a stale one from before
	let spacesReady = $state(false);
	loadSpaces()
		.then(() => (spacesReady = true))
		.catch(() => {});
	loadMe().catch(() => {});

	/** Version of the running server, e.g. v1.2.0 or v1.2.0-3-gabc1234 for a build after that release. */
	let appVersion = $state('');
	fetchFeatureFlags()
		.then((flags) => (appVersion = flags['version'] ?? ''))
		.catch(() => {});

	// An installed app can stay open for days: offer a reload once the server runs another version
	let offeredVersion = '';
	async function checkForUpdate() {
		if (!appVersion || document.visibilityState !== 'visible') return;
		const version = (await fetchFeatureFlags().catch(() => ({}) as Record<string, string>))['version'];
		if (!version || version === appVersion || version === offeredVersion) return;
		offeredVersion = version;
		showToast(`Ragbag ${version} is available`, { label: 'Reload', run: () => location.reload() }, 0);
	}
	$effect(() => {
		const timer = setInterval(checkForUpdate, 30 * 60 * 1000);
		document.addEventListener('visibilitychange', checkForUpdate);
		return () => {
			clearInterval(timer);
			document.removeEventListener('visibilitychange', checkForUpdate);
		};
	});

	let scrollRoot = $state<HTMLElement>();

	$effect(() => {
		if (session.me) loadPreferences(session.me.id);
	});

	$effect(() => {
		if (activeSpaceId) {
			loadCollections(activeSpaceId);
		}
	});

	// Opening a collection (click, quick switcher, breadcrumb, link) shows it with its children in the tree.
	// Only once per opened collection: reloading the tree must not reopen what the user collapsed.
	let expandedFor: string | null = null;
	$effect(() => {
		if (!activeCollectionId || activeCollectionId === expandedFor) return;
		const path = pathToNode(menuTree, activeCollectionId);
		if (path.length === 0) return;
		expandedFor = activeCollectionId;
		for (const node of path) {
			if (node.children?.length) setExpanded(node.id!, true);
		}
	});

	// Leave spaces the user is no longer a member of (removed, deleted, bad link)
	$effect(() => {
		if (!spacesReady || !activeSpaceId) return;
		// Remembered per user, so it waits for the account to be loaded
		session.me;
		if (activeSpace) {
			rememberSpace(activeSpace.id);
		} else {
			navigate('/');
		}
	});

	setContext('reload-collections', () => {
		if (activeSpaceId) loadCollections(activeSpaceId);
	});

	setContext('has-space-permission', (permission: SpacePermissionEnum) => activePermissions.has(permission));

	setContext('reload-spaces', loadSpaces);

	// New links of others show up without reloading: refresh the counters every minute while the tab is visible
	function refreshCounters() {
		if (document.visibilityState !== 'visible') return;
		loadSpaces().catch(() => {});
		if (activeSpaceId) loadCollections(activeSpaceId);
	}
	$effect(() => {
		const timer = setInterval(refreshCounters, 60_000);
		document.addEventListener('visibilitychange', refreshCounters);
		return () => {
			clearInterval(timer);
			document.removeEventListener('visibilitychange', refreshCounters);
		};
	});

	setContext('open-invite', () => (showInvite = true));

	const isMac = /mac/i.test(navigator.platform || navigator.userAgent);

	setContext('collection-name', (collectionId: string) => findLabel(menuTree, collectionId));

	/** The collection tree of the sidebar, e.g. for sub-collections and move targets on the collection page. */
	setContext('collection-tree', () => menuTree);

	// Visibility is inherited: below a private collection only its members see anything
	setContext('private-ancestor', (collectionId: string) =>
		pathToNode(menuTree, collectionId)
			.slice(0, -1)
			.reverse()
			.find((node) => node.restricted)?.label,
	);

	function findLabel(nodes: TreeNode[], id: string): string | undefined {
		for (const node of nodes) {
			if (node.id === id) return node.label;
			const found = node.children && findLabel(node.children, id);
			if (found) return found;
		}
		return undefined;
	}

	// Close the off-canvas navigation after navigating on mobile
	$effect(() => {
		route.pathname;
		sidebarOpen = false;
	});

	// --- Search ---
	let searchQuery = $state('');
	let searchTimeout: ReturnType<typeof setTimeout> | undefined;

	/** Phones have no room for the search box in the header; it opens as a row below it. */
	let mobileSearchOpen = $state(false);

	// Reset the search box when switching spaces; prefill it on the search page
	$effect(() => {
		activeSpaceId;
		const q = route.search.q;
		const onSearchPage = route.pathname.endsWith('/search');
		searchQuery = onSearchPage && typeof q === 'string' ? q : '';
		if (!onSearchPage) mobileSearchOpen = false;
	});

	function focusOnMount(element: HTMLElement) {
		element.focus();
	}

	function doSearch(value: string) {
		if (!activeSpaceId || !value.trim()) return;
		navigate('/space/:spaceId/search', { params: { spaceId: activeSpaceId }, search: { q: value.trim() } });
	}

	function handleSearchInput() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => doSearch(searchQuery), 300);
	}

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			clearTimeout(searchTimeout);
			doSearch(searchQuery);
		}
	}

	// --- Resize ---
	const SPACE_BAR_WIDTH = 72;

	let isResizing = $state(false);

	function startResize(e: MouseEvent) {
		e.preventDefault();
		isResizing = true;

		const onMouseMove = (e: MouseEvent) => {
			setSidebarWidth(e.clientX - SPACE_BAR_WIDTH);
		};

		const onMouseUp = () => {
			isResizing = false;
			saveSidebarWidth();
			document.removeEventListener('mousemove', onMouseMove);
			document.removeEventListener('mouseup', onMouseUp);
		};

		document.addEventListener('mousemove', onMouseMove);
		document.addEventListener('mouseup', onMouseUp);
	}

	// --- Menus & panels ---
	let spaceMenuOpen = $state(false);
	let userMenuOpen = $state(false);
	let switcherOpen = $state(false);
	let showCreateOrJoin = $state(false);
	let showInvite = $state(false);
	let showProfile = $state(false);
	let showAdminUsers = $state(false);

	function closeMenus() {
		spaceMenuOpen = false;
		userMenuOpen = false;
		collectionMenu = null;
	}

	// --- Collection context menu (right click or ⋯ in the tree) ---
	let collectionMenu = $state<{ node: TreeNode; x: number; y: number } | null>(null);
	const canDeleteCollections = $derived(activePermissions.has('DELETE_COLLECTIONS'));

	setContext('collection-menu', (node: TreeNode, x: number, y: number) => {
		spaceMenuOpen = false;
		userMenuOpen = false;
		// Keep the menu (w-52, up to six items) inside the window
		collectionMenu = { node, x: Math.min(x, window.innerWidth - 216), y: Math.min(y, window.innerHeight - 240) };
	});

	function runCollectionAction(node: TreeNode, action: CollectionAction) {
		closeMenus();
		pendingCollectionAction.current = { collectionId: node.id!, action };
		if (activeCollectionId !== node.id) {
			navigate('/space/:spaceId/collections/:collectionId', {
				params: { spaceId: activeSpaceId!, collectionId: node.id! },
			});
		}
	}

	// --- Locked collections (hidden from the own view until unlocked with the PIN) ---
	let lockDialogOpen = $state(false);
	let lockDialogMode = $state<'unlock' | 'setup'>('unlock');
	let afterPinSetup: (() => void) | undefined;
	const lockShortcut = isMac ? '⌘⇧L' : 'Ctrl+Shift+L';

	function openUnlock() {
		closeMenus();
		afterPinSetup = undefined;
		lockDialogMode = session.me?.hasPin ? 'unlock' : 'setup';
		lockDialogOpen = true;
	}

	function toggleLock() {
		if (lockState.unlocked) lock();
		else openUnlock();
	}

	/** Collections the user locked, from the tree as it was before it reloads. */
	function insideLocked(collectionId: string | null): boolean {
		return !!collectionId && pathToNode(menuTree, collectionId).some((node) => node.locked);
	}

	// Locking or unlocking changes what every page may show: leave a locked collection, then reload everything
	let seenLockVersion = lockState.version;
	$effect(() => {
		const version = lockState.version;
		if (version === seenLockVersion) return;
		seenLockVersion = version;
		if (!lockState.unlocked && insideLocked(activeCollectionId) && activeSpaceId) {
			navigate('/space/:spaceId', { params: { spaceId: activeSpaceId } });
		}
		if (activeSpaceId) loadCollections(activeSpaceId);
		loadSpaces().catch(() => {});
	});

	// Activity keeps an unlock alive; idle tabs lock again when the server's token runs out
	$effect(() => {
		const timer = setInterval(checkExpiry, 15_000);
		return () => clearInterval(timer);
	});

	async function lockCollection(collectionId: string, label: string) {
		closeMenus();
		if (!session.me?.hasPin) {
			afterPinSetup = () => lockCollection(collectionId, label);
			lockDialogMode = 'setup';
			lockDialogOpen = true;
			return;
		}
		try {
			await collectionApi.lockCollection({ collectionId });
		} catch (err) {
			showToast(await errorMessage(err, 'Could not lock the collection.'));
			return;
		}
		if (lockState.unlocked) {
			showToast(`“${label}” is locked. It disappears when you lock (${lockShortcut}).`);
		} else {
			// Leave it right away when it is the open one (or inside it)
			const path = activeCollectionId ? pathToNode(menuTree, activeCollectionId) : [];
			if (path.some((node) => node.id === collectionId) && activeSpaceId) {
				navigate('/space/:spaceId', { params: { spaceId: activeSpaceId } });
			}
			showToast(`“${label}” is hidden. Unlock with ${lockShortcut} or in the account menu.`);
		}
		if (activeSpaceId) loadCollections(activeSpaceId);
		loadSpaces().catch(() => {});
	}

	async function unlockCollection(collectionId: string, label: string) {
		closeMenus();
		try {
			await collectionApi.unlockCollection({ collectionId });
			showToast(`“${label}” is no longer locked.`);
			if (activeSpaceId) loadCollections(activeSpaceId);
		} catch (err) {
			showToast(await errorMessage(err, 'Could not change the collection.'));
		}
	}

	setContext('lock-collection', lockCollection);
	setContext('unlock-collection', unlockCollection);

	async function copyCollectionLink(node: TreeNode) {
		closeMenus();
		const copied = await copyText(new URL(node.href!, window.location.origin).href);
		showToast(copied ? 'Link copied' : 'Could not copy the link');
	}

	function onWindowKeydown(e: KeyboardEvent) {
		keepUnlocked();
		if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === 'l') {
			e.preventDefault();
			toggleLock();
		} else if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			switcherOpen = !switcherOpen;
		} else if (e.key === 'Escape') {
			closeMenus();
			mobileSearchOpen = false;
		}
	}

	const switcherItems = $derived<SwitcherItem[]>([
		...session.spaces.map((space) => ({
			id: space.id,
			label: space.name,
			hint: 'Space',
			kind: 'space' as const,
			avatar: { name: space.name, iconId: space.iconId },
			go: () => selectSpace(space.id),
		})),
		...(activeSpaceId
			? flattenTree(menuTree).map((node) => ({
					id: node.id,
					label: node.label,
					hint: activeSpace?.name ?? '',
					kind: 'collection' as const,
					go: () =>
						navigate('/space/:spaceId/collections/:collectionId', {
							params: { spaceId: activeSpaceId, collectionId: node.id },
						}),
				}))
			: []),
	]);

	async function handleSpaceDone(spaceId: string) {
		await loadSpaces();
		selectSpace(spaceId);
	}

	let showLeaveModal = $state(false);
	let leaveError = $state('');
	let leaving = $state(false);

	function openLeaveModal() {
		closeMenus();
		leaveError = '';
		showLeaveModal = true;
	}

	async function handleLeaveSpace() {
		if (!activeSpaceId) return;
		leaving = true;
		leaveError = '';
		try {
			await spaceApi.leaveSpace({ spaceId: activeSpaceId });
			showLeaveModal = false;
			await loadSpaces();
			navigate('/');
		} catch (err) {
			leaveError = await errorMessage(err, 'Failed to leave space.');
		} finally {
			leaving = false;
		}
	}

	async function handleLogout(everywhere = false) {
		closeMenus();
		resetLock();
		await (everywhere ? logoutEverywhere() : logout()).catch(() => {});
		resetSession();
		navigate('/login');
	}

	// --- Add Collection Modal ---
	let showAddModal = $state(false);
	let newCollectionName = $state('');
	let newCollectionParentId = $state('');
	let createError = $state('');
	let creating = $state(false);
	let newCollectionPrivate = $state(false);
	let newCollectionMembers = $state(new Set<string>());
	let spaceMembers = $state<SpaceMemberDto[]>([]);

	async function togglePrivate() {
		if (newCollectionPrivate && activeSpaceId && spaceMembers.length === 0) {
			spaceMembers = await spaceApi.getSpaceMembers({ spaceId: activeSpaceId }).catch(() => []);
		}
	}

	const parentOptions = $derived(flattenTree(menuTree));

	function openAddModal(parentId = activeCollectionId) {
		if (!canCreateCollections) return;
		closeMenus();
		newCollectionName = '';
		// Default to the collection the user is currently looking at
		newCollectionParentId = parentId ?? '';
		createError = '';
		creating = false;
		newCollectionPrivate = false;
		newCollectionMembers = new Set();
		spaceMembers = [];
		showAddModal = true;
	}

	async function handleCreateCollection() {
		if (!canCreateCollections) return;
		const name = newCollectionName.trim();
		if (!name || !activeSpaceId) return;
		createError = '';
		creating = true;
		try {
			const created = await collectionApi.createCollection({
				spaceId: activeSpaceId,
				createCollectionDto: {
					name,
					parentId: newCollectionParentId || undefined,
					restricted: newCollectionPrivate,
					memberIds: newCollectionPrivate ? [...newCollectionMembers] : undefined,
				},
			});
			showAddModal = false;
			await loadCollections(activeSpaceId);
			navigate('/space/:spaceId/collections/:collectionId', {
				params: { spaceId: activeSpaceId, collectionId: created.id },
			});
		} catch (err) {
			createError = await errorMessage(err, 'Failed to create collection.');
		} finally {
			creating = false;
		}
	}

	type Crumb = { label: string; href?: string; icon?: Component<{ class?: string }> };

	/** Header breadcrumbs: space › collection ancestors › current page. */
	const breadcrumbs = $derived.by((): Crumb[] => {
		if (!activeSpace) return [];
		const spaceHref = p('/space/:spaceId', { params: { spaceId: activeSpace.id } });
		const path = route.pathname;
		const space: Crumb = { label: activeSpace.name, href: spaceHref, icon: HomeSolid };
		if (path.endsWith('/settings')) return [space, { label: 'Settings', icon: CogSolid }];
		if (path.endsWith('/search')) return [space, { label: 'Search', icon: SearchOutline }];
		if (path.endsWith('/favorites')) return [space, { label: 'Favorites', icon: StarSolid }];
		if (path.endsWith('/trash')) return [space, { label: 'Trash', icon: TrashBinOutline }];
		if (activeCollectionId) {
			const trail = pathToNode(menuTree, activeCollectionId);
			if (trail.length === 0) return [space, { label: 'Collection', icon: FolderOutline }];
			return [
				space,
				...trail.map((node, i) => ({
					label: node.label,
					href: i < trail.length - 1 ? node.href : undefined,
					icon: i === trail.length - 1 ? (node.icon ?? FolderOutline) : undefined,
				})),
			];
		}
		return [{ ...space, href: undefined }, { label: 'Dashboard' }];
	});

	const navTree = $derived<TreeNode[]>(
		activeSpaceId
			? [
					{
						label: 'Dashboard',
						icon: HomeSolid,
						path: `/space/${activeSpaceId}`,
						href: p('/space/:spaceId', { params: { spaceId: activeSpaceId } }),
					},
					{
						label: 'Favorites',
						icon: StarSolid,
						path: `/space/${activeSpaceId}/favorites`,
						href: p('/space/:spaceId/favorites', { params: { spaceId: activeSpaceId } }),
					},
					{
						label: 'Trash',
						icon: TrashBinOutline,
						path: `/space/${activeSpaceId}/trash`,
						href: p('/space/:spaceId/trash', { params: { spaceId: activeSpaceId } }),
					},
				]
			: [],
	);

	async function handleMove(movedNode: TreeNode, newParent: TreeNode | null, before: TreeNode | null) {
		if (!canEditCollections) return;
		await collectionApi.moveCollection({
			collectionId: movedNode.id!,
			moveCollectionDto: {
				parentId: newParent?.id ?? undefined,
				beforeId: before?.id ?? undefined,
			},
		});
		if (activeSpaceId) {
			await loadCollections(activeSpaceId);
		}
	}
</script>

<svelte:window
	onkeydown={onWindowKeydown}
	onclick={closeMenus}
	oncontextmenu={closeMenus}
	onpointerdown={keepUnlocked}
/>

<div class="flex h-dvh overflow-hidden bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100" class:select-none={isResizing}>
	<!-- Mobile backdrop -->
	{#if sidebarOpen}
		<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
		<div class="fixed inset-0 z-30 bg-gray-950/60 md:hidden" onclick={() => (sidebarOpen = false)}></div>
	{/if}

	<!-- Navigation: space rail + sidebar (off-canvas on mobile) -->
	<div
		class="fixed inset-y-0 left-0 z-40 flex transition-transform duration-200 md:static md:translate-x-0 {sidebarOpen
			? 'translate-x-0'
			: '-translate-x-full'}"
	>
		<!-- Space rail (always dark, like Discord) -->
		<SpaceRail
			{activeSpaceId}
			onselect={selectSpace}
			onhome={() => navigate('/')}
			oncreate={() => (showCreateOrJoin = true)}
			onseen={(spaceId) => spaceId === activeSpaceId && loadCollections(spaceId)}
		/>

		<!-- Sidebar -->
		<aside
			class="relative flex h-full flex-col bg-gray-50 dark:bg-gray-900"
			style:width="{preferences.sidebarWidth}px"
			aria-label="Space navigation"
		>
			<!-- Space header with menu -->
			<div class="relative h-12 shrink-0 border-b border-gray-200 shadow-sm dark:border-gray-950/60">
				{#if activeSpace}
					<button
						type="button"
						class="flex h-full w-full items-center justify-between px-4 text-left font-semibold transition-colors hover:bg-gray-200/60 dark:hover:bg-gray-800"
						aria-haspopup="menu"
						aria-expanded={spaceMenuOpen}
						onclick={(e) => {
							e.stopPropagation();
							userMenuOpen = false;
							spaceMenuOpen = !spaceMenuOpen;
						}}
					>
						<span class="flex min-w-0 items-center gap-2.5">
							<Avatar name={activeSpace.name} imageId={activeSpace.iconId} class="h-7 w-7 rounded-lg text-[10px]" />
							<span class="truncate">{activeSpace.name}</span>
						</span>
						<ChevronDownOutline class="h-4 w-4 shrink-0 transition-transform {spaceMenuOpen ? 'rotate-180' : ''}" />
					</button>
					{#if spaceMenuOpen}
						<div
							role="menu"
							class="absolute left-2 right-2 top-[calc(100%+4px)] z-50 rounded-lg bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
						>
							{#if canManageMembers}
								<button
									role="menuitem"
									class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-primary-400 hover:bg-primary-500 hover:text-white"
									onclick={() => ((showInvite = true), closeMenus())}
								>
									Invite people <UserAddSolid class="h-4 w-4" />
								</button>
							{/if}
							<button
								role="menuitem"
								class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
								onclick={() => (closeMenus(), navigate('/space/:spaceId/settings', { params: { spaceId: activeSpace.id } }))}
							>
								Space settings <CogSolid class="h-4 w-4" />
							</button>
							{#if canCreateCollections}
								<button
									role="menuitem"
									class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
									onclick={() => openAddModal()}
								>
									Create collection <FolderPlusSolid class="h-4 w-4" />
								</button>
							{/if}
							<div class="my-1 h-px bg-gray-800"></div>
							<button
								role="menuitem"
								class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-red-400 hover:bg-red-600 hover:text-white"
								onclick={openLeaveModal}
							>
								Leave space <ArrowRightToBracketOutline class="h-4 w-4" />
							</button>
						</div>
					{/if}
				{/if}
			</div>

			<div class="flex-1 overflow-y-auto pb-3">
				<ul class="space-y-0.5 pt-3">
					{#each navTree as node (node.label)}
						<TreeItem {node} />
					{/each}
				</ul>
				<div class="mt-4 flex items-center justify-between px-4 pb-1">
					<h3 class="text-[11px] font-semibold uppercase tracking-wider text-gray-500">Collections</h3>
					{#if canCreateCollections}
						<button
							type="button"
							title="Create collection"
							aria-label="Create collection"
							class="rounded p-0.5 text-gray-400 transition-colors hover:text-gray-700 dark:hover:text-gray-200"
							onclick={(e) => (e.stopPropagation(), openAddModal())}
						>
							<PlusOutline class="h-4 w-4" />
						</button>
					{/if}
				</div>
				{#if menuTree.length === 0 && activeSpace}
					<p class="px-4 py-2 text-xs text-gray-500">
						{canCreateCollections ? 'No collections yet. Create one with +.' : 'No collections yet.'}
					</p>
				{/if}
				<DraggableTreeList bind:nodes={menuTree} onmove={handleMove} draggable={canEditCollections} />
			</div>

			<!-- User panel -->
			<div class="relative flex h-14 shrink-0 items-center gap-2 bg-gray-100 px-2 dark:bg-gray-950/50">
				{#if session.me}
					<button
						type="button"
						title="Profile & appearance"
						aria-label="Profile & appearance"
						class="shrink-0 rounded-full transition-opacity hover:opacity-80"
						onclick={(e) => (e.stopPropagation(), closeMenus(), (showProfile = true))}
					>
						<Avatar name={session.me.username} imageId={session.me.avatarId} />
					</button>
					<div class="min-w-0 flex-1 leading-tight">
						<p class="truncate text-sm font-semibold">{session.me.username}</p>
						{#if activeSpace?.role}
							<p class="truncate text-xs capitalize text-gray-500">{activeSpace.role.toLowerCase()}</p>
						{/if}
					</div>
				{:else}
					<div class="flex-1"></div>
				{/if}
				{#if lockState.unlocked}
					<button
						type="button"
						class="flex shrink-0 items-center gap-1 rounded-md bg-amber-500/15 px-1.5 py-1 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-500/30 hover:bg-amber-500/25 dark:text-amber-300"
						title="Locked collections are visible. Click or press {lockShortcut} to lock."
						onclick={(e) => (e.stopPropagation(), lock())}
					>
						<LockOpenOutline class="h-3.5 w-3.5" /> Unlocked
					</button>
				{/if}
				<button
					type="button"
					title="Account"
					aria-label="Account menu"
					aria-haspopup="menu"
					aria-expanded={userMenuOpen}
					class="rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
					onclick={(e) => {
						e.stopPropagation();
						spaceMenuOpen = false;
						userMenuOpen = !userMenuOpen;
					}}
				>
					<CogSolid class="h-5 w-5" />
				</button>
				{#if userMenuOpen}
					<div
						role="menu"
						class="absolute bottom-[calc(100%+4px)] left-2 right-2 z-50 rounded-lg bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
					>
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
							onclick={() => (closeMenus(), (showProfile = true))}
						>
							Profile & appearance <UserCircleOutline class="h-4 w-4" />
						</button>
						{#if session.me?.instanceAdmin}
							<button
								role="menuitem"
								class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
								onclick={() => (closeMenus(), (showAdminUsers = true))}
							>
								Manage users <UsersOutline class="h-4 w-4" />
							</button>
						{/if}
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
							onclick={() => (closeMenus(), toggleLock())}
						>
							{lockState.unlocked ? 'Lock now' : 'Unlock…'}
							<kbd class="font-sans text-xs opacity-60">{lockShortcut}</kbd>
						</button>
						<div class="my-1 h-px bg-gray-800"></div>
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
							onclick={() => handleLogout(false)}
						>
							Log out <ArrowRightToBracketOutline class="h-4 w-4" />
						</button>
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-red-400 hover:bg-red-600 hover:text-white"
							onclick={() => handleLogout(true)}
						>
							Log out on all devices
						</button>
						{#if appVersion}
							<div class="my-1 h-px bg-gray-800"></div>
							<p class="truncate px-2 pb-0.5 pt-1 text-xs text-gray-500" title="Version of this ragbag server">
								Ragbag {appVersion}
							</p>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Resize handle (desktop only) -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="absolute inset-y-0 -right-1 z-10 hidden w-2 cursor-col-resize md:block"
				onmousedown={startResize}
			>
				<div
					class="mx-auto h-full w-0.5 transition-colors {isResizing
						? 'bg-primary-500'
						: 'bg-transparent hover:bg-gray-300 dark:hover:bg-gray-700'}"
				></div>
			</div>
		</aside>
	</div>

	<!-- Main column: own header, only this area scrolls -->
	<div class="flex min-w-0 flex-1 flex-col">
		<header
			class="flex h-12 shrink-0 items-center gap-3 border-b border-gray-200 px-3 shadow-sm dark:border-gray-950/60"
		>
			<button
				type="button"
				aria-label="Open navigation"
				class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 md:hidden dark:hover:bg-gray-800"
				onclick={() => (sidebarOpen = true)}
			>
				<BarsOutline class="h-5 w-5" />
			</button>
			<nav aria-label="Breadcrumb" class="min-w-0">
				<ol class="flex min-w-0 items-center gap-1 text-sm">
					{#each breadcrumbs as crumb, i (i)}
						{@const last = i === breadcrumbs.length - 1}
						<!-- On small screens only the current page is shown -->
						<li class="items-center gap-1 {last ? 'flex min-w-0' : 'hidden shrink-0 sm:flex'}">
							{#if i > 0}
								<!-- Phones only show the last crumb, which then needs no separator -->
								<ChevronRightOutline class="hidden h-3.5 w-3.5 shrink-0 text-gray-400 sm:block" />
							{/if}
							{#if crumb.href && !last}
								<a
									href={crumb.href}
									class="flex max-w-48 items-center gap-1.5 truncate rounded px-1 py-0.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
								>
									{#if crumb.icon}
										<crumb.icon class="h-4 w-4 shrink-0" />
									{/if}
									<span class="truncate">{crumb.label}</span>
								</a>
							{:else}
								<span
									class="flex min-w-0 items-center gap-1.5 px-1 py-0.5 {last
										? 'font-semibold'
										: 'text-gray-500 dark:text-gray-400'}"
									aria-current={last ? 'page' : undefined}
								>
									{#if crumb.icon}
										<crumb.icon class="h-4 w-4 shrink-0 {last ? 'text-gray-400' : ''}" />
									{/if}
									<span class="truncate">{crumb.label}</span>
								</span>
							{/if}
						</li>
					{/each}
				</ol>
			</nav>
			<div class="ms-auto flex items-center gap-2">
				{#if activeSpaceId}
					<div class="relative hidden sm:block">
						<input
							type="search"
							placeholder="Search {activeSpace?.name ?? 'bookmarks'}"
							class="w-56 rounded-md border-0 bg-gray-100 py-1.5 pl-8 pr-3 text-sm text-gray-900 transition-all placeholder:text-gray-500 focus:w-72 focus:ring-2 focus:ring-primary-500 dark:bg-gray-950/70 dark:text-white"
							bind:value={searchQuery}
							oninput={handleSearchInput}
							onkeydown={handleSearchKeydown}
						/>
						<SearchOutline class="absolute left-2.5 top-2 h-4 w-4 text-gray-500" />
					</div>
				{/if}
				{#if activeSpaceId}
					<NewsBell spaceId={activeSpaceId} />
					<button
						type="button"
						aria-label="Search"
						aria-expanded={mobileSearchOpen}
						class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 sm:hidden dark:hover:bg-gray-800"
						onclick={() => (mobileSearchOpen = !mobileSearchOpen)}
					>
						<SearchOutline class="h-5 w-5" />
					</button>
				{/if}
				<!-- The quick switcher is a keyboard feature; phones have the navigation drawer -->
				<button
					type="button"
					class="hidden items-center gap-2 rounded-md bg-gray-100 px-2.5 py-1.5 text-xs text-gray-500 transition-colors hover:text-gray-800 sm:flex dark:bg-gray-950/70 dark:text-gray-400 dark:hover:text-gray-200"
					onclick={() => (switcherOpen = true)}
					title="Quick switcher"
				>
					<span class="hidden lg:inline">Jump to…</span>
					<kbd class="font-sans font-semibold">{isMac ? '⌘K' : 'Ctrl K'}</kbd>
				</button>
			</div>
		</header>
		{#if mobileSearchOpen && activeSpaceId}
			<div class="flex shrink-0 items-center gap-2 border-b border-gray-200 px-3 py-2 sm:hidden dark:border-gray-950/60">
				<div class="relative flex-1">
					<input
						type="search"
						enterkeyhint="search"
						placeholder="Search {activeSpace?.name ?? 'bookmarks'}"
						class="w-full rounded-md border-0 bg-gray-100 py-2 pl-8 pr-3 text-base text-gray-900 placeholder:text-gray-500 focus:ring-2 focus:ring-primary-500 dark:bg-gray-950/70 dark:text-white"
						bind:value={searchQuery}
						oninput={handleSearchInput}
						onkeydown={handleSearchKeydown}
						{@attach focusOnMount}
					/>
					<SearchOutline class="absolute left-2.5 top-3 h-4 w-4 text-gray-500" />
				</div>
				<button
					type="button"
					aria-label="Close search"
					class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800"
					onclick={() => (mobileSearchOpen = false)}
				>
					<CloseOutline class="h-5 w-5" />
				</button>
			</div>
		{/if}

		<main class="relative flex-1 overflow-y-auto overscroll-y-contain" data-scroll-root bind:this={scrollRoot}>
			<PullToRefresh root={scrollRoot} />
			<div class="mx-auto w-full max-w-6xl px-4 py-6 md:px-8">
				{#key lockState.version}
					{@render children()}
				{/key}
			</div>
		</main>
	</div>
</div>

<!-- Add Collection Modal -->
<Modal title="New Collection" bind:open={showAddModal} size="sm" autoclose={false}>
	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleCreateCollection();
		}}
		class="flex flex-col gap-4"
	>
		<div>
			<Label for="col-name" class="mb-2">Name</Label>
			<Input
				id="col-name"
				type="text"
				bind:value={newCollectionName}
				required
				maxlength={80}
				placeholder="e.g. Frontend Resources"
			/>
		</div>
		<div>
			<Label for="col-parent" class="mb-2">Parent Collection (optional)</Label>
			<Select id="col-parent" placeholder="" bind:value={newCollectionParentId}>
				<option value="">None (top level)</option>
				{#each parentOptions as opt (opt.id)}
					<option value={opt.id}>{'─'.repeat(opt.depth)} {opt.label}</option>
				{/each}
			</Select>
		</div>
		<div>
			<Toggle bind:checked={newCollectionPrivate} onchange={togglePrivate}>Private collection</Toggle>
			<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
				Only you and the members you pick can see it, not even space admins. Great for gift ideas.
			</p>
		</div>
		{#if newCollectionPrivate}
			<MemberChecklist
				members={spaceMembers}
				bind:selected={newCollectionMembers}
				currentUserId={session.me?.id}
			/>
		{/if}
		{#if createError}
			<p class="text-sm text-red-500 dark:text-red-400">{createError}</p>
		{/if}
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (showAddModal = false)}>Cancel</Button>
			<Button type="submit" disabled={creating || !newCollectionName.trim()}>{creating ? 'Creating...' : 'Create'}</Button>
		</div>
	</form>
</Modal>

<Modal title="Leave '{activeSpace?.name}'" bind:open={showLeaveModal} size="sm" autoclose={false}>
	<p class="text-gray-700 dark:text-gray-300">
		Are you sure you want to leave <strong>{activeSpace?.name}</strong>? You won't be able to rejoin unless you are
		invited again.
	</p>
	{#if leaveError}
		<p class="text-sm text-red-500 dark:text-red-400">{leaveError}</p>
	{/if}
	<div class="flex justify-end gap-2">
		<Button color="light" onclick={() => (showLeaveModal = false)}>Cancel</Button>
		<Button color="red" disabled={leaving} onclick={handleLeaveSpace}>Leave space</Button>
	</div>
</Modal>

<CreateOrJoinSpaceModal bind:open={showCreateOrJoin} ondone={handleSpaceDone} />

{#if activeSpace}
	<InvitePeopleModal
		bind:open={showInvite}
		spaceId={activeSpace.id}
		spaceName={activeSpace.name}
		myRole={activeSpace.role ?? 'VIEWER'}
	/>
{/if}

<QuickSwitcher bind:open={switcherOpen} items={switcherItems} />

{#if collectionMenu}
	{@const node = collectionMenu.node}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		role="menu"
		tabindex="-1"
		class="fixed z-50 w-52 rounded-lg bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
		style:left="{collectionMenu.x}px"
		style:top="{collectionMenu.y}px"
		onclick={(e) => e.stopPropagation()}
	>
		<p class="truncate px-2 pb-1 pt-0.5 text-xs font-semibold text-gray-400">{node.label}</p>
		{#if canCreateCollections}
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => openAddModal(node.id)}
			>
				New sub-collection <FolderPlusSolid class="h-4 w-4" />
			</button>
		{/if}
		{#if canEditCollections}
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => runCollectionAction(node, 'edit')}
			>
				Edit collection <PenOutline class="h-4 w-4" />
			</button>
		{/if}
		<button
			role="menuitem"
			class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
			onclick={() => runCollectionAction(node, 'access')}
		>
			Who can see this <LockOutline class="h-4 w-4" />
		</button>
		<button
			role="menuitem"
			class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
			onclick={() => runCollectionAction(node, 'share')}
		>
			Share publicly <GlobeOutline class="h-4 w-4" />
		</button>
		{#if node.locked}
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => unlockCollection(node.id!, node.label)}
			>
				Don't lock for me <EyeOutline class="h-4 w-4" />
			</button>
		{:else}
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				title="Hide it from your view until you unlock with your PIN; nobody else is affected"
				onclick={() => lockCollection(node.id!, node.label)}
			>
				Lock for me <EyeSlashOutline class="h-4 w-4" />
			</button>
		{/if}
		<button
			role="menuitem"
			class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
			onclick={() => copyCollectionLink(node)}
		>
			Copy link <LinkOutline class="h-4 w-4" />
		</button>
		{#if canDeleteCollections}
			<div class="my-1 h-px bg-gray-800"></div>
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-red-400 hover:bg-red-600 hover:text-white"
				onclick={() => runCollectionAction(node, 'delete')}
			>
				Delete collection <TrashBinOutline class="h-4 w-4" />
			</button>
		{/if}
	</div>
{/if}

<LockDialog
	bind:open={lockDialogOpen}
	mode={lockDialogMode}
	ondone={() => {
		const next = afterPinSetup;
		afterPinSetup = undefined;
		next?.();
	}}
/>

<Toasts />

<ProfileModal bind:open={showProfile} />

{#if session.me?.instanceAdmin}
	<AdminUsersModal bind:open={showAdminUsers} />
{/if}
