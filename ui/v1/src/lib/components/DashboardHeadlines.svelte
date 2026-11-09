<script lang="ts">
	import { DashboardApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, HeadlinesDto } from 'ragbag-frontend-sdk';
	import { LinkOutline } from 'flowbite-svelte-icons';
	import { apiConfig } from '../api/client';
	import { getDomain, safeUrl } from '../utils/format';
	import { headlineAge, isFreshHeadline } from '../utils/links';

	/** A dashboard section: the headlines of several links' feeds merged into one list, newest first. */
	let { spaceId, sources }: { spaceId: string; sources: BookmarkDto[] } = $props();

	const LOADED = 30;
	const SHOWN = 10;

	let result = $state<HeadlinesDto | null>(null);
	let failed = $state(false);
	let expanded = $state(false);

	const ids = $derived(sources.map((source) => source.id));
	const byId = $derived(new Map(sources.map((source) => [source.id, source])));

	$effect(() => {
		const wanted = ids;
		const key = wanted.join(',');
		result = null;
		failed = false;
		new DashboardApi(apiConfig())
			.getSpaceHeadlines({ spaceId, bookmarkId: wanted, limit: LOADED })
			.then((loaded) => {
				if (key === ids.join(',')) result = loaded;
			})
			.catch(() => (failed = true));
	});

	const items = $derived(
		(result?.items ?? []).flatMap((item) => {
			const href = safeUrl(item.url);
			if (!href) return [];
			const source = item.bookmarkId ? byId.get(item.bookmarkId) : undefined;
			return [{ ...item, href, source, favicon: safeUrl(source?.favicon) }];
		}),
	);
	const shown = $derived(expanded ? items : items.slice(0, SHOWN));
	let failedIcons = $state<Set<string>>(new Set());
</script>

<div class="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
	{#if !result && !failed}
		<div class="flex flex-col gap-3 p-4" aria-label="Loading headlines">
			{#each { length: 4 } as _, i (i)}
				<div class="h-3.5 animate-pulse rounded bg-gray-200 dark:bg-gray-800" style="width: {85 - i * 12}%"></div>
			{/each}
		</div>
	{:else if failed || result?.failed}
		<p class="px-4 py-6 text-center text-sm text-gray-500">The feeds cannot be loaded right now.</p>
	{:else if items.length === 0}
		<p class="px-4 py-6 text-center text-sm text-gray-500">No current headlines.</p>
	{:else}
		<ul class="divide-y divide-gray-100 dark:divide-gray-800">
			{#each shown as item (item.href)}
				{@const fresh = isFreshHeadline(item.publishedAt)}
				<li>
					<a
						href={item.href}
						target="_blank"
						rel="noopener noreferrer"
						class="group flex items-baseline gap-3 px-4 py-2.5 transition-colors first:rounded-t-xl hover:bg-gray-50 dark:hover:bg-gray-800/60"
					>
						<time
							class="w-8 shrink-0 text-xs tabular-nums {fresh
								? 'font-semibold text-primary-600 dark:text-primary-400'
								: 'text-gray-400 dark:text-gray-500'}"
							datetime={item.publishedAt?.toISOString()}
							title={item.publishedAt?.toLocaleString()}>{item.publishedAt ? headlineAge(item.publishedAt) : ''}</time
						>
						<span class="min-w-0 flex-1">
							<span
								class="line-clamp-2 break-words text-sm group-hover:text-primary-600 group-hover:underline dark:group-hover:text-primary-400 {fresh
									? 'font-medium text-gray-900 dark:text-gray-100'
									: 'text-gray-700 dark:text-gray-300'}">{item.title}</span
							>
							{#if item.source}
								<span class="mt-0.5 flex items-center gap-1.5 text-xs text-gray-500">
									{#if item.favicon && !failedIcons.has(item.favicon)}
										<img
											src={item.favicon}
											alt=""
											loading="lazy"
											referrerpolicy="no-referrer"
											class="h-3.5 w-3.5 rounded-sm"
											onerror={() => (failedIcons = new Set([...failedIcons, item.favicon!]))}
										/>
									{:else}
										<LinkOutline class="h-3.5 w-3.5" />
									{/if}
									<span class="truncate">{item.source.title || getDomain(item.source.url)}</span>
								</span>
							{/if}
						</span>
					</a>
				</li>
			{/each}
		</ul>
		{#if items.length > SHOWN}
			<button
				type="button"
				class="w-full rounded-b-xl border-t border-gray-100 py-2 text-sm text-gray-500 hover:bg-gray-50 hover:text-gray-800 dark:border-gray-800 dark:hover:bg-gray-800/60 dark:hover:text-gray-200"
				onclick={() => (expanded = !expanded)}
			>
				{expanded ? 'Show fewer' : `Show ${items.length - SHOWN} more`}
			</button>
		{/if}
	{/if}
</div>
