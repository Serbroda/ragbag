<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Navbar, NavBrand, Button, Sidebar, SidebarWrapper } from 'flowbite-svelte';
	import { logout } from '../api/client';
	import { navigate } from '../router';
	import TreeItem from '../components/TreeItem.svelte';
	import type { TreeNode } from '../components/tree';

	let { children }: { children: Snippet } = $props();

	let sidebarOpen = $state(false);

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

	const menuTree: TreeNode[] = [
		{
			label: 'Dashboard',
			href: '/',
		},
		{
			label: 'Shop',
			href: '/shop',
			expanded: true,
			children: [
				{ label: 'Products', href: '/shop/products' },
				{ label: 'Orders', href: '/shop/orders' },
			],
		},
		{
			label: 'Vegetables',
			href: '/vegetables',
			children: [
				{ label: 'Fruits', href: '/vegetables/fruits' },
				{ label: 'Roots', href: '/vegetables/roots' },
			],
		},
		{
			label: 'Tech',
			href: '/tech',
			children: [
				{
					label: 'Monitors',
					href: '/tech/monitors',
					children: [
						{ label: 'Flatscreen', href: '/tech/monitors/flatscreen' },
						{ label: 'LCD', href: '/tech/monitors/lcd' },
					],
				},
				{ label: 'Keyboards', href: '/tech/keyboards' },
			],
		},
		{
			label: 'Spaces',
			href: '/spaces',
		},
		{
			label: 'Bookmarks',
			href: '/bookmarks',
		},
	];

	const bottomTree: TreeNode[] = [
		{ label: 'Settings', href: '/settings' },
	];
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
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
					<path fill-rule="evenodd" d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 15a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clip-rule="evenodd" />
				</svg>
			</button>
			<NavBrand>
				<span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white">Ragbag</span>
			</NavBrand>
		</div>
		<div class="flex items-center">
			<Button size="sm" color="light" onclick={handleLogout}>Logout</Button>
		</div>
	</Navbar>

	<!-- Sidebar + Content -->
	<Sidebar
		isOpen={sidebarOpen}
		{closeSidebar}
		breakpoint="md"
		alwaysOpen
		position="fixed"
		class="top-[61px] z-40 h-[calc(100vh-61px)]"
		backdrop={false}
		params={{ x: -50, duration: 50 }}
	>
		<SidebarWrapper class="flex h-full flex-col justify-between overflow-y-auto">
			<nav>
				<ul class="space-y-0.5 py-2">
					{#each menuTree as node (node.label)}
						<TreeItem {node} />
					{/each}
				</ul>
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

	<!-- Main Content -->
	<div class="p-4 md:ml-64">
		{@render children()}
	</div>
</div>
