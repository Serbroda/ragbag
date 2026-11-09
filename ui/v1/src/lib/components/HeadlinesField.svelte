<script lang="ts">
	import { Input, Label, Toggle } from 'flowbite-svelte';
	import type { HeadlineSettings } from '../utils/links';

	/**
	 * Whether the card shows the latest headlines of a news feed, and which. The feed may be left empty: the server
	 * then looks for the feed of the link's page; it also takes a page instead of the feed itself.
	 */
	let {
		idPrefix,
		headlines = $bindable(null),
		found,
	}: {
		idPrefix: string;
		/** null shows no headlines. */
		headlines?: HeadlineSettings | null;
		/** The feed the page links to, from "Fetch details". */
		found?: string;
	} = $props();

	function toggle(on: boolean) {
		headlines = on ? { feedUrl: found ?? '', filter: '' } : null;
	}

	// A feed found later fills an empty field
	$effect(() => {
		if (found && headlines && !headlines.feedUrl.trim()) headlines.feedUrl = found;
	});
</script>

<div class="flex flex-col gap-3">
	<div>
		<Toggle checked={!!headlines} onchange={(e) => toggle((e.currentTarget as HTMLInputElement).checked)}>
			Show the latest headlines
		</Toggle>
		<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
			{#if found && !headlines}
				This page has a news feed: the card can show what is new there.
			{:else}
				For news sites, patch notes or blogs with an RSS or Atom feed.
			{/if}
		</p>
	</div>
	{#if headlines}
		<div>
			<Label for="{idPrefix}-feed" class="mb-2">Feed</Label>
			<Input
				id="{idPrefix}-feed"
				type="text"
				inputmode="url"
				placeholder="Found on the link's page if empty"
				bind:value={headlines.feedUrl}
			/>
		</div>
		<div>
			<Label for="{idPrefix}-feed-filter" class="mb-2">Only headlines with</Label>
			<Input
				id="{idPrefix}-feed-filter"
				type="text"
				maxlength={100}
				placeholder="Optional, e.g. Forever, Raid"
				bind:value={headlines.filter}
			/>
		</div>
	{/if}
</div>
