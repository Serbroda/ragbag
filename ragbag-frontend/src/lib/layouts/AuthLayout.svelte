<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Navbar, NavBrand, Button, Sidebar, SidebarWrapper } from 'flowbite-svelte';
	import { HomeSolid, GlobeSolid, StarSolid, TagSolid } from 'flowbite-svelte-icons';
	import {apiConfig, logout} from '../api/client';
	import { navigate } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import DraggableTreeList from '../components/DraggableTreeList.svelte';
	import {collectionsToTree, type TreeNode} from '../components/tree';
	import {CollectionApi, SpaceApi} from "ragbag-frontend-sdk";

	let { children }: { children: Snippet } = $props();

	let sidebarOpen = $state(false);

	// --- Resize ---
	const MIN_WIDTH = 200;
	const MAX_WIDTH = 480;
	const DEFAULT_WIDTH = 256;

	let sidebarWidth = $state(DEFAULT_WIDTH);
	let isResizing = $state(false);

	function startResize(e: MouseEvent) {
		e.preventDefault();
		isResizing = true;

		const onMouseMove = (e: MouseEvent) => {
			sidebarWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, e.clientX));
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

	const spaceApi = new SpaceApi(apiConfig());
	const collectionApi = new CollectionApi(apiConfig());

	let menuTree = $state<TreeNode[]>([]);

	async function loadCollections() {
		const spaces = await spaceApi.getSpaces();
		const collections = await collectionApi.getCollections({ spaceId: spaces[0].id });
		menuTree = collectionsToTree(collections);
	}

	loadCollections();

	const bottomTree: TreeNode[] = [{ label: 'Settings', href: '/settings' }];

	async function handleMove(movedNode: TreeNode, newParent: TreeNode | null) {
		await collectionApi.moveCollection({
			collectionId: movedNode.id!,
			moveCollectionDto: {
				parentId: newParent?.id ?? undefined,
			}
		});
		console.log(`Moved "${movedNode.label}" ${newParent ? `into "${newParent.label}"` : 'to root'}`);
	}
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900" class:select-none={isResizing}>
	<!-- Top Bar -->
	<Navbar class="border-b border-gray-200 dark:border-gray-700">
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
				<span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white"
					>Ragbag</span
				>
			</NavBrand>
		</div>
		<div class="flex items-center">
			<Button size="sm" color="light" onclick={handleLogout}>Logout</Button>
		</div>
	</Navbar>

	<!-- Sidebar -->
	<Sidebar
		isOpen={sidebarOpen}
		{closeSidebar}
		breakpoint="md"
		position="fixed"
		class="top-[61px] z-40 h-[calc(100vh-61px)]"
		params={{ x: -50, duration: 50 }}
		style="width: {sidebarWidth}px"
		divClass="h-full overflow-hidden bg-gray-50 dark:bg-gray-800"
	>
		<SidebarWrapper class="flex h-full flex-col overflow-y-auto">
			<nav>
				<ul class="space-y-0.5 py-2">
					<TreeItem node={{ label: 'Dashboard', icon: HomeSolid }} />
					<TreeItem node={{ label: 'Links', icon: GlobeSolid }} />
					<TreeItem node={{ label: 'Favorites', icon: StarSolid }} />
					<TreeItem node={{ label: 'Tags', icon: TagSolid }} />
				</ul>
			</nav>
			<nav class="border-t border-gray-200 dark:border-gray-700">
				<DraggableTreeList
						bind:nodes={menuTree}
						onmove={handleMove}
						draggable={true}
				/>
			</nav>
			<nav class="border-t border-gray-200 dark:border-gray-700">
				<ul class="space-y-0.5 py-2">
					{#each bottomTree as node (node.label)}
						<TreeItem {node} />
					{/each}
				</ul>
			</nav>
		</SidebarWrapper>
	</Sidebar>

	<!-- Resize Handle (desktop only) -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed top-[61px] z-50 hidden h-[calc(100vh-61px)] w-1.5 cursor-col-resize md:block"
		class:bg-primary-500={isResizing}
		style:left="{sidebarWidth - 3}px"
		onmousedown={startResize}
	>
		<div
			class="h-full w-full hover:bg-gray-300 dark:hover:bg-gray-600"
			class:!bg-primary-500={isResizing}
		></div>
	</div>

	<!-- Main Content -->
	<div class="p-4" style:margin-left="0" style:--sidebar-w="{sidebarWidth}px">
		<div class="md-sidebar-offset">
			{@render children()}
		</div>
	</div>
</div>

<style>
	@media (min-width: 768px) {
		.md-sidebar-offset {
			margin-left: var(--sidebar-w);
		}
	}
</style>
