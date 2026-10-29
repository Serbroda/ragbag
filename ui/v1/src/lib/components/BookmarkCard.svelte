<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import type { Snippet } from 'svelte';
	import { FolderOutline, LinkOutline } from 'flowbite-svelte-icons';
	import { getDomain, safeUrl, timeAgo } from '../utils/format';
	import { preferences, type LinkView } from '../stores/preferences.svelte';
	import FavoriteButton from './FavoriteButton.svelte';
	import Reactions from './Reactions.svelte';
	import { p, route } from '../router';

	let {
		bookmark,
		view: viewOverride,
		collectionName,
		collectionHref,
		actions,
		onfavoritechange,
		ondragstart,
		ondragend,
	}: {
		bookmark: BookmarkDto;
		/** Defaults to the user's link display preference. */
		view?: LinkView;
		/** Shown in the footer when the card is listed outside its collection. */
		collectionName?: string;
		collectionHref?: string;
		actions?: Snippet;
		onfavoritechange?: (favorite: boolean) => void;
		/** Makes the card draggable, e.g. onto a collection in the sidebar. */
		ondragstart?: (event: DragEvent) => void;
		ondragend?: (event: DragEvent) => void;
	} = $props();

	const spaceId = $derived(route.params.spaceId);

	const view = $derived(viewOverride ?? preferences.linkView);

	let favorite = $state(false);
	$effect(() => {
		favorite = bookmark.favorite ?? false;
	});

	const href = $derived(safeUrl(bookmark.url));
	let faviconFailed = $state(false);
	let imageFailed = $state(false);
	const favicon = $derived(faviconFailed ? undefined : safeUrl(bookmark.favicon));
	const image = $derived(imageFailed || view !== 'cards' ? undefined : safeUrl(bookmark.ogImage));
</script>

{#snippet faviconIcon(size: string)}
	{#if favicon}
		<img
			src={favicon}
			alt=""
			loading="lazy"
			referrerpolicy="no-referrer"
			class="{size} shrink-0 rounded-sm"
			onerror={() => (faviconFailed = true)}
		/>
	{:else}
		<LinkOutline class="{size} shrink-0 text-gray-400" />
	{/if}
{/snippet}

{#snippet titleLink(classes: string)}
	<!-- The title link covers the whole card; controls are lifted above it with z-10 -->
	<a
		{href}
		target="_blank"
		rel="noopener noreferrer"
		class="break-words text-gray-900 after:absolute after:inset-0 after:content-[''] group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400 {classes}"
	>
		{bookmark.title || bookmark.url}
	</a>
{/snippet}

{#snippet meta()}
	{#if collectionName}
		{#if collectionHref}
			<a
				href={collectionHref}
				class="relative z-10 inline-flex items-center gap-1 font-medium hover:text-gray-800 dark:hover:text-gray-200"
			>
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
{/snippet}

{#snippet newBadge()}
	{#if bookmark.isNew}
		<span
			class="shrink-0 rounded bg-primary-500 px-1.5 py-px text-[10px] font-bold uppercase tracking-wide text-white"
			title="Added since your last visit">New</span
		>
	{/if}
{/snippet}

{#snippet tags(limit: number)}
	{#each (bookmark.tags ?? []).slice(0, limit) as tag (tag)}
		<!-- Lifted above the title link that covers the card -->
		<a
			href={spaceId ? p('/space/:spaceId/search', { params: { spaceId }, search: { q: `#${tag}` } }) : undefined}
			title="Show all links tagged #{tag}"
			class="relative z-10 rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-700 transition-colors hover:bg-primary-500/25 dark:text-primary-300"
		>
			#{tag}
		</a>
	{/each}
{/snippet}

{#if view === 'list'}
	<article
		draggable={ondragstart ? 'true' : undefined}
		{ondragstart}
		{ondragend}
		class="group relative flex items-center gap-3 px-3 py-2 transition-colors focus-within:z-20 first:rounded-t-xl last:rounded-b-xl hover:bg-gray-50 dark:hover:bg-gray-800/60"
	>
		{@render faviconIcon('h-5 w-5')}
		<div class="min-w-0 flex-1">
			<p class="flex min-w-0 items-center gap-2 text-sm">
				{@render newBadge()}<span class="truncate">{@render titleLink('font-medium')}</span>
			</p>
			<p class="truncate text-xs text-gray-500">
				{getDomain(bookmark.url)}
				{#if collectionName || bookmark.createdBy || bookmark.createdAt}·{/if}
				{@render meta()}
			</p>
		</div>
		{#if bookmark.tags?.length}
			<div class="hidden shrink-0 gap-1.5 lg:flex">{@render tags(3)}</div>
		{/if}
		<div class="shrink-0"><Reactions bookmarkId={bookmark.id} reactions={bookmark.reactions} /></div>
		<div
			class="relative z-10 shrink-0 transition {favorite ? '' : 'opacity-0 focus-within:opacity-100 group-hover:opacity-100'}"
		>
			<FavoriteButton bookmarkId={bookmark.id} bind:favorite onchange={onfavoritechange} />
		</div>
		{#if actions}
			<div class="relative z-10 shrink-0">
				{@render actions()}
			</div>
		{/if}
	</article>
{:else}
	<article
		draggable={ondragstart ? 'true' : undefined}
		{ondragstart}
		{ondragend}
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

		<div class="flex flex-1 flex-col {view === 'compact' ? 'gap-1.5 p-3' : 'gap-2 p-4'}">
			<div class="flex items-center gap-2 pe-8 text-xs text-gray-500">
				{@render faviconIcon('h-4 w-4')}
				<span class="truncate">{getDomain(bookmark.url)}</span>
				{@render newBadge()}
			</div>

			{@render titleLink('line-clamp-2 font-semibold leading-snug')}

			{#if bookmark.description}
				<p
					class="break-words text-sm text-gray-600 dark:text-gray-400 {view === 'compact'
						? 'line-clamp-1'
						: 'line-clamp-2'}"
				>
					{bookmark.description}
				</p>
			{/if}

			{#if bookmark.tags && bookmark.tags.length > 0}
				<div class="flex flex-wrap gap-1.5">{@render tags(view === 'compact' ? 4 : 20)}</div>
			{/if}

			<div class="relative z-10 mt-auto flex items-center gap-2 pt-2 text-xs text-gray-500">
				<p class="min-w-0 flex-1 truncate">{@render meta()}</p>
				<Reactions bookmarkId={bookmark.id} reactions={bookmark.reactions} />
				{#if actions}
					<div class="relative">
						{@render actions()}
					</div>
				{/if}
			</div>
		</div>
	</article>
{/if}
