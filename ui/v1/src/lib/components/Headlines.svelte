<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { HeadlinesDto } from 'ragbag-frontend-sdk';
	import { apiConfig, getSharedHeadlines } from '../api/client';
	import { safeUrl } from '../utils/format';
	import { headlineAge, isFreshHeadline } from '../utils/links';

	/**
	 * The latest headlines of a link's news feed on its card. Loaded once the card comes near the screen (the
	 * server loads the feed, at most once an hour); each headline opens its article.
	 */
	let {
		bookmarkId,
		shareToken,
		limit,
		reloadKey,
	}: {
		bookmarkId: string;
		/** Shown through a public link: loaded without an account. */
		shareToken?: string;
		limit: number;
		/** Loads again when it changes (a feed or filter saved in the edit dialog). */
		reloadKey: string;
	} = $props();

	let element = $state<HTMLElement | null>(null);
	let visible = $state(false);
	let result = $state<HeadlinesDto | null>(null);
	let failed = $state(false);

	$effect(() => {
		if (!element || visible) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) visible = true;
			},
			{ rootMargin: '200px' },
		);
		observer.observe(element);
		return () => observer.disconnect();
	});

	$effect(() => {
		if (!visible) return;
		const key = reloadKey;
		const id = bookmarkId;
		result = null;
		failed = false;
		const request = shareToken
			? getSharedHeadlines(shareToken, id)
			: new BookmarkApi(apiConfig()).getBookmarkHeadlines({ bookmarkId: id });
		request
			.then((loaded) => {
				if (key === reloadKey && id === bookmarkId) result = loaded;
			})
			.catch(() => (failed = true));
	});

	const items = $derived(
		(result?.items ?? []).slice(0, limit).flatMap((item) => {
			const href = safeUrl(item.url);
			return href ? [{ ...item, href }] : [];
		}),
	);
</script>

<div bind:this={element} class="relative z-10 border-s-2 border-primary-500/40 ps-2.5 text-[13px] leading-snug">
	{#if !result && !failed}
		<div class="flex flex-col gap-1.5 py-0.5" aria-label="Loading headlines">
			{#each { length: Math.min(limit, 3) } as _, i (i)}
				<div class="h-3 animate-pulse rounded bg-gray-200 dark:bg-gray-800" style="width: {90 - i * 15}%"></div>
			{/each}
		</div>
	{:else if failed || result?.failed}
		<p class="text-xs text-gray-400 dark:text-gray-500">Headlines unavailable right now</p>
	{:else if items.length === 0}
		<p class="text-xs text-gray-400 dark:text-gray-500">No current headlines</p>
	{:else}
		<ul class="flex flex-col gap-1.5" aria-label={result?.feedTitle ? `Headlines from ${result.feedTitle}` : 'Headlines'}>
			{#each items as item (item.href)}
				{@const fresh = isFreshHeadline(item.publishedAt)}
				<li class="flex min-w-0 items-baseline gap-2">
					<time
						class="w-7 shrink-0 text-[11px] tabular-nums {fresh
							? 'font-semibold text-primary-600 dark:text-primary-400'
							: 'text-gray-400 dark:text-gray-500'}"
						datetime={item.publishedAt?.toISOString()}
						title={item.publishedAt?.toLocaleString()}>{item.publishedAt ? headlineAge(item.publishedAt) : ''}</time
					>
					<a
						href={item.href}
						target="_blank"
						rel="noopener noreferrer"
						title={item.title}
						class="line-clamp-2 min-w-0 flex-1 break-words hover:text-primary-600 hover:underline dark:hover:text-primary-400 {fresh
							? 'font-medium text-gray-900 dark:text-gray-100'
							: 'text-gray-600 dark:text-gray-400'}"
					>
						{item.title}
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</div>
