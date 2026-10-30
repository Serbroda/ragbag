<script lang="ts">
	import { BookmarkApi, CollectionApi } from 'ragbag-frontend-sdk';
	import type { CollectionDto } from 'ragbag-frontend-sdk';
	import { Button, Label, Select } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { navigate, route } from '../router';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import LinkFormFields from '../components/LinkFormFields.svelte';
	import { collectionsToTree, flattenTree } from '../components/tree';
	import { lastSpaceId, loadSpaces, rememberSpace, session } from '../stores/spaces.svelte';

	/**
	 * Adding a link from anywhere: the share menu of the installed app (Android) and the bookmarklet open this page
	 * with ?url=…&title=… . Space and collection are remembered, so it is mostly "Add" and done.
	 */
	const param = (name: string) => {
		const value = route.search[name];
		return typeof value === 'string' ? value : value == null ? '' : String(value);
	};
	// Share menus often put the address into "text", sometimes together with a title
	const sharedText = param('text');
	const sharedUrl = param('url') || sharedText.match(/https?:\/\/\S+/)?.[0] || '';
	const sharedTitle = param('title') || (sharedText && !sharedText.startsWith('http') ? sharedText.replace(sharedUrl, '').trim() : '');

	let url = $state(sharedUrl);
	let title = $state(sharedTitle);
	let description = $state('');
	let tags = $state('');
	let spaceId = $state('');
	let collectionId = $state('');
	let collections = $state<CollectionDto[]>([]);
	let saving = $state(false);
	let error = $state('');
	let saved = $state<{ spaceId: string; collectionId: string; name: string } | null>(null);
	let form = $state<LinkFormFields | null>(null);

	const options = $derived(flattenTree(collectionsToTree(collections, spaceId)));
	const storageKey = (space: string) => `quick-add:${space}`;

	loadSpaces()
		.then(() => {
			const last = lastSpaceId();
			spaceId = session.spaces.find((s) => s.id === last)?.id ?? session.spaces[0]?.id ?? '';
		})
		.catch(() => (error = 'Could not load your spaces.'));

	$effect(() => {
		const space = spaceId;
		if (!space) return;
		new CollectionApi(apiConfig())
			.getCollections({ spaceId: space })
			.then((loaded) => {
				if (space !== spaceId) return;
				collections = loaded;
				let remembered: string | null = null;
				try {
					remembered = localStorage.getItem(storageKey(space));
				} catch {
					// Unavailable storage: the first collection
				}
				const flat = flattenTree(collectionsToTree(loaded, space));
				collectionId = flat.some((o) => o.id === remembered) ? remembered! : (flat[0]?.id ?? '');
				// Load title, description and tags of a shared link right away (once a collection is known)
				if (sharedUrl && !sharedTitle && collectionId) setTimeout(() => form?.fetchMetadata(), 0);
			})
			.catch(() => (collections = []));
	});

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
				localStorage.setItem(storageKey(spaceId), collectionId);
				rememberSpace(spaceId);
			} catch {
				// Only the defaults for next time are lost
			}
			saved = { spaceId, collectionId, name: options.find((o) => o.id === collectionId)?.label ?? 'collection' };
		} catch (err) {
			error = await errorMessage(err, 'Failed to add link.');
		} finally {
			saving = false;
		}
	}

	/** A popup from the bookmarklet closes itself; otherwise go to the collection. */
	function done() {
		if (window.opener) {
			window.close();
			return;
		}
		if (saved) {
			navigate('/space/:spaceId/collections/:collectionId', {
				params: { spaceId: saved.spaceId, collectionId: saved.collectionId },
			});
		}
	}
</script>

<CenteredScreen subtitle={saved ? undefined : 'Save a link'}>
	{#if saved}
		<div class="flex flex-col items-center gap-4 text-center">
			<p class="text-lg font-semibold text-white">Saved to {saved.name} ✓</p>
			<div class="flex gap-2">
				<Button onclick={done}>{window.opener ? 'Close' : 'Open collection'}</Button>
				<Button
					color="alternative"
					onclick={() => ((saved = null), (url = ''), (title = ''), (description = ''), (tags = ''))}>Add another</Button
				>
			</div>
		</div>
	{:else if session.loaded && session.spaces.length === 0}
		<p class="text-center text-gray-300">Create a space in Ragbag first, then you can save links to it.</p>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={save}>
			<div class="grid grid-cols-2 gap-3">
				<div>
					<Label for="add-space" class="mb-2">Space</Label>
					<Select id="add-space" placeholder="" bind:value={spaceId}>
						{#each session.spaces as space (space.id)}
							<option value={space.id}>{space.name}</option>
						{/each}
					</Select>
				</div>
				<div>
					<Label for="add-collection" class="mb-2">Collection</Label>
					<Select id="add-collection" placeholder="" bind:value={collectionId} required>
						{#each options as option (option.id)}
							<option value={option.id}>{'─'.repeat(option.depth)} {option.label}</option>
						{/each}
					</Select>
				</div>
			</div>
			{#if spaceId && collectionId}
				<LinkFormFields
					bind:this={form}
					{collectionId}
					{spaceId}
					idPrefix="add-page"
					bind:url
					bind:title
					bind:description
					bind:tags
				/>
			{:else if spaceId && collections.length === 0}
				<p class="text-sm text-gray-400">This space has no collections yet.</p>
			{/if}
			{#if error}
				<p class="text-sm text-red-400">{error}</p>
			{/if}
			<Button type="submit" disabled={saving || !url.trim() || !collectionId}>{saving ? 'Saving...' : 'Add link'}</Button>
		</form>
	{/if}
</CenteredScreen>
