<script lang="ts">
	import { CollectionApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { SpaceMemberDto } from 'ragbag-frontend-sdk';
	import { untrack } from 'svelte';
	import { Button, Modal, Toggle } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { session } from '../stores/spaces.svelte';
	import MemberChecklist from './MemberChecklist.svelte';

	let {
		open = $bindable(false),
		collectionId,
		collectionName,
		spaceId,
		onchanged,
	}: {
		open?: boolean;
		collectionId: string;
		collectionName: string;
		spaceId: string;
		onchanged?: () => void;
	} = $props();

	let members = $state<SpaceMemberDto[]>([]);
	let restricted = $state(false);
	let selected = $state(new Set<string>());
	let canManage = $state(false);
	let loading = $state(false);
	let saving = $state(false);
	let error = $state('');

	$effect(() => {
		if (!open) return;
		untrack(load);
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const [access, spaceMembers] = await Promise.all([
				new CollectionApi(apiConfig()).getCollectionAccess({ collectionId }),
				new SpaceApi(apiConfig()).getSpaceMembers({ spaceId }),
			]);
			restricted = access.restricted;
			canManage = access.canManage;
			selected = new Set(access.members.map((m) => m.userId));
			members = spaceMembers;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load who can see this collection.');
		} finally {
			loading = false;
		}
	}

	async function save() {
		saving = true;
		error = '';
		try {
			await new CollectionApi(apiConfig()).updateCollectionAccess({
				collectionId,
				updateCollectionAccessDto: { restricted, memberIds: restricted ? [...selected] : [] },
			});
			open = false;
			onchanged?.();
		} catch (err) {
			error = await errorMessage(err, 'Failed to save.');
		} finally {
			saving = false;
		}
	}
</script>

<Modal bind:open size="sm" title="Who can see “{collectionName}”?" autoclose={false}>
	{#if loading}
		<p class="text-sm text-gray-500 dark:text-gray-400">Loading…</p>
	{:else}
		<div class="flex flex-col gap-4">
			<Toggle bind:checked={restricted} disabled={!canManage}>Private collection</Toggle>
			<p class="-mt-2 text-sm text-gray-500 dark:text-gray-400">
				{#if restricted}
					Only the members selected below see this collection, its sub-collections and links. Not even space
					owners and admins can see it otherwise.
				{:else}
					Everyone in the space can see this collection.
				{/if}
			</p>
			{#if restricted}
				<MemberChecklist {members} bind:selected currentUserId={session.me?.id} disabled={!canManage} />
			{/if}
			{#if !canManage}
				<p class="text-sm text-amber-600 dark:text-amber-400">
					Only the creator of this collection or a space admin can change this.
				</p>
			{/if}
		</div>
	{/if}
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			<Button color="alternative" onclick={() => (open = false)}>Cancel</Button>
			{#if canManage}
				<Button disabled={saving || loading} onclick={save}>{saving ? 'Saving…' : 'Save'}</Button>
			{/if}
		</div>
	{/snippet}
</Modal>
