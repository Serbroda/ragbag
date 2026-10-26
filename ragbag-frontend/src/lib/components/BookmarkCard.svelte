<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import type { Snippet } from 'svelte';
	import { FolderOutline, LinkOutline } from 'flowbite-svelte-icons';
	import { getDomain, safeUrl, timeAgo } from '../utils/format';
	import FavoriteButton from './FavoriteButton.svelte';

	let {
		bookmark,
		collectionName,
		collectionHref,
		actions,
		onfavoritechange,
	}: {
		bookmark: BookmarkDto;
		/** Shown in the footer when the card is listed outside its collection. */
		collectionName?: string;
		collectionHref?: string;
		actions?: Snippet;
		onfavoritechange?: (favorite: boolean) => void;
	} = $props();

	let favorite = $state(false);
	$effect(() => {
		favorite = bookmark.favorite ?? false;
	});

	const href = $derived(safeUrl(bookmark.url));
	let faviconFailed = $state(false);
	let imageFailed = $state(false);
	const favicon = $derived(faviconFailed ? undefined : safeUrl(bookmark.favicon));
	const image = $derived(imageFailed ? undefined : safeUrl(bookmark.ogImage));
</script>

<article
	class="group relative flex h-full flex-col overflow-hidden rounded-xl border border-gray-200 bg-white transition duration-200 hover:-translate-y-0.5 hover:border-gray-300 hover:shadow-lg dark:border-gray-800 dark:bg-gray-900 dark:hover:border-gray-700 dark:hover:shadow-black/30"
>
	<div
		class="absolute right-2 top-2 z-10 rounded-md bg-white/85 shadow-sm backdrop-blur transition dark:bg-gray-900/85 {favorite
			? ''
			: 'opacity-0 focus-within:opacity-100 group-hover:opacity-100'}"
	>
		<FavoriteButton bookmarkId={bookmark.id} bind:favorite onchange={onfavoritechange} />
	</div>

	{#if image}
		<img
			src={image}
			alt=""
			loading="lazy"
			referrerpolicy="no-referrer"
			class="aspect-[2.4/1] w-full bg-gray-100 object-cover dark:bg-gray-800"
			onerror={() => (imageFailed = true)}
		/>
	{/if}

	<div class="flex flex-1 flex-col gap-2 p-4">
		<div class="flex items-center gap-2 text-xs text-gray-500">
			{#if favicon}
				<img
					src={favicon}
					alt=""
					loading="lazy"
					referrerpolicy="no-referrer"
					class="h-4 w-4 shrink-0 rounded-sm"
					onerror={() => (faviconFailed = true)}
				/>
			{:else}
				<LinkOutline class="h-4 w-4 shrink-0" />
			{/if}
			<span class="truncate">{getDomain(bookmark.url)}</span>
		</div>

		<!-- The title link covers the whole card; footer controls sit above it -->
		<a
			{href}
			target="_blank"
			rel="noopener noreferrer"
			class="line-clamp-2 break-words font-semibold leading-snug text-gray-900 after:absolute after:inset-0 after:content-[''] group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400"
		>
			{bookmark.title || bookmark.url}
		</a>

		{#if bookmark.description}
			<p class="line-clamp-2 break-words text-sm text-gray-600 dark:text-gray-400">{bookmark.description}</p>
		{/if}

		{#if bookmark.tags && bookmark.tags.length > 0}
			<div class="flex flex-wrap gap-1.5">
				{#each bookmark.tags as tag (tag)}
					<span
						class="rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300"
					>
						#{tag}
					</span>
				{/each}
			</div>
		{/if}

		<div class="relative z-10 mt-auto flex items-center gap-2 pt-2 text-xs text-gray-500">
			<p class="min-w-0 flex-1 truncate">
				{#if collectionName}
					{#if collectionHref}
						<a href={collectionHref} class="inline-flex items-center gap-1 font-medium hover:text-gray-800 dark:hover:text-gray-200">
							<FolderOutline class="h-3.5 w-3.5" />{collectionName}</a
						>
						·
					{:else}
						<span class="font-medium">{collectionName}</span> ·
					{/if}
				{/if}
				{#if bookmark.createdBy}{bookmark.createdBy}{/if}
				{#if bookmark.createdAt}
					·
					<time datetime={bookmark.createdAt.toISOString()} title={bookmark.createdAt.toLocaleString()}>
						{timeAgo(bookmark.createdAt)}
					</time>
				{/if}
			</p>
			{#if actions}
				<div class="relative">
					{@render actions()}
				</div>
			{/if}
		</div>
	</div>
</article>
