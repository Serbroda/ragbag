<script lang="ts">
	import { ChevronDownOutline, SortOutline } from 'flowbite-svelte-icons';
	import { LINK_SORTS, preferences, setLinkSort, type LinkSort } from '../stores/preferences.svelte';
</script>

<!-- Personal and saved per device, like the link display -->
<label class="relative inline-flex shrink-0 items-center" title="Sort links">
	<SortOutline class="pointer-events-none absolute left-2 h-4 w-4 text-gray-500 dark:text-gray-400" />
	<span class="sr-only">Sort links</span>
	<select
		class="cursor-pointer appearance-none rounded-lg border-0 bg-gray-100 py-1.5 pl-7 pr-7 text-sm text-gray-700 focus:ring-2 focus:ring-primary-500 dark:bg-gray-800 dark:text-gray-200"
		value={preferences.linkSort}
		onchange={(e) => setLinkSort((e.currentTarget as HTMLSelectElement).value as LinkSort)}
	>
		{#each LINK_SORTS as sort (sort.value)}
			<option value={sort.value}>{sort.label}</option>
		{/each}
	</select>
	<ChevronDownOutline class="pointer-events-none absolute right-2 h-3.5 w-3.5 text-gray-500 dark:text-gray-400" />
</label>
