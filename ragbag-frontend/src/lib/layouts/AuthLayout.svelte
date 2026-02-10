<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Navbar, NavBrand, Button } from 'flowbite-svelte';
	import { logout } from '../api/client';
	import { navigate } from '../router';

	let { children }: { children: Snippet } = $props();

	async function handleLogout() {
		await logout();
		navigate('/login');
	}
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<Navbar>
		<NavBrand>
			<span class="self-center whitespace-nowrap text-xl font-semibold dark:text-white">Ragbag</span>
		</NavBrand>
		<div class="flex items-center">
			<Button size="sm" color="light" onclick={handleLogout}>Logout</Button>
		</div>
	</Navbar>

	{@render children()}
</div>
