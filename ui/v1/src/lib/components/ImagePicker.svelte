<script lang="ts">
	import { Button, Input } from 'flowbite-svelte';
	import { LinkOutline, UploadOutline } from 'flowbite-svelte-icons';
	import { errorMessage } from '../api/client';
	import Avatar from './Avatar.svelte';

	let {
		name,
		imageId,
		shape = 'rounded-full',
		onupload,
		onurl,
		onremove,
	}: {
		/** Used for the initials fallback. */
		name: string;
		imageId?: string | null;
		shape?: 'rounded-full' | 'rounded-2xl';
		onupload: (file: File) => Promise<void>;
		onurl: (url: string) => Promise<void>;
		onremove: () => Promise<void>;
	} = $props();

	const ACCEPT = 'image/png,image/jpeg,image/gif,image/webp';
	const MAX_BYTES = 5 * 1024 * 1024;

	let fileInput = $state<HTMLInputElement | null>(null);
	let showUrl = $state(false);
	let url = $state('');
	let busy = $state(false);
	let error = $state('');
	let dragging = $state(false);

	async function run(action: () => Promise<void>) {
		busy = true;
		error = '';
		try {
			await action();
			showUrl = false;
			url = '';
		} catch (err) {
			error = await errorMessage(err, 'Failed to update the image.');
		} finally {
			busy = false;
		}
	}

	function upload(file: File | undefined) {
		if (!file) return;
		if (!ACCEPT.split(',').includes(file.type)) {
			error = 'Please choose a PNG, JPEG, GIF or WebP image.';
			return;
		}
		if (file.size > MAX_BYTES) {
			error = 'Images must be smaller than 5 MB.';
			return;
		}
		run(() => onupload(file));
	}

	function onfilechange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		upload(input.files?.[0]);
		input.value = '';
	}

	function ondrop(e: DragEvent) {
		e.preventDefault();
		dragging = false;
		upload(e.dataTransfer?.files?.[0]);
	}
</script>

<div class="flex flex-col gap-3">
	<div class="flex items-center gap-4">
		<button
			type="button"
			title="Upload an image (or drop one here)"
			class="relative shrink-0 rounded-full outline-offset-4 {dragging ? 'outline-2 outline-dashed outline-primary-500' : ''}"
			disabled={busy}
			onclick={() => fileInput?.click()}
			ondragover={(e) => (e.preventDefault(), (dragging = true))}
			ondragleave={() => (dragging = false)}
			{ondrop}
		>
			<Avatar {name} {imageId} class="h-20 w-20 text-2xl {shape} {busy ? 'opacity-50' : ''}" />
		</button>
		<div class="flex flex-wrap gap-2">
			<Button size="xs" color="light" disabled={busy} onclick={() => fileInput?.click()}>
				<UploadOutline class="me-1.5 h-4 w-4" /> Upload
			</Button>
			<Button size="xs" color="light" disabled={busy} onclick={() => (showUrl = !showUrl)}>
				<LinkOutline class="me-1.5 h-4 w-4" /> From URL
			</Button>
			{#if imageId}
				<Button size="xs" color="light" disabled={busy} onclick={() => run(onremove)}>Remove</Button>
			{/if}
		</div>
		<input bind:this={fileInput} type="file" accept={ACCEPT} class="hidden" onchange={onfilechange} />
	</div>
	{#if showUrl}
		<form
			class="flex gap-2"
			onsubmit={(e) => {
				e.preventDefault();
				if (url.trim()) run(() => onurl(url.trim()));
			}}
		>
			<Input type="url" size="sm" placeholder="https://example.com/logo.png" bind:value={url} required />
			<Button size="sm" type="submit" disabled={busy || !url.trim()}>{busy ? 'Loading…' : 'Load'}</Button>
		</form>
	{/if}
	<p class="text-xs text-gray-500 dark:text-gray-400">
		PNG, JPEG, GIF or WebP up to 5 MB. Images are cropped to a square.
	</p>
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
</div>
