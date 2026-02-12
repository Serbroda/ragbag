<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { SpaceDto } from 'ragbag-frontend-sdk';
	import { setContext } from 'svelte';
	import { Navbar, NavBrand, Button, Modal, Label, Input, Select } from 'flowbite-svelte';
	import { HomeSolid, GlobeSolid, StarSolid, TagSolid, CirclePlusSolid } from 'flowbite-svelte-icons';
	import { apiConfig, logout } from '../api/client';
	import { navigate, route } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import DraggableTreeList from '../components/DraggableTreeList.svelte';
	import { collectionsToTree, expandToNode, flattenTree, type TreeNode } from '../components/tree';
	import { CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';

	let { children }: { children: Snippet } = $props();

	let sidebarOpen = $state(false);

	// --- Resize ---
	const SPACE_BAR_WIDTH = 64;
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

	// --- Navigation ---
	function toggleSidebar() {
		sidebarOpen = !sidebarOpen;
	}

	function closeSidebar() {
		sidebarOpen = false;
	}

	async function handleLogout() {
		await logout();
		navigate('/login');
	}

	// --- Spaces & Collections ---
	const spaceApi = new SpaceApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());

	let spaces = $state<SpaceDto[]>([]);
	let menuTree = $state<TreeNode[]>([]);

	const activeSpaceId = $derived(route.params.spaceId ?? null);
	const activeCollectionId = $derived(route.params.collectionId ?? null);

	async function loadSpaces() {
		spaces = await spaceApi.getSpaces();
	}

	async function loadCollections(spaceId: string) {
		const collections = await collectionApi.getCollections({ spaceId });
		const tree = collectionsToTree(collections, spaceId);
		if (activeCollectionId) {
			expandToNode(tree, activeCollectionId);
		}
		menuTree = tree;
	}

	function selectSpace(spaceId: string) {
		navigate('/space/:spaceId', { params: { spaceId } });
	}

	loadSpaces();

	$effect(() => {
		if (activeSpaceId) {
			loadCollections(activeSpaceId);
		}
	});

	setContext('reload-collections', () => {
		if (activeSpaceId) loadCollections(activeSpaceId);
	});

	// --- Add Collection Modal ---
	let showAddModal = $state(false);
	let newCollectionName = $state('');
	let newCollectionParentId = $state('');
	let createError = $state('');
	let creating = $state(false);

	const parentOptions = $derived(flattenTree(menuTree));

	function openAddModal() {
		newCollectionName = '';
		newCollectionParentId = '';
		createError = '';
		creating = false;
		showAddModal = true;
	}

	async function handleCreateCollection() {
		const name = newCollectionName.trim();
		if (!name || !activeSpaceId) return;
		createError = '';
		creating = true;
		try {
			await collectionApi.createCollection({
				spaceId: activeSpaceId,
				createCollectionDto: {
					name,
					parentId: newCollectionParentId || undefined,
				},
			});
			await loadCollections(activeSpaceId);
			showAddModal = false;
		} catch {
			createError = 'Failed to create collection.';
		} finally {
			creating = false;
		}
	}

	const bottomTree: TreeNode[] = [{ label: 'Settings', href: '/settings' }];

	async function handleMove(movedNode: TreeNode, newParent: TreeNode | null) {
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

	function getInitials(name: string): string {
		return name
			.split(/\s+/)
			.map((w) => w[0])
			.join('')
			.toUpperCase()
			.slice(0, 2);
	}
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900" class:select-none={isResizing}>
	<!-- Top Bar -->
	<Navbar fluid class="border-b border-gray-200 px-3 dark:border-gray-800">
		<div class="flex items-center gap-2">
			<button
				type="button"
				aria-label="Toggle sidebar"
				class="inline-flex items-center rounded-lg p-2 text-sm text-gray-500 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-gray-200 md:hidden dark:text-gray-400 dark:hover:bg-gray-700 dark:focus:ring-gray-600"
				onclick={toggleSidebar}
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
				<span class="brand-glow self-center whitespace-nowrap text-xl font-bold tracking-tight"
					>Ragbag</span
				>
			</NavBrand>
		</div>
		<div class="flex items-center">
			<Button size="sm" color="light" onclick={handleLogout}>Logout</Button>
		</div>
	</Navbar>
	<div class="navbar-accent"></div>

	<!-- Mobile Backdrop -->
	{#if sidebarOpen}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="fixed inset-0 top-[61px] z-30 bg-gray-900/50 md:hidden"
			onclick={closeSidebar}
		></div>
	{/if}

	<!-- Space Bar (always dark, like Discord) -->
	<aside
		class="fixed top-[61px] left-0 z-40 h-[calc(100vh-61px)] w-16 flex-col items-center overflow-y-auto bg-gray-900 py-3 dark:bg-gray-950 {sidebarOpen ? 'flex' : 'hidden'} md:flex"
	>
		<!-- Home button -->
		<div class="relative flex w-full items-center justify-center pb-2">
			<button
				type="button"
				title="Home"
				class="flex h-10 w-10 items-center justify-center rounded-2xl bg-gray-700 text-primary-400 transition-all duration-200 hover:rounded-xl hover:bg-primary-500 hover:text-white"
				onclick={() => activeSpaceId && navigate('/space/:spaceId', { params: { spaceId: activeSpaceId } })}
			>
				<HomeSolid class="h-5 w-5" />
			</button>
		</div>
		<div class="mx-auto mb-2 h-px w-8 bg-gray-700"></div>

		<!-- Spaces -->
		{#each spaces as space (space.id)}
			<div class="relative flex w-full items-center justify-center py-1">
				<!-- Active indicator pill -->
				<span
					class="absolute left-0 w-1 rounded-r-full bg-white transition-all duration-200 {activeSpaceId === space.id ? 'h-5' : 'h-0 opacity-0'}"
				></span>
				<button
					type="button"
					title={space.name}
					class="flex h-10 w-10 shrink-0 items-center justify-center text-xs font-bold transition-all duration-200 {activeSpaceId === space.id
						? 'rounded-xl bg-primary-500 text-white shadow-lg shadow-primary-500/30'
						: 'rounded-2xl bg-gray-700 text-gray-300 hover:rounded-xl hover:bg-primary-500/80 hover:text-white'}"
					onclick={() => selectSpace(space.id)}
				>
					{getInitials(space.name)}
				</button>
			</div>
		{/each}
	</aside>

	<!-- Sidebar -->
	<aside
		class="fixed top-[61px] left-16 z-40 h-[calc(100vh-61px)] overflow-hidden border-r border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800 {sidebarOpen ? 'block' : 'hidden'} md:block"
		style="width: {sidebarWidth}px"
	>
		<div class="flex h-full flex-col overflow-y-auto">
			<nav>
				<ul class="space-y-0.5 py-2">
					<TreeItem node={{ label: 'Dashboard', icon: HomeSolid }} />
					<TreeItem node={{ label: 'Links', icon: GlobeSolid }} />
					<TreeItem node={{ label: 'Favorites', icon: StarSolid }} />
					<TreeItem node={{ label: 'Tags', icon: TagSolid }} />
				</ul>
			</nav>
			<nav class="border-t border-gray-200 dark:border-gray-700">
				<div class="flex items-center justify-between px-3 pt-3 pb-1">
					<h3
						class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400"
					>
						Collections
					</h3>
					<button
						type="button"
						title="Add collection"
						class="rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-600 dark:hover:text-gray-300"
						onclick={openAddModal}
					>
						<CirclePlusSolid class="h-3.5 w-3.5" />
					</button>
				</div>
				<DraggableTreeList bind:nodes={menuTree} onmove={handleMove} draggable={true} />
			</nav>
			<nav class="border-t border-gray-200 dark:border-gray-700">
				<ul class="space-y-0.5 py-2">
					{#each bottomTree as node (node.label)}
						<TreeItem {node} />
					{/each}
				</ul>
			</nav>
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
		<div
			class="h-full w-full hover:bg-gray-300 dark:hover:bg-gray-600"
			class:!bg-primary-500={isResizing}
		></div>
	</div>

	<!-- Main Content -->
	<div class="p-4" style:margin-left="0" style:--sidebar-w="{SPACE_BAR_WIDTH + sidebarWidth}px">
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
			<Input id="col-name" type="text" bind:value={newCollectionName} required placeholder="e.g. Frontend Resources" />
		</div>
		<div>
			<Label for="col-parent" class="mb-2">Parent Collection (optional)</Label>
			<Select id="col-parent" bind:value={newCollectionParentId}>
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
			<Button type="submit" disabled={creating}>{creating ? 'Creating...' : 'Create'}</Button>
		</div>
	</form>
</Modal>

<style>
	@media (min-width: 768px) {
		.md-sidebar-offset {
			margin-left: var(--sidebar-w);
		}
	}
</style>
