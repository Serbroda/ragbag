<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { BookmarkMetadataDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Spinner, Textarea } from 'flowbite-svelte';
	import { WandMagicSparklesOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { getDomain, safeUrl } from '../utils/format';

	let {
		collectionId,
		idPrefix,
		url = $bindable(''),
		title = $bindable(''),
		description = $bindable(''),
		tags = $bindable(''),
	}: {
		collectionId: string;
		/** Keeps element ids unique when several forms are mounted. */
		idPrefix: string;
		url?: string;
		title?: string;
		description?: string;
		tags?: string;
	} = $props();

	let fetching = $state(false);
	let status = $state<{ kind: 'ok' | 'info' | 'error'; text: string } | null>(null);
	let preview = $state<BookmarkMetadataDto | null>(null);

	// A new URL invalidates the previous result
	let lastFetchedUrl = '';
	$effect(() => {
		if (url.trim() !== lastFetchedUrl) {
			status = null;
			preview = null;
		}
	});

	async function fetchMetadata() {
		if (!url.trim() || fetching) return;
		fetching = true;
		status = null;
		try {
			const metadata = await new BookmarkApi(apiConfig()).fetchBookmarkMetadata({
				collectionId,
				fetchMetadataRequest: { url: url.trim() },
			});
			url = metadata.url;
			lastFetchedUrl = metadata.url;
			if (!metadata.found) {
				preview = null;
				status = { kind: 'info', text: 'No details found for this page. You can fill them in yourself.' };
				return;
			}
			preview = metadata;
			if (metadata.title) title = metadata.title;
			if (metadata.description) description = metadata.description;
			// Keywords only fill an empty tag field, they never replace the user's tags
			if (!tags.trim() && metadata.keywords?.length) tags = metadata.keywords.join(', ');
			status = { kind: 'ok', text: 'Details loaded from the page.' };
		} catch (err) {
			status = { kind: 'error', text: await errorMessage(err, 'Could not load details for this URL.') };
		} finally {
			fetching = false;
		}
	}

	/** Like a chat embed: pasting a URL into an empty form loads its details right away. */
	function onpaste() {
		if (title.trim() || description.trim()) return;
		// The input value updates after the paste event
		setTimeout(fetchMetadata, 0);
	}
</script>

<div>
	<Label for="{idPrefix}-url" class="mb-2">URL</Label>
	<div class="flex gap-2">
		<Input
			id="{idPrefix}-url"
			type="text"
			inputmode="url"
			placeholder="https://..."
			bind:value={url}
			required
			{onpaste}
		/>
		<Button
			color="light"
			class="shrink-0"
			disabled={!url.trim() || fetching}
			onclick={fetchMetadata}
			title="Load title, description and tags from the page"
		>
			{#if fetching}
				<Spinner size="4" class="me-2" />
			{:else}
				<WandMagicSparklesOutline class="me-2 h-4 w-4" />
			{/if}
			Fetch details
		</Button>
	</div>
	{#if status}
		<p
			class="mt-2 text-xs {status.kind === 'error'
				? 'text-red-500 dark:text-red-400'
				: status.kind === 'ok'
					? 'text-green-600 dark:text-green-400'
					: 'text-gray-500 dark:text-gray-400'}"
		>
			{status.text}
		</p>
	{/if}
</div>

{#if preview}
	{@const image = safeUrl(preview.ogImage)}
	{@const favicon = safeUrl(preview.favicon)}
	<div class="flex gap-3 rounded-lg border-l-4 border-primary-500 bg-gray-50 p-3 dark:bg-gray-900/50">
		<div class="min-w-0 flex-1">
			<p class="flex items-center gap-1.5 truncate text-xs text-gray-500 dark:text-gray-400">
				{#if favicon}
					<img src={favicon} alt="" class="h-3.5 w-3.5 rounded-sm" referrerpolicy="no-referrer" />
				{/if}
				{getDomain(preview.url)}
			</p>
			{#if preview.title}
				<p class="line-clamp-1 text-sm font-semibold text-gray-900 dark:text-white">{preview.title}</p>
			{/if}
			{#if preview.description}
				<p class="line-clamp-2 text-xs text-gray-600 dark:text-gray-300">{preview.description}</p>
			{/if}
		</div>
		{#if image}
			<img
				src={image}
				alt=""
				class="h-16 w-16 shrink-0 rounded object-cover"
				referrerpolicy="no-referrer"
				onerror={(e) => ((e.currentTarget as HTMLImageElement).style.display = 'none')}
			/>
		{/if}
	</div>
{/if}

<div>
	<Label for="{idPrefix}-title" class="mb-2">Title</Label>
	<Input id="{idPrefix}-title" type="text" placeholder="Optional – loaded from the page if empty" bind:value={title} />
</div>
<div>
	<Label for="{idPrefix}-description" class="mb-2">Description</Label>
	<Textarea id="{idPrefix}-description" placeholder="Optional" rows={3} class="w-full" bind:value={description} />
</div>
<div>
	<Label for="{idPrefix}-tags" class="mb-2">Tags</Label>
	<Input id="{idPrefix}-tags" type="text" placeholder="Comma separated" bind:value={tags} />
</div>
