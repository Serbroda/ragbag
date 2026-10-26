<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import type { Snippet } from 'svelte';
	import { LinkOutline } from 'flowbite-svelte-icons';
	import { getDomain, safeUrl } from '../utils/format';

	/** Compact launcher tile for dashboards, in the spirit of homelab start pages. */
	let { bookmark, overlay }: { bookmark: BookmarkDto; overlay?: Snippet } = $props();

	const href = $derived(safeUrl(bookmark.url));
	let faviconFailed = $state(false);
	const favicon = $derived(faviconFailed ? undefined : safeUrl(bookmark.favicon));
</script>

<div
	class="group relative flex h-full items-center gap-3 rounded-xl border border-gray-200 bg-white p-3 transition duration-200 hover:-translate-y-0.5 hover:border-gray-300 hover:shadow-lg dark:border-gray-800 dark:bg-gray-900 dark:hover:border-gray-700 dark:hover:shadow-black/30"
>
	<div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800">
		{#if favicon}
			<img
				src={favicon}
				alt=""
				loading="lazy"
				referrerpolicy="no-referrer"
				class="h-6 w-6 rounded"
				onerror={() => (faviconFailed = true)}
			/>
		{:else}
			<LinkOutline class="h-5 w-5 text-gray-400" />
		{/if}
	</div>
	<div class="min-w-0 flex-1">
		<a
			{href}
			target="_blank"
			rel="noopener noreferrer"
			class="line-clamp-1 text-sm font-semibold text-gray-900 after:absolute after:inset-0 after:content-[''] group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400"
			title={bookmark.title || bookmark.url}
		>
			{bookmark.title || bookmark.url}
		</a>
		<p class="truncate text-xs text-gray-500">{getDomain(bookmark.url)}</p>
	</div>
	{#if overlay}
		<div class="relative z-10">{@render overlay()}</div>
	{/if}
</div>
