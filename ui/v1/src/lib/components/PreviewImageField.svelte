<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Spinner } from 'flowbite-svelte';
	import { ImageOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { previewImage } from '../utils/links';

	/**
	 * The preview image of a link being edited: the page's (default), an upload, an own address or none.
	 * Uploads and "use the page's image" apply right away (like avatars); the address and hiding are saved with
	 * the rest of the form.
	 */
	let {
		bookmark,
		imageUrl = $bindable(''),
		hidden = $bindable(false),
		onchange,
	}: {
		bookmark: BookmarkDto;
		imageUrl?: string;
		hidden?: boolean;
		/** The link after an upload or reset. */
		onchange: (updated: BookmarkDto) => void;
	} = $props();

	let busy = $state(false);
	let error = $state('');
	let dragging = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	/** What the card will show once saved. */
	const shown = $derived(
		previewImage({ ...bookmark, customImageUrl: imageUrl.trim() || undefined, imageHidden: hidden }) ??
			(imageUrl.trim() && !hidden ? imageUrl.trim() : undefined),
	);
	const custom = $derived(!!bookmark.customImageId || !!bookmark.customImageUrl || bookmark.imageHidden);

	async function upload(file: File | undefined) {
		if (!file) return;
		busy = true;
		error = '';
		try {
			const updated = await new BookmarkApi(apiConfig()).setBookmarkImage({ bookmarkId: bookmark.id, file });
			imageUrl = '';
			hidden = false;
			onchange(updated);
		} catch (err) {
			error = await errorMessage(err, 'The image could not be uploaded.');
		} finally {
			busy = false;
		}
	}

	async function reset() {
		busy = true;
		error = '';
		try {
			const updated = await new BookmarkApi(apiConfig()).resetBookmarkImage({ bookmarkId: bookmark.id });
			imageUrl = '';
			hidden = false;
			onchange(updated);
		} catch (err) {
			error = await errorMessage(err, 'That did not work.');
		} finally {
			busy = false;
		}
	}

	/** An image copied to the clipboard can be pasted into the dialog. */
	function onpaste(e: ClipboardEvent) {
		const file = [...(e.clipboardData?.files ?? [])].find((f) => f.type.startsWith('image/'));
		if (!file) return;
		e.preventDefault();
		upload(file);
	}
</script>

<svelte:window {onpaste} />

<div>
	<Label class="mb-2">Preview image</Label>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="relative flex aspect-[2.4/1] w-full items-center justify-center overflow-hidden rounded-lg border-2 border-dashed transition-colors {dragging
			? 'border-primary-500 bg-primary-500/10'
			: 'border-gray-300 dark:border-gray-600'}"
		ondragover={(e) => (e.preventDefault(), (dragging = true))}
		ondragleave={() => (dragging = false)}
		ondrop={(e) => {
			e.preventDefault();
			dragging = false;
			upload([...(e.dataTransfer?.files ?? [])].find((f) => f.type.startsWith('image/')));
		}}
	>
		{#if shown}
			<img src={shown} alt="" referrerpolicy="no-referrer" class="h-full w-full object-cover" />
		{:else}
			<span class="flex flex-col items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
				<ImageOutline class="h-6 w-6" />
				{hidden ? 'No preview image' : 'Drop or paste an image here'}
			</span>
		{/if}
		{#if busy}
			<span class="absolute inset-0 flex items-center justify-center bg-black/40"><Spinner size="6" /></span>
		{/if}
	</div>
	<div class="mt-2 flex flex-wrap items-center gap-2">
		<input bind:this={fileInput} type="file" accept="image/*" class="hidden" onchange={(e) => upload(e.currentTarget.files?.[0])} />
		<Button size="xs" color="light" disabled={busy} onclick={() => fileInput?.click()}>Upload…</Button>
		<Button size="xs" color="light" disabled={busy} onclick={() => (hidden = !hidden)}>
			{hidden ? 'Show an image' : 'No image'}
		</Button>
		{#if custom}
			<Button size="xs" color="light" disabled={busy} onclick={reset} title="The image from the page itself">
				Use the page's image
			</Button>
		{/if}
		{#if bookmark.customImageId}
			<span class="text-xs text-gray-500 dark:text-gray-400">Uploaded image</span>
		{/if}
	</div>
	{#if !hidden}
		<Input
			class="mt-2"
			size="sm"
			type="text"
			inputmode="url"
			placeholder={bookmark.customImageId ? 'Or an image address (replaces the upload)' : 'Or an image address'}
			bind:value={imageUrl}
		/>
	{/if}
	{#if error}
		<p class="mt-1 text-xs text-red-500 dark:text-red-400">{error}</p>
	{/if}
</div>
