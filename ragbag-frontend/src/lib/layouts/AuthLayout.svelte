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
		CogSolid,
		FolderOutline,
		FolderPlusSolid,
		HomeSolid,
		PlusOutline,
		SearchOutline,
		StarSolid,
		UserAddSolid,
		UserCircleOutline,
		UsersOutline,
	} from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage, logout, logoutEverywhere } from '../api/client';
	import { navigate, p, route } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import DraggableTreeList from '../components/DraggableTreeList.svelte';
	import CreateOrJoinSpaceModal from '../components/CreateOrJoinSpaceModal.svelte';
	import InvitePeopleModal from '../components/InvitePeopleModal.svelte';
	import QuickSwitcher, { type SwitcherItem } from '../components/QuickSwitcher.svelte';
	import Logo from '../components/Logo.svelte';
	import Avatar from '../components/Avatar.svelte';
	import MemberChecklist from '../components/MemberChecklist.svelte';
	import ProfileModal from '../components/ProfileModal.svelte';
	import AdminUsersModal from '../components/AdminUsersModal.svelte';
	import { collectionsToTree, expandToNode, flattenTree, pathToNode, type TreeNode } from '../components/tree';
	import { CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { SpaceMemberDto } from 'ragbag-frontend-sdk';
	import { loadPreferences } from '../stores/preferences.svelte';
	import { loadMe, loadSpaces, rememberSpace, resetSession, session } from '../stores/spaces.svelte';
	import { initials } from '../utils/format';

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
			const tree = collectionsToTree(collections, spaceId);
			if (activeCollectionId) {
				expandToNode(tree, activeCollectionId);
			}
			menuTree = tree;
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

	$effect(() => {
		if (session.me) loadPreferences(session.me.id);
	});

	$effect(() => {
		if (activeSpaceId) {
			loadCollections(activeSpaceId);
		}
	});

	// Leave spaces the user is no longer a member of (removed, deleted, bad link)
	$effect(() => {
		if (!spacesReady || !activeSpaceId) return;
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

	setContext('open-invite', () => (showInvite = true));

	const isMac = /mac/i.test(navigator.platform || navigator.userAgent);

	setContext('collection-name', (collectionId: string) => findLabel(menuTree, collectionId));

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

	// Reset the search box when switching spaces; prefill it on the search page
	$effect(() => {
		activeSpaceId;
		const q = route.search.q;
		searchQuery = route.pathname.endsWith('/search') && typeof q === 'string' ? q : '';
	});

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
	const MIN_WIDTH = 200;
	const MAX_WIDTH = 480;
	const DEFAULT_WIDTH = 256;

	let sidebarWidth = $state(DEFAULT_WIDTH);
	let isResizing = $state(false);

	function startResize(e: MouseEvent) {
		e.preventDefault();
		isResizing = true;

		const onMouseMove = (e: MouseEvent) => {
			sidebarWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, e.clientX - SPACE_BAR_WIDTH));
		};

		const onMouseUp = () => {
			isResizing = false;
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
	}

	function onWindowKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
			e.preventDefault();
			switcherOpen = !switcherOpen;
		} else if (e.key === 'Escape') {
			closeMenus();
		}
	}

	const switcherItems = $derived<SwitcherItem[]>([
		...session.spaces.map((space) => ({
			id: space.id,
			label: space.name,
			hint: 'Space',
			kind: 'space' as const,
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

	function openAddModal() {
		if (!canCreateCollections) return;
		closeMenus();
		newCollectionName = '';
		// Default to the collection the user is currently looking at
		newCollectionParentId = activeCollectionId ?? '';
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
				]
			: [],
	);

	async function handleMove(movedNode: TreeNode, newParent: TreeNode | null) {
		if (!canEditCollections) return;
		await collectionApi.moveCollection({
			collectionId: movedNode.id!,
			moveCollectionDto: {
				parentId: newParent?.id ?? undefined,
			},
		});
		if (activeSpaceId) {
			await loadCollections(activeSpaceId);
		}
	}
</script>

<svelte:window onkeydown={onWindowKeydown} onclick={closeMenus} />

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
		<nav
			class="flex w-[72px] shrink-0 flex-col items-center gap-2 overflow-y-auto bg-gray-950 py-3"
			aria-label="Spaces"
		>
			<button
				type="button"
				title="Ragbag"
				aria-label="Ragbag home"
				class="transition-transform duration-200 hover:scale-105"
				onclick={() => navigate('/')}
			>
				<Logo size={48} />
			</button>
			<div class="h-0.5 w-8 rounded bg-gray-800"></div>

			{#each session.spaces as space (space.id)}
				{@const active = activeSpaceId === space.id}
				<div class="group relative flex w-full items-center justify-center">
					<span
						class="absolute left-0 w-1 rounded-r-full bg-white transition-all duration-200 {active
							? 'h-10'
							: 'h-0 group-hover:h-5'}"
					></span>
					<button
						type="button"
						aria-label={space.name}
						aria-current={active ? 'page' : undefined}
						class="h-12 w-12 shrink-0 overflow-hidden transition-all duration-200 {active
							? 'rounded-2xl shadow-lg'
							: 'rounded-3xl hover:rounded-2xl'}"
						onclick={() => selectSpace(space.id)}
					>
						{#if space.iconId || active}
							<Avatar name={space.name} imageId={space.iconId} class="h-full w-full text-sm" />
						{:else}
							<span
								class="flex h-full w-full items-center justify-center bg-gray-800 text-sm font-semibold text-gray-300 transition-colors hover:bg-primary-500 hover:text-white"
							>
								{initials(space.name)}
							</span>
						{/if}
					</button>
					<span
						class="pointer-events-none fixed left-[80px] z-50 hidden whitespace-nowrap rounded-md bg-gray-950 px-3 py-1.5 text-sm font-semibold text-white shadow-xl ring-1 ring-white/10 md:group-hover:block"
					>
						{space.name}
					</span>
				</div>
			{/each}

			<div class="group relative flex w-full items-center justify-center">
				<button
					type="button"
					aria-label="Create or join a space"
					class="flex h-12 w-12 items-center justify-center rounded-3xl bg-gray-800 text-green-400 transition-all duration-200 hover:rounded-2xl hover:bg-green-600 hover:text-white"
					onclick={() => (showCreateOrJoin = true)}
				>
					<PlusOutline class="h-6 w-6" />
				</button>
				<span
					class="pointer-events-none fixed left-[80px] z-50 hidden whitespace-nowrap rounded-md bg-gray-950 px-3 py-1.5 text-sm font-semibold text-white shadow-xl ring-1 ring-white/10 md:group-hover:block"
				>
					Create or join a space
				</span>
			</div>
		</nav>

		<!-- Sidebar -->
		<aside
			class="relative flex h-full flex-col bg-gray-50 dark:bg-gray-900"
			style:width="{sidebarWidth}px"
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
						<span class="truncate">{activeSpace.name}</span>
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
									onclick={openAddModal}
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
				<button
					type="button"
					class="flex items-center gap-2 rounded-md bg-gray-100 px-2.5 py-1.5 text-xs text-gray-500 transition-colors hover:text-gray-800 dark:bg-gray-950/70 dark:text-gray-400 dark:hover:text-gray-200"
					onclick={() => (switcherOpen = true)}
					title="Quick switcher"
				>
					<span class="hidden lg:inline">Jump to…</span>
					<kbd class="font-sans font-semibold">{isMac ? '⌘K' : 'Ctrl K'}</kbd>
				</button>
			</div>
		</header>

		<main class="flex-1 overflow-y-auto" data-scroll-root>
			<div class="mx-auto w-full max-w-6xl px-4 py-6 md:px-8">
				{@render children()}
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

<ProfileModal bind:open={showProfile} />

{#if session.me?.instanceAdmin}
	<AdminUsersModal bind:open={showAdminUsers} />
{/if}
