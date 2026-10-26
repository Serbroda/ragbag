<script lang="ts">
	import { Router } from 'sv-router';
	import { isAuthenticated, refreshAccessToken } from './lib/api/client';
	import './lib/router';

	let ready = $state(false);
	// sv-router strips the base from the path before matching, so a base of '/' would turn
	// '/login' into 'login' and match nothing. Only pass a real prefix like '/app'.
	const configuredBase = import.meta.env.VITE_APP_BASE?.replace(/\/+$/, '');
	const routerBase =
		configuredBase || (globalThis.location?.pathname?.startsWith('/app') ? '/app' : undefined);

	async function init() {
		if (isAuthenticated()) {
			await refreshAccessToken();
		}
		ready = true;
	}

	init();
</script>

{#if ready}
	<Router base={routerBase} />
{:else}
	<div class="flex min-h-dvh items-center justify-center bg-gray-950">
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	</div>
{/if}
