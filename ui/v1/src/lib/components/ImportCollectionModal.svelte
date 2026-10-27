<script lang="ts">
	import { CollectionApi } from 'ragbag-frontend-sdk';
	import type { ImportResultDto } from 'ragbag-frontend-sdk';
	import { untrack } from 'svelte';
	import { Button, Modal } from 'flowbite-svelte';
	import { UploadOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';

	let {
		open = $bindable(false),
		collectionId,
		collectionName,
		ondone,
	}: {
		open?: boolean;
		collectionId: string;
		collectionName: string;
		ondone?: () => void;
	} = $props();

	const api = new CollectionApi(apiConfig());

	let fileInput = $state<HTMLInputElement | null>(null);
	let fileName = $state('');
	let content = $state<object | null>(null);
	let preview = $state<ImportResultDto | null>(null);
	let result = $state<ImportResultDto | null>(null);
	let busy = $state(false);
	let error = $state('');

	$effect(() => {
		if (!open) return;
		untrack(() => {
			fileName = '';
			content = null;
			preview = null;
			result = null;
			error = '';
		});
	});

	async function choose(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';
		if (!file) return;
		fileName = file.name;
		preview = null;
		result = null;
		error = '';
		try {
			content = JSON.parse(await file.text());
		} catch {
			content = null;
			error = 'This file is not valid JSON.';
			return;
		}
		busy = true;
		try {
			// The server runs the whole import and rolls it back, so the preview is exact
			preview = await api.importCollection({ collectionId, body: content!, preview: true });
		} catch (err) {
			error = await errorMessage(err, 'This file cannot be imported.');
		} finally {
			busy = false;
		}
	}

	async function runImport() {
		if (!content) return;
		busy = true;
		error = '';
		try {
			result = await api.importCollection({ collectionId, body: content, preview: false });
			ondone?.();
		} catch (err) {
			error = await errorMessage(err, 'Import failed.');
		} finally {
			busy = false;
		}
	}

	function plural(count: number, word: string) {
		return `${count} ${word}${count === 1 ? '' : 's'}`;
	}
</script>

{#snippet summary(r: ImportResultDto, done: boolean)}
	<ul class="space-y-1 text-sm text-gray-700 dark:text-gray-300">
		<li>
			<strong>{plural(r.bookmarksImported, 'link')}</strong>
			{done ? 'imported' : 'will be imported'}
		</li>
		{#if r.collectionsCreated > 0}
			<li>{plural(r.collectionsCreated, 'sub-collection')} {done ? 'created' : 'will be created'}</li>
		{/if}
		{#if r.collectionsMerged > 0}
			<li>{plural(r.collectionsMerged, 'sub-collection')} merged with existing ones of the same name</li>
		{/if}
		{#if r.bookmarksSkipped > 0}
			<li class="text-gray-500">{plural(r.bookmarksSkipped, 'link')} skipped, already in the collection</li>
		{/if}
		{#if r.bookmarksInvalid > 0}
			<li class="text-amber-600 dark:text-amber-400">
				{plural(r.bookmarksInvalid, 'link')} skipped, no valid http(s) URL
			</li>
		{/if}
	</ul>
	<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">
		Exported by Ragbag {r.appVersion || '?'} · format version {r.formatVersion}
	</p>
{/snippet}

<Modal bind:open size="sm" title="Import into “{collectionName}”" autoclose={false}>
	{#if result}
		<p class="mb-3 font-medium text-green-600 dark:text-green-400">Import finished.</p>
		{@render summary(result, true)}
	{:else}
		<p class="text-sm text-gray-500 dark:text-gray-400">
			Choose a Ragbag export (.json). Its links are added here and its sub-collections are merged with
			existing ones of the same name. You see what happens before anything is changed.
		</p>
		<div class="flex items-center gap-3">
			<Button color="light" size="sm" disabled={busy} onclick={() => fileInput?.click()}>
				<UploadOutline class="me-1.5 h-4 w-4" /> Choose file
			</Button>
			<span class="min-w-0 truncate text-sm text-gray-600 dark:text-gray-300">{fileName}</span>
			<input bind:this={fileInput} type="file" accept="application/json,.json" class="hidden" onchange={choose} />
		</div>
		{#if busy && !preview}
			<p class="text-sm text-gray-500">Checking file…</p>
		{/if}
		{#if preview}
			<div class="rounded-lg bg-gray-50 p-3 dark:bg-gray-900">{@render summary(preview, false)}</div>
		{/if}
	{/if}
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			{#if result}
				<Button onclick={() => (open = false)}>Close</Button>
			{:else}
				<Button color="alternative" onclick={() => (open = false)}>Cancel</Button>
				<Button disabled={busy || !preview} onclick={runImport}>{busy && preview ? 'Importing…' : 'Import'}</Button>
			{/if}
		</div>
	{/snippet}
</Modal>
