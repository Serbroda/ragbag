<script lang="ts">
	import type { LegalPageDto } from 'ragbag-frontend-sdk';
	import { ArrowLeftOutline } from 'flowbite-svelte-icons';
	import { errorMessage, getLegalPage, isAuthenticated } from '../api/client';
	import { p, route } from '../router';
	import Logo from '../components/Logo.svelte';
	import LegalLinks from '../components/LegalLinks.svelte';

	/** Imprint, privacy policy or terms of use: Markdown files of the operator, rendered by the server. */
	const page = $derived(route.params.page ?? '');

	let legal = $state<LegalPageDto | null>(null);
	let error = $state('');

	$effect(() => {
		load(page);
	});

	async function load(name: string) {
		error = '';
		try {
			const result = await getLegalPage(name);
			if (name === page) legal = result;
		} catch (err) {
			legal = null;
			error = await errorMessage(err, 'This page does not exist.');
		}
	}
</script>

<svelte:head>
	<title>{legal ? `${legal.title} · Ragbag` : 'Ragbag'}</title>
</svelte:head>

<div class="min-h-dvh bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100">
	<header class="border-b border-gray-200 dark:border-gray-950/60">
		<div class="mx-auto flex h-12 w-full max-w-3xl items-center gap-3 px-4 md:px-8">
			<a
				href={isAuthenticated() ? p('/') : p('/login')}
				class="flex items-center gap-2 rounded-md px-1 py-1 text-sm text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
				title="Back to Ragbag"
			>
				<ArrowLeftOutline class="h-4 w-4" />
				<Logo size={24} />
			</a>
		</div>
	</header>
	<main class="mx-auto w-full max-w-3xl px-4 py-8 md:px-8">
		{#if legal}
			<h1 class="mb-6 text-2xl font-bold tracking-tight">{legal.title}</h1>
			<!-- Rendered by the server from the operator's Markdown, without raw HTML -->
			<div class="legal">{@html legal.html}</div>
		{:else if error}
			<p class="text-gray-500 dark:text-gray-400">{error}</p>
		{/if}
		<LegalLinks class="mt-12 border-t border-gray-200 pt-4 text-xs text-gray-500 dark:border-gray-800" />
	</main>
</div>
