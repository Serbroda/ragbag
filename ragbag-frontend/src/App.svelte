<script lang="ts">
	import { Router } from 'sv-router';
	import { isAuthenticated, refreshAccessToken } from './lib/api/client';
	import './lib/router';

	let ready = $state(false);

	async function init() {
		if (isAuthenticated()) {
			await refreshAccessToken();
		}
		ready = true;
	}

	init();
</script>

{#if ready}
	<Router />
{:else}
	<div class="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	</div>
{/if}
