<script lang="ts">
	import Avatar from './Avatar.svelte';

	/** Overlapping avatars of a few people ("who added this", the members of a space), the rest as "+3". */
	let {
		people,
		max = 3,
		class: className = 'h-6 w-6 text-[9px]',
		ring = 'ring-white dark:ring-gray-850',
	}: {
		people: { name: string; avatarId?: string | null }[];
		max?: number;
		/** Size and text size of each avatar */
		class?: string;
		/** The ring separating the avatars: the color of the surface they sit on */
		ring?: string;
	} = $props();

	const shown = $derived(people.length > max ? people.slice(0, max - 1) : people);
	const rest = $derived(people.length - shown.length);
</script>

<span class="flex shrink-0 -space-x-1.5" aria-hidden="true">
	{#each shown as person, i (i)}
		<Avatar name={person.name} imageId={person.avatarId} class="rounded-full ring-2 {ring} {className}" />
	{/each}
	{#if rest > 0}
		<span
			class="flex items-center justify-center rounded-full bg-gray-200 font-semibold text-gray-600 ring-2 dark:bg-gray-700 dark:text-gray-200 {ring} {className}"
		>
			<!-- Three digits do not fit the circle (a space with 300 members) -->
			{rest > 99 ? '99+' : `+${rest}`}
		</span>
	{/if}
</span>
