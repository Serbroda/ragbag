<script lang="ts" module>
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { TagCountDto } from 'ragbag-frontend-sdk';
	import { apiConfig } from '../api/client';

	// One request per space and minute is plenty for suggestions
	const cache = new Map<string, { at: number; tags: Promise<TagCountDto[]> }>();

	function spaceTags(spaceId: string): Promise<TagCountDto[]> {
		const cached = cache.get(spaceId);
		if (cached && Date.now() - cached.at < 60_000) return cached.tags;
		const tags = new SpaceApi(apiConfig()).getSpaceTags({ spaceId }).catch(() => []);
		cache.set(spaceId, { at: Date.now(), tags });
		return tags;
	}
</script>

<script lang="ts">
	/**
	 * Comma separated tags with suggestions from the tags the space already uses, so "recipe" does not end up
	 * next to "Recipes" and "recipes".
	 */
	let {
		id,
		spaceId,
		value = $bindable(''),
	}: { id: string; spaceId: string | undefined; value?: string } = $props();

	let known = $state<TagCountDto[]>([]);
	let focused = $state(false);
	let active = $state(0);

	$effect(() => {
		const space = spaceId;
		if (space) spaceTags(space).then((tags) => space === spaceId && (known = tags));
	});

	const entered = $derived(
		value
			.split(',')
			.slice(0, -1)
			.map((tag) => tag.trim().toLowerCase())
			.filter(Boolean),
	);
	/** The tag being typed: everything after the last comma. */
	const current = $derived((value.split(',').at(-1) ?? '').trim().toLowerCase());
	const suggestions = $derived(
		known
			.filter((entry) => !entered.includes(entry.tag.toLowerCase()))
			.filter((entry) => entry.tag.toLowerCase().includes(current) && entry.tag.toLowerCase() !== current)
			.sort((a, b) => Number(b.tag.toLowerCase().startsWith(current)) - Number(a.tag.toLowerCase().startsWith(current)))
			.slice(0, 8),
	);
	const open = $derived(focused && suggestions.length > 0);

	$effect(() => {
		current;
		active = 0;
	});

	function choose(tag: string) {
		const parts = value.split(',');
		parts[parts.length - 1] = parts.length > 1 ? ` ${tag}` : tag;
		value = parts.join(',') + ', ';
	}

	function onkeydown(e: KeyboardEvent) {
		if (!open) return;
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			active = (active + 1) % suggestions.length;
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			active = (active - 1 + suggestions.length) % suggestions.length;
		} else if ((e.key === 'Enter' || e.key === 'Tab') && current) {
			// Enter would submit the form; with something typed it picks the suggestion instead
			e.preventDefault();
			choose(suggestions[active].tag);
		} else if (e.key === 'Escape') {
			e.stopPropagation();
			focused = false;
		}
	}
</script>

<div class="relative">
	<input
		{id}
		type="text"
		autocomplete="off"
		placeholder="Comma separated"
		class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400"
		role="combobox"
		aria-expanded={open}
		aria-controls="{id}-suggestions"
		aria-autocomplete="list"
		bind:value
		onfocus={() => (focused = true)}
		oninput={() => (focused = true)}
		onblur={() => setTimeout(() => (focused = false), 150)}
		{onkeydown}
	/>
	{#if open}
		<ul
			id="{id}-suggestions"
			role="listbox"
			class="absolute left-0 right-0 top-full z-30 mt-1 max-h-56 overflow-y-auto rounded-lg bg-white p-1 text-sm shadow-lg ring-1 ring-gray-200 dark:bg-gray-800 dark:ring-gray-700"
		>
			{#each suggestions as suggestion, i (suggestion.tag)}
				<li role="option" aria-selected={i === active}>
					<button
						type="button"
						class="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left {i === active
							? 'bg-primary-500 text-white'
							: 'text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700'}"
						onmousedown={(e) => (e.preventDefault(), choose(suggestion.tag))}
					>
						<span>#{suggestion.tag}</span>
						<span class="text-xs tabular-nums opacity-70">{suggestion.count}</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>
