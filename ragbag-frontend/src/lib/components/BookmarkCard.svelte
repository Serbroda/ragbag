<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import type { Snippet } from 'svelte';
	import { Card } from 'flowbite-svelte';
	import { LinkOutline } from 'flowbite-svelte-icons';
	import { getDomain, safeUrl, timeAgo } from '../utils/format';

	let {
		bookmark,
		collectionName,
		collectionHref,
		actions,
	}: {
		bookmark: BookmarkDto;
		/** Shown in the meta line when the card is listed outside its collection. */
		collectionName?: string;
		collectionHref?: string;
		actions?: Snippet;
	} = $props();

	const href = $derived(safeUrl(bookmark.url));
	let faviconFailed = $state(false);
	const favicon = $derived(faviconFailed ? undefined : safeUrl(bookmark.favicon));
</script>

<Card
	class="relative w-full max-w-none p-4 hover:border-gray-400 md:w-[calc(50%-0.5rem)] lg:w-[calc(33.333%-0.75rem)] {actions
		? 'pb-12'
		: ''}"
>
	<div class="flex flex-col gap-2">
		<div class="flex items-start gap-4">
			{#if favicon}
				<img
					src={favicon}
					alt=""
					loading="lazy"
					referrerpolicy="no-referrer"
					class="mt-1 h-6 w-6 shrink-0 rounded"
					onerror={() => (faviconFailed = true)}
				/>
			{:else}
				<div class="mt-1 flex h-6 w-6 shrink-0 items-center justify-center rounded bg-gray-100 dark:bg-gray-700">
					<LinkOutline class="h-4 w-4 text-gray-400" />
				</div>
			{/if}

			<div class="min-w-0 flex-1">
				<a
					{href}
					target="_blank"
					rel="noopener noreferrer"
					class="line-clamp-2 block break-words font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400"
				>
					{bookmark.title || bookmark.url}
				</a>
				<p class="truncate text-xs text-gray-400 dark:text-gray-500">{getDomain(bookmark.url)}</p>
			</div>
		</div>

		{#if bookmark.description}
			<div class="line-clamp-2 break-words text-sm text-gray-500 dark:text-gray-400">
				{bookmark.description}
			</div>
		{/if}

		{#if bookmark.tags && bookmark.tags.length > 0}
			<div class="flex flex-wrap gap-1">
				{#each bookmark.tags as tag (tag)}
					<span
						class="rounded-full border border-gray-200 px-2 py-0.5 text-xs text-gray-600 dark:border-gray-700 dark:text-gray-300"
					>
						{tag}
					</span>
				{/each}
			</div>
		{/if}

		{#if bookmark.createdBy || bookmark.createdAt || collectionName}
			<p class="text-xs text-gray-400 dark:text-gray-500">
				{#if bookmark.createdBy}added by <span class="font-medium">{bookmark.createdBy}</span>{/if}
				{#if bookmark.createdAt}
					<time datetime={bookmark.createdAt.toISOString()} title={bookmark.createdAt.toLocaleString()}>
						{timeAgo(bookmark.createdAt)}</time
					>{/if}
				{#if collectionName}
					in
					{#if collectionHref}
						<a href={collectionHref} class="font-medium hover:underline">{collectionName}</a>
					{:else}
						<span class="font-medium">{collectionName}</span>
					{/if}
				{/if}
			</p>
		{/if}
	</div>

	{#if actions}
		<div class="absolute bottom-3 right-3">
			{@render actions()}
		</div>
	{/if}
</Card>
