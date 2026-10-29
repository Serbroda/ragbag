<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { CollectionNewsDto } from 'ragbag-frontend-sdk';
	import { getContext, untrack } from 'svelte';
	import { BellOutline, FolderOutline } from 'flowbite-svelte-icons';
	import { apiConfig } from '../api/client';
	import { p } from '../router';
	import { session } from '../stores/spaces.svelte';
	import { timeAgo } from '../utils/format';

	/** Bell in the header: "3 new links in Japan 2027 · by lena", what others added since the user last looked. */
	let { spaceId }: { spaceId: string } = $props();

	const collectionName = getContext<((id: string) => string | undefined) | undefined>('collection-name');
	const reloadSidebar = getContext<(() => void) | undefined>('reload-collections');
	const reloadSpaces = getContext<(() => Promise<void>) | undefined>('reload-spaces');

	let open = $state(false);
	let news = $state<CollectionNewsDto[]>([]);
	let marking = $state(false);

	// The space list is polled; reload when the space changes or its number of new links does
	const newCount = $derived(session.spaces.find((space) => space.id === spaceId)?.newCount ?? 0);
	$effect(() => {
		const id = spaceId;
		newCount;
		untrack(() => load(id));
	});

	async function load(id: string) {
		try {
			const loaded = await new SpaceApi(apiConfig()).getSpaceNews({ spaceId: id });
			if (id === spaceId) news = loaded;
		} catch {
			news = [];
		}
	}

	async function markAllSeen() {
		marking = true;
		try {
			await new SpaceApi(apiConfig()).markSpaceSeen({ spaceId });
			news = [];
			open = false;
			reloadSidebar?.();
			await reloadSpaces?.();
		} finally {
			marking = false;
		}
	}
</script>

<svelte:window onclick={() => (open = false)} onkeydown={(e) => e.key === 'Escape' && (open = false)} />

<div class="relative">
	<button
		type="button"
		class="relative rounded-md p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200"
		title="What's new"
		aria-label={newCount > 0 ? `What's new: ${newCount} new links` : "What's new"}
		aria-haspopup="dialog"
		aria-expanded={open}
		onclick={(e) => {
			e.stopPropagation();
			open = !open;
		}}
	>
		<BellOutline class="h-5 w-5" />
		{#if newCount > 0}
			<span
				class="absolute -right-0.5 -top-0.5 min-w-4 rounded-full bg-primary-500 px-1 text-center text-[10px] font-bold leading-4 text-white"
			>
				{newCount > 99 ? '99+' : newCount}
			</span>
		{/if}
	</button>

	{#if open}
		<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
		<div
			class="fixed inset-x-3 top-14 z-50 rounded-lg sm:absolute sm:inset-x-auto sm:right-0 sm:top-[calc(100%+6px)] sm:w-[22rem] bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
			role="dialog"
			tabindex="-1"
			aria-label="What's new"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="flex items-center justify-between gap-3 px-2 pb-1.5 pt-1">
				<p class="text-xs font-semibold uppercase tracking-wider text-gray-400">New since your last visit</p>
				{#if news.length > 0}
					<button
						type="button"
						class="shrink-0 text-xs font-medium text-primary-400 hover:underline disabled:opacity-50"
						disabled={marking}
						onclick={markAllSeen}
					>
						Mark all as read
					</button>
				{/if}
			</div>
			{#if news.length === 0}
				<p class="px-2 py-3 text-gray-400">You're all caught up.</p>
			{:else}
				<ul class="max-h-80 overflow-y-auto">
					{#each news as item (item.collectionId)}
						<li>
							<a
								href={p('/space/:spaceId/collections/:collectionId', {
									params: { spaceId, collectionId: item.collectionId },
								})}
								class="flex items-start gap-2.5 rounded-md px-2 py-2 hover:bg-primary-500 hover:text-white"
								onclick={() => (open = false)}
							>
								<FolderOutline class="mt-0.5 h-4 w-4 shrink-0 opacity-70" />
								<span class="min-w-0 flex-1">
									<span class="block">
										<span class="font-semibold">{item.count} new link{item.count === 1 ? '' : 's'}</span>
										in <span class="font-semibold">{collectionName?.(item.collectionId) ?? 'a collection'}</span>
									</span>
									<span class="block truncate text-xs opacity-70">
										by {item.addedBy.join(', ')} · {timeAgo(item.latestAt)}
									</span>
								</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>
