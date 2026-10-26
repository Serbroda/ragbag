<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { SpacePermissionEnum } from 'ragbag-frontend-sdk';
	import { setContext } from 'svelte';
	import { Navbar, NavBrand, Button, Modal, Label, Input, Select } from 'flowbite-svelte';
	import {
		ArrowRightToBracketOutline,
		ChevronDownOutline,
		CirclePlusSolid,
		CogSolid,
		FolderPlusSolid,
		HomeSolid,
		PlusOutline,
		SearchOutline,
		UserAddSolid,
	} from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage, logout, logoutEverywhere } from '../api/client';
	import { navigate, route } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import DraggableTreeList from '../components/DraggableTreeList.svelte';
	import CreateOrJoinSpaceModal from '../components/CreateOrJoinSpaceModal.svelte';
	import InvitePeopleModal from '../components/InvitePeopleModal.svelte';
	import QuickSwitcher, { type SwitcherItem } from '../components/QuickSwitcher.svelte';
	import { collectionsToTree, expandToNode, flattenTree, type TreeNode } from '../components/tree';
	import { CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';
	import { loadMe, loadSpaces, rememberSpace, resetSession, session } from '../stores/spaces.svelte';
	import { colorFor, initials } from '../utils/format';

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

	function findLabel(nodes: TreeNode[], id: string): string | undefined {
		for (const node of nodes) {
			if (node.id === id) return node.label;
			const found = node.children && findLabel(node.children, id);
			if (found) return found;
		}
		return undefined;
	}

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

	const parentOptions = $derived(flattenTree(menuTree));

	function openAddModal() {
		if (!canCreateCollections) return;
		closeMenus();
		newCollectionName = '';
		// Default to the collection the user is currently looking at
		newCollectionParentId = activeCollectionId ?? '';
		createError = '';
		creating = false;
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

	const navTree = $derived<TreeNode[]>(
		activeSpaceId ? [{ label: 'Dashboard', icon: HomeSolid, href: `/space/${activeSpaceId}` }] : [],
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

<div class="min-h-screen bg-gray-50 dark:bg-gray-900" class:select-none={isResizing}>
	<!-- Top Bar -->
	<Navbar fluid class="border-b border-gray-200 px-3 dark:border-gray-800">
		<div class="flex items-center gap-2">
			<button
				type="button"
				aria-label="Toggle sidebar"
				class="inline-flex items-center rounded-lg p-2 text-sm text-gray-500 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-200 md:hidden dark:text-gray-400 dark:hover:bg-gray-700 dark:focus:ring-gray-600"
				onclick={() => (sidebarOpen = !sidebarOpen)}
			>
				<svg class="h-6 w-6" fill="currentColor" viewBox="0 0 20 20">
					<path
						fill-rule="evenodd"
						d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 15a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z"
						clip-rule="evenodd"
					/>
				</svg>
			</button>
			<NavBrand>
				<span class="brand-glow self-center whitespace-nowrap text-xl font-bold tracking-tight">Ragbag</span>
			</NavBrand>
		</div>
		<div class="flex flex-1 items-center justify-center px-4">
			{#if activeSpaceId}
				<div class="relative w-full max-w-md">
					<input
						type="text"
						placeholder="Search {activeSpace?.name ?? 'bookmarks'}..."
						class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-1.5 pl-9 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:border-primary-500 dark:focus:ring-primary-500"
						bind:value={searchQuery}
						oninput={handleSearchInput}
						onkeydown={handleSearchKeydown}
					/>
					<SearchOutline class="absolute left-2.5 top-2 h-4 w-4 text-gray-400" />
				</div>
			{/if}
		</div>
		<div class="flex items-center gap-1">
			<button
				type="button"
				class="hidden items-center gap-2 rounded-lg border border-gray-200 px-2.5 py-1 text-xs text-gray-500 hover:bg-gray-100 sm:flex dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-700"
				onclick={() => (switcherOpen = true)}
				title="Quick switcher"
			>
				Jump to… <kbd class="rounded bg-gray-100 px-1 font-sans dark:bg-gray-800">{isMac ? '⌘K' : 'Ctrl K'}</kbd>
			</button>
		</div>
	</Navbar>
	<div class="navbar-accent"></div>

	<!-- Mobile Backdrop -->
	{#if sidebarOpen}
		<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
		<div class="fixed inset-0 top-[61px] z-30 bg-gray-900/50 md:hidden" onclick={() => (sidebarOpen = false)}></div>
	{/if}

	<!-- Space Bar (always dark, like Discord) -->
	<aside
		class="fixed top-[61px] left-0 z-40 h-[calc(100vh-61px)] w-[72px] flex-col items-center overflow-y-auto bg-gray-900 py-3 dark:bg-gray-950 {sidebarOpen
			? 'flex'
			: 'hidden'} md:flex"
		aria-label="Spaces"
	>
		{#each session.spaces as space (space.id)}
			{@const active = activeSpaceId === space.id}
			<div class="group relative flex w-full items-center justify-center py-1">
				<span
					class="absolute left-0 w-1 rounded-r-full bg-white transition-all duration-200 {active
						? 'h-10'
						: 'h-0 group-hover:h-5'}"
				></span>
				<button
					type="button"
					title={space.name}
					aria-label={space.name}
					aria-current={active ? 'page' : undefined}
					class="flex h-12 w-12 shrink-0 items-center justify-center text-sm font-bold text-white transition-all duration-200 {active
						? `rounded-2xl ${colorFor(space.name)}`
						: 'rounded-3xl bg-gray-700 text-gray-200 hover:rounded-2xl hover:bg-primary-500'}"
					onclick={() => selectSpace(space.id)}
				>
					{initials(space.name)}
				</button>
				<!-- Tooltip -->
				<span
					class="pointer-events-none fixed left-[80px] z-50 hidden whitespace-nowrap rounded-md bg-gray-950 px-3 py-1.5 text-sm font-semibold text-white shadow-lg group-hover:md:block"
				>
					{space.name}
				</span>
			</div>
		{/each}

		<div class="mx-auto my-1 h-0.5 w-8 rounded bg-gray-700"></div>

		<div class="group relative flex w-full items-center justify-center py-1">
			<button
				type="button"
				title="Create or join a space"
				aria-label="Create or join a space"
				class="flex h-12 w-12 items-center justify-center rounded-3xl bg-gray-700 text-green-400 transition-all duration-200 hover:rounded-2xl hover:bg-green-500 hover:text-white"
				onclick={() => (showCreateOrJoin = true)}
			>
				<PlusOutline class="h-6 w-6" />
			</button>
		</div>
	</aside>

	<!-- Sidebar -->
	<aside
		class="fixed top-[61px] left-[72px] z-40 h-[calc(100vh-61px)] border-r border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800 {sidebarOpen
			? 'block'
			: 'hidden'} md:block"
		style="width: {sidebarWidth}px"
	>
		<div class="flex h-full flex-col">
			<!-- Space header with menu -->
			{#if activeSpace}
				<div class="relative border-b border-gray-200 dark:border-gray-700">
					<button
						type="button"
						class="flex h-12 w-full items-center justify-between px-4 text-left font-semibold text-gray-900 hover:bg-gray-100 dark:text-white dark:hover:bg-gray-700"
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
							class="absolute left-2 right-2 top-12 z-50 rounded-md bg-gray-950 p-1.5 text-sm text-gray-200 shadow-xl"
						>
							{#if canManageMembers}
								<button
									role="menuitem"
									class="flex w-full items-center justify-between rounded px-2 py-1.5 text-primary-400 hover:bg-primary-500 hover:text-white"
									onclick={() => ((showInvite = true), closeMenus())}
								>
									Invite people <UserAddSolid class="h-4 w-4" />
								</button>
							{/if}
							<button
								role="menuitem"
								class="flex w-full items-center justify-between rounded px-2 py-1.5 hover:bg-primary-500 hover:text-white"
								onclick={() => (closeMenus(), navigate('/space/:spaceId/settings', { params: { spaceId: activeSpace.id } }))}
							>
								Space settings <CogSolid class="h-4 w-4" />
							</button>
							{#if canCreateCollections}
								<button
									role="menuitem"
									class="flex w-full items-center justify-between rounded px-2 py-1.5 hover:bg-primary-500 hover:text-white"
									onclick={openAddModal}
								>
									Create collection <FolderPlusSolid class="h-4 w-4" />
								</button>
							{/if}
							<div class="my-1 h-px bg-gray-800"></div>
							<button
								role="menuitem"
								class="flex w-full items-center justify-between rounded px-2 py-1.5 text-red-400 hover:bg-red-500 hover:text-white"
								onclick={openLeaveModal}
							>
								Leave space <ArrowRightToBracketOutline class="h-4 w-4" />
							</button>
						</div>
					{/if}
				</div>
			{/if}

			<div class="flex-1 overflow-y-auto">
				<nav>
					<ul class="space-y-0.5 py-2">
						{#each navTree as node (node.label)}
							<TreeItem {node} />
						{/each}
					</ul>
				</nav>
				<nav class="border-t border-gray-200 dark:border-gray-700">
					<div class="flex items-center justify-between px-3 pt-3 pb-1">
						<h3 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
							Collections
						</h3>
						{#if canCreateCollections}
							<button
								type="button"
								title="Create collection"
								aria-label="Create collection"
								class="rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-300"
								onclick={(e) => (e.stopPropagation(), openAddModal())}
							>
								<CirclePlusSolid class="h-3.5 w-3.5" />
							</button>
						{/if}
					</div>
					{#if menuTree.length === 0 && activeSpace}
						<p class="px-4 py-2 text-xs text-gray-400 dark:text-gray-500">
							{canCreateCollections ? 'No collections yet. Create one with +.' : 'No collections yet.'}
						</p>
					{/if}
					<DraggableTreeList bind:nodes={menuTree} onmove={handleMove} draggable={canEditCollections} />
				</nav>
			</div>

			<!-- User panel -->
			<div class="relative flex h-14 shrink-0 items-center gap-2 border-t border-gray-200 bg-gray-100 px-2 dark:border-gray-700 dark:bg-gray-900/60">
				{#if session.me}
					<span
						class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold text-white {colorFor(
							session.me.username,
						)}"
					>
						{initials(session.me.username)}
					</span>
					<div class="min-w-0 flex-1 leading-tight">
						<p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{session.me.username}</p>
						{#if activeSpace?.role}
							<p class="truncate text-xs capitalize text-gray-500 dark:text-gray-400">
								{activeSpace.role.toLowerCase()}
							</p>
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
					class="rounded p-1.5 text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200"
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
						class="absolute bottom-14 left-2 right-2 z-50 rounded-md bg-gray-950 p-1.5 text-sm text-gray-200 shadow-xl"
					>
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded px-2 py-1.5 hover:bg-primary-500 hover:text-white"
							onclick={() => handleLogout(false)}
						>
							Log out <ArrowRightToBracketOutline class="h-4 w-4" />
						</button>
						<button
							role="menuitem"
							class="flex w-full items-center justify-between rounded px-2 py-1.5 text-red-400 hover:bg-red-500 hover:text-white"
							onclick={() => handleLogout(true)}
						>
							Log out on all devices
						</button>
					</div>
				{/if}
			</div>
		</div>
	</aside>

	<!-- Resize Handle (desktop only) -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed top-[61px] z-50 hidden h-[calc(100vh-61px)] w-1.5 cursor-col-resize md:block"
		class:bg-primary-500={isResizing}
		style:left="{SPACE_BAR_WIDTH + sidebarWidth - 3}px"
		onmousedown={startResize}
	>
		<div class="h-full w-full hover:bg-gray-300 dark:hover:bg-gray-600" class:!bg-primary-500={isResizing}></div>
	</div>

	<!-- Main Content -->
	<div class="p-4" style:--sidebar-w="{SPACE_BAR_WIDTH + sidebarWidth}px">
		<div class="md-sidebar-offset">
			{@render children()}
		</div>
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

<style>
	@media (min-width: 768px) {
		.md-sidebar-offset {
			margin-left: var(--sidebar-w);
		}
	}
</style>
