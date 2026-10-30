<script lang="ts">
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import type { Snippet } from 'svelte';
	import { FolderOutline, LinkOutline } from 'flowbite-svelte-icons';
	import { getDomain, safeUrl, timeAgo } from '../utils/format';
	import { previewImage } from '../utils/links';
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
		publicView = false,
		selectable = false,
		selected = false,
		selecting = false,
		onselect,
		raised = false,
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
		/** Shown through a public link: no favorites, reactions, authors or links into the app. */
		publicView?: boolean;
		/** Multi-select: a checkbox on the card (on hover, always once something is selected). */
		selectable?: boolean;
		selected?: boolean;
		selecting?: boolean;
		onselect?: (event: MouseEvent) => void;
		/** Lifts the card above its neighbours, e.g. while its menu is open. */
		raised?: boolean;
	} = $props();

	// Touch screens have no hover: a long press selects the card (and starts selecting)
	let pressTimer: ReturnType<typeof setTimeout> | undefined;
	let pressed = false;
	let pressStart = { x: 0, y: 0 };

	function onpointerdown(e: PointerEvent) {
		if (!selectable || e.pointerType === 'mouse') return;
		pressed = false;
		pressStart = { x: e.clientX, y: e.clientY };
		clearTimeout(pressTimer);
		pressTimer = setTimeout(() => {
			pressed = true;
			navigator.vibrate?.(15);
			onselect?.(new MouseEvent('click'));
		}, 450);
	}

	function cancelPress() {
		clearTimeout(pressTimer);
	}

	/** A finger always moves a little; only scrolling (more than a few pixels) cancels the long press. */
	function onpointermove(e: PointerEvent) {
		if (Math.hypot(e.clientX - pressStart.x, e.clientY - pressStart.y) > 8) cancelPress();
	}

	/** While selecting, tapping the card (its title link covers it) selects instead of opening the link. */
	function onlinkclick(e: MouseEvent) {
		if (pressed) {
			pressed = false;
			e.preventDefault();
			return;
		}
		if (selectable && selecting) {
			e.preventDefault();
			onselect?.(e);
		}
	}

	const spaceId = $derived(route.params.spaceId);

	const view = $derived(viewOverride ?? preferences.linkView);

	let favorite = $state(false);
	$effect(() => {
		favorite = bookmark.favorite ?? false;
	});

	const href = $derived(safeUrl(bookmark.url));
	let faviconFailed = $state(false);
	/** The image that failed to load; a changed image (e.g. a new upload) gets a new chance. */
	let failedImage = $state<string | undefined>();
	const favicon = $derived(faviconFailed ? undefined : safeUrl(bookmark.favicon));
	const source = $derived(previewImage(bookmark));
	const image = $derived(view !== 'cards' || source === failedImage ? undefined : source);
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
		class="break-words text-gray-900 after:absolute after:inset-0 after:content-[''] group-hover:text-primary-600 dark:text-white dark:group-hover:text-primary-400 {selectable
			? '[-webkit-touch-callout:none]'
			: ''} {classes}"
		onclick={onlinkclick}
		oncontextmenu={(e) => pressed && e.preventDefault()}
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
	{#if bookmark.createdBy && !publicView}{bookmark.createdBy}{/if}
	{#if bookmark.createdAt}
		{#if bookmark.createdBy && !publicView}·{/if}
		<time datetime={bookmark.createdAt.toISOString()} title={bookmark.createdAt.toLocaleString()}>
			{timeAgo(bookmark.createdAt)}
		</time>
	{/if}
{/snippet}

{#snippet checkbox(classes: string)}
	{#if selectable}
		<button
			type="button"
			role="checkbox"
			aria-checked={selected}
			aria-label="Select {bookmark.title || bookmark.url}"
			title="Select (Shift: a range)"
			class="z-10 flex h-5 w-5 shrink-0 items-center justify-center rounded border-2 transition {selected
				? 'border-primary-500 bg-primary-500 text-white'
				: 'border-gray-400 bg-white dark:border-gray-500 dark:bg-gray-900'} {classes}"
			onclick={(e) => {
				e.preventDefault();
				e.stopPropagation();
				onselect?.(e);
			}}
		>
			{#if selected}
				<svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5"
					><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg
				>
			{/if}
		</button>
	{/if}
{/snippet}

<!-- The favicon, which turns into the checkbox on hover (and stays one while something is selected) -->
{#snippet lead(size: string)}
	{#if selectable && (selected || selecting)}
		{@render checkbox('relative')}
	{:else if selectable}
		<span class="relative flex shrink-0 items-center">
			<span class="group-hover:hidden group-focus-within:hidden">{@render faviconIcon(size)}</span>
			{@render checkbox('relative hidden group-hover:flex group-focus-within:flex')}
		</span>
	{:else}
		{@render faviconIcon(size)}
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
		{#if publicView || !spaceId}
			<span class="rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300">
				#{tag}
			</span>
		{:else}
			<a
				href={p('/space/:spaceId/search', { params: { spaceId }, search: { q: `#${tag}` } })}
				title="Show all links tagged #{tag}"
				class="relative z-10 rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-700 transition-colors hover:bg-primary-500/25 dark:text-primary-300"
			>
				#{tag}
			</a>
		{/if}
	{/each}
{/snippet}

{#if view === 'list'}
	<article
		draggable={ondragstart ? 'true' : undefined}
		{ondragstart}
		{ondragend}
		class="group relative flex items-center gap-3 px-3 py-2 transition-colors focus-within:z-20 first:rounded-t-xl last:rounded-b-xl hover:bg-gray-50 dark:hover:bg-gray-800/60 {selected
			? 'bg-primary-500/10'
			: ''} {raised ? 'z-30' : ''}"
		{onpointerdown}
		onpointerup={cancelPress}
		{onpointermove}
		onpointercancel={cancelPress}
	>
		{@render lead('h-5 w-5')}
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
		{#if !publicView}
			<div class="shrink-0"><Reactions bookmarkId={bookmark.id} reactions={bookmark.reactions} /></div>
			<div
				class="relative z-10 shrink-0 transition {favorite ? '' : 'opacity-0 focus-within:opacity-100 group-hover:opacity-100'}"
			>
				<FavoriteButton bookmarkId={bookmark.id} bind:favorite onchange={onfavoritechange} />
			</div>
		{/if}
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
		{onpointerdown}
		onpointerup={cancelPress}
		{onpointermove}
		onpointercancel={cancelPress}
		class="group relative flex h-full flex-col rounded-xl border bg-white transition duration-200 {raised ? 'z-30' : ''} hover:-translate-y-0.5 hover:shadow-lg dark:bg-gray-900 dark:hover:shadow-black/30 {selected
			? 'border-primary-500 ring-2 ring-primary-500'
			: 'border-gray-200 hover:border-gray-300 dark:border-gray-800 dark:hover:border-gray-700'}"
	>
		{#if !publicView}
			<div
				class="absolute right-2 top-2 z-10 rounded-md bg-white/85 shadow-sm backdrop-blur transition dark:bg-gray-900/85 {favorite
					? ''
					: 'opacity-0 focus-within:opacity-100 group-hover:opacity-100'}"
			>
				<FavoriteButton bookmarkId={bookmark.id} bind:favorite onchange={onfavoritechange} />
			</div>
		{/if}

		{#if image}
			<img
				src={image}
				alt=""
				loading="lazy"
				referrerpolicy="no-referrer"
				class="aspect-[2.4/1] w-full rounded-t-[11px] bg-gray-100 object-cover dark:bg-gray-800"
				onerror={() => (failedImage = source)}
			/>
		{/if}

		<div class="flex flex-1 flex-col {view === 'compact' ? 'gap-1.5 p-3' : 'gap-2 p-4'}">
			<div class="flex items-center gap-2 pe-8 text-xs text-gray-500">
				{@render lead('h-4 w-4')}
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
				{#if !publicView}
					<Reactions bookmarkId={bookmark.id} reactions={bookmark.reactions} />
				{/if}
				{#if actions}
					<div class="relative">
						{@render actions()}
					</div>
				{/if}
			</div>
		</div>
	</article>
{/if}
