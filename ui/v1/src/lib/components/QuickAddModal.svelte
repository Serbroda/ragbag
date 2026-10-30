<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import { Button, Label, Modal, Select } from 'flowbite-svelte';
	import { getContext, tick } from 'svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { navigate } from '../router';
	import { showToast } from '../stores/toast.svelte';
	import LinkFormFields from './LinkFormFields.svelte';
	import { flattenTree, type TreeNode } from './tree';

	/**
	 * Adding a link without opening a collection first (from the dashboard): the collection is picked in the
	 * dialog and remembered per space, so the next time pasting the URL and pressing Enter is enough.
	 */
	let {
		open = $bindable(false),
		spaceId,
		onadded,
	}: { open?: boolean; spaceId: string; onadded?: () => void } = $props();

	const collectionTree = getContext<(() => TreeNode[]) | undefined>('collection-tree');
	const reloadSidebar = getContext<(() => void) | undefined>('reload-collections');
	const options = $derived(flattenTree(collectionTree?.() ?? []));

	let collectionId = $state('');
	let url = $state('');
	let title = $state('');
	let description = $state('');
	let tags = $state('');
	let saving = $state(false);
	let error = $state('');
	let form = $state<LinkFormFields | null>(null);

	const storageKey = $derived(`quick-add:${spaceId}`);

	/** Opens the dialog, optionally with a URL (pasted on the page) whose details are loaded right away. */
	export async function show(pastedUrl = '') {
		url = pastedUrl;
		title = description = tags = error = '';
		let remembered: string | null = null;
		try {
			remembered = localStorage.getItem(storageKey);
		} catch {
			// Unavailable storage: start with the first collection
		}
		collectionId = options.some((option) => option.id === remembered) ? remembered! : (options[0]?.id ?? '');
		open = true;
		await tick();
		if (pastedUrl && collectionId) form?.fetchMetadata();
		else setTimeout(() => document.getElementById('quick-add-url')?.focus(), 50);
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!collectionId || !url.trim()) return;
		saving = true;
		error = '';
		try {
			await new BookmarkApi(apiConfig()).createBookmark({
				collectionId,
				createBookmarkDto: {
					url,
					title: title || undefined,
					description: description || undefined,
					tags: tags
						.split(',')
						.map((tag) => tag.trim())
						.filter(Boolean),
				},
			});
			try {
				localStorage.setItem(storageKey, collectionId);
			} catch {
				// Only the default for next time is lost
			}
			open = false;
			const target = collectionId;
			const name = options.find((option) => option.id === target)?.label ?? 'collection';
			showToast(`Added to ${name}`, {
				label: 'Open',
				run: () =>
					navigate('/space/:spaceId/collections/:collectionId', { params: { spaceId, collectionId: target } }),
			});
			reloadSidebar?.();
			onadded?.();
		} catch (err) {
			error = await errorMessage(err, 'Failed to add link.');
		} finally {
			saving = false;
		}
	}
</script>

<Modal title="Add Link" bind:open size="sm" autoclose={false}>
	{#if options.length === 0}
		<p class="text-sm text-gray-600 dark:text-gray-300">
			This space has no collections yet. Create one in the sidebar first, then add links to it.
		</p>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={save}>
			<div>
				<Label for="quick-add-collection" class="mb-2">Collection</Label>
				<Select id="quick-add-collection" placeholder="" bind:value={collectionId} required>
					{#each options as option (option.id)}
						<option value={option.id}>{'─'.repeat(option.depth)} {option.label}</option>
					{/each}
				</Select>
			</div>
			<LinkFormFields
				bind:this={form}
				{collectionId}
				idPrefix="quick-add"
				bind:url
				bind:title
				bind:description
				bind:tags
			/>
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end gap-2">
				<Button color="light" onclick={() => (open = false)}>Cancel</Button>
				<Button type="submit" disabled={saving || !url.trim()}>{saving ? 'Saving...' : 'Add'}</Button>
			</div>
		</form>
	{/if}
</Modal>
