<script lang="ts">
	import { GridOutline, ImageOutline, ListOutline } from 'flowbite-svelte-icons';
	import { LINK_VIEWS, preferences, setLinkView, type LinkView } from '../stores/preferences.svelte';

	const ICONS: Record<LinkView, typeof GridOutline> = {
		cards: ImageOutline,
		compact: GridOutline,
		list: ListOutline,
	};
</script>

<div
	role="radiogroup"
	aria-label="Link display"
	class="inline-flex shrink-0 rounded-lg bg-gray-100 p-0.5 dark:bg-gray-800"
>
	{#each LINK_VIEWS as view (view.value)}
		{@const Icon = ICONS[view.value]}
		{@const active = preferences.linkView === view.value}
		<button
			type="button"
			role="radio"
			aria-checked={active}
			title={view.description}
			aria-label={view.label}
			class="rounded-md p-1.5 transition-colors {active
				? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white'
				: 'text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'}"
			onclick={() => setLinkView(view.value)}
		>
			<Icon class="h-4 w-4" />
		</button>
	{/each}
</div>
