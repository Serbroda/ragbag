<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Navbar, NavBrand, Button, Sidebar, SidebarGroup, SidebarItem, SidebarWrapper } from 'flowbite-svelte';
	import { logout } from '../api/client';
	import { navigate } from '../router';

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
	>
		<SidebarWrapper class="h-full">
			<SidebarGroup>
				<SidebarItem label="Dashboard" href="/" />
				<SidebarItem label="Spaces" href="/spaces" />
				<SidebarItem label="Bookmarks" href="/bookmarks" />
			</SidebarGroup>
			<SidebarGroup border>
				<SidebarItem label="Settings" href="/settings" />
			</SidebarGroup>
		</SidebarWrapper>
	</Sidebar>

	<!-- Main Content -->
	<div class="p-4 md:ml-64">
		{@render children()}
	</div>
</div>
