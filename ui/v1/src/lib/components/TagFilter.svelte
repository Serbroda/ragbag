<script lang="ts">
	import { CloseOutline } from 'flowbite-svelte-icons';
	import { filterByTags } from '../utils/links';

	/**
	 * Tag chips to narrow a list of links: clicking selects a tag, several selected tags mean "has all of them".
	 * Counts refer to what is left with the current selection, so every chip leads somewhere.
	 */
	let {
		links,
		selected = $bindable([]),
		limit = 12,
	}: { links: { tags?: string[]; url: string }[]; selected?: string[]; limit?: number } = $props();

	let showAll = $state(false);

	const chips = $derived.by(() => {
		const remaining = filterByTags(links, selected);
		// Tags differ only in case are one chip, shown in the spelling used most (ties: alphabetically)
		const counts = new Map<string, { tag: string; count: number; spellings: Map<string, number> }>();
		for (const link of remaining) {
			for (const tag of link.tags ?? []) {
				const key = tag.toLowerCase();
				const entry = counts.get(key) ?? { tag, count: 0, spellings: new Map() };
				entry.count++;
				entry.spellings.set(tag, (entry.spellings.get(tag) ?? 0) + 1);
				counts.set(key, entry);
			}
		}
		for (const entry of counts.values()) {
			entry.tag = [...entry.spellings].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))[0][0];
		}
		const isSelected = (tag: string) => selected.some((s) => s.toLowerCase() === tag.toLowerCase());
		return [...counts.values()]
			.map(({ tag, count }) => ({ tag, count, active: isSelected(tag) }))
			.sort((a, b) => Number(b.active) - Number(a.active) || b.count - a.count || a.tag.localeCompare(b.tag));
	});
	const visible = $derived(showAll ? chips : chips.slice(0, Math.max(limit, selected.length)));

	function toggle(tag: string) {
		const key = tag.toLowerCase();
		selected = selected.some((s) => s.toLowerCase() === key)
			? selected.filter((s) => s.toLowerCase() !== key)
			: [...selected, tag];
	}
</script>

{#if chips.length > 0}
	<div class="flex min-w-0 flex-wrap items-center gap-1.5" role="group" aria-label="Filter by tag">
		{#each visible as chip (chip.tag.toLowerCase())}
			<button
				type="button"
				aria-pressed={chip.active}
				class="inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium transition-colors {chip.active
					? 'bg-primary-500 text-white hover:bg-primary-600'
					: 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700'}"
				onclick={() => toggle(chip.tag)}
			>
				#{chip.tag}
				<span class="tabular-nums {chip.active ? 'text-white/80' : 'text-gray-400 dark:text-gray-500'}">{chip.count}</span>
			</button>
		{/each}
		{#if chips.length > visible.length}
			<button
				type="button"
				class="rounded-full px-2 py-1 text-xs font-medium text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
				onclick={() => (showAll = true)}
			>
				+{chips.length - visible.length} more
			</button>
		{:else if showAll && chips.length > limit}
			<button
				type="button"
				class="rounded-full px-2 py-1 text-xs font-medium text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
				onclick={() => (showAll = false)}
			>
				Show less
			</button>
		{/if}
		{#if selected.length > 0}
			<button
				type="button"
				class="inline-flex items-center gap-0.5 rounded-full px-2 py-1 text-xs font-medium text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
				onclick={() => (selected = [])}
			>
				<CloseOutline class="h-3.5 w-3.5" /> Clear
			</button>
		{/if}
	</div>
{/if}
