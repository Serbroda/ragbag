<script lang="ts">
	import type { DashboardSectionDto } from 'ragbag-frontend-sdk';
	import { NewspaperOutline } from 'flowbite-svelte-icons';
	import DashboardHeadlines from './DashboardHeadlines.svelte';

	/**
	 * Headline sections that follow each other on the dashboard: columns side by side where there is room, tabs
	 * on narrow screens (the chosen tab is remembered per space on this device). Each shows how many headlines are
	 * new since the last visit, so a category out of view still catches the eye.
	 */
	let { spaceId, sections }: { spaceId: string; sections: DashboardSectionDto[] } = $props();

	/** Below this width (42rem, the container query @2xl) the sections become tabs. */
	const COLUMNS_FROM = 672;

	let width = $state(0);
	const tabs = $derived(sections.length > 1 && width > 0 && width < COLUMNS_FROM);

	const tabKey = $derived(`ragbag:headline-tab:${spaceId}`);
	/** The tab picked here, else the one remembered */
	let chosen = $state<string | null>(null);
	const active = $derived.by(() => {
		const title = chosen ?? read(tabKey);
		return Math.max(0, sections.findIndex((section) => section.title === title));
	});
	let newCounts = $state<number[]>([]);

	function read(key: string): string {
		try {
			return localStorage.getItem(key) ?? '';
		} catch {
			return '';
		}
	}

	function choose(index: number) {
		chosen = sections[index].title;
		try {
			localStorage.setItem(tabKey, chosen);
		} catch {
			// Unavailable storage: the first tab after the next load
		}
	}
</script>

<div class="@container" bind:clientWidth={width}>
	{#if tabs}
		<div class="mb-3 flex gap-1 overflow-x-auto" role="tablist">
			{#each sections as section, index (section.id ?? index)}
				<button
					type="button"
					role="tab"
					aria-selected={index === active}
					class="flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1 text-sm font-medium transition-colors {index === active
						? 'bg-primary-500/15 text-primary-700 dark:text-primary-300'
						: 'text-gray-500 hover:bg-gray-100 hover:text-gray-800 dark:hover:bg-gray-800 dark:hover:text-gray-200'}"
					onclick={() => choose(index)}
				>
					{section.title}
					{#if newCounts[index]}
						<span class="rounded-full bg-primary-500 px-1.5 text-xs text-white" title="New since your last visit"
							>{newCounts[index]}</span
						>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
	<div
		class="grid gap-x-6 gap-y-10 {sections.length > 1 ? '@2xl:grid-cols-2' : ''} {sections.length > 2 ? '@5xl:grid-cols-3' : ''}"
	>
		{#each sections as section, index (section.id ?? index)}
			<section class={tabs && index !== active ? 'hidden' : 'min-w-0'}>
				{#if !tabs}
					<h2 class="mb-3 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
						<NewspaperOutline class="h-3.5 w-3.5 shrink-0" />
						<span class="truncate">{section.title}</span>
						<span class="shrink-0 font-normal normal-case tracking-normal text-gray-400"
							>· {section.bookmarks.length} feed{section.bookmarks.length === 1 ? '' : 's'}</span
						>
						{#if newCounts[index]}
							<span
								class="shrink-0 rounded-full bg-primary-500/15 px-1.5 font-medium normal-case tracking-normal text-primary-700 dark:text-primary-300"
								>{newCounts[index]} new</span
							>
						{/if}
					</h2>
				{/if}
				<DashboardHeadlines
					{spaceId}
					sources={section.bookmarks}
					count={section.headlineCount ?? 5}
					seenKey="ragbag:headlines-seen:{spaceId}:{section.title}"
					visible={!tabs || index === active}
					bind:newCount={newCounts[index]}
				/>
			</section>
		{/each}
	</div>
</div>
