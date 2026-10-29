<script lang="ts">
	import { CollectionApi } from 'ragbag-frontend-sdk';
	import type { CollectionShareDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Modal } from 'flowbite-svelte';
	import { GlobeOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage, shareLink } from '../api/client';
	import { showToast } from '../stores/toast.svelte';
	import { copyText } from '../utils/clipboard';

	let {
		open = $bindable(false),
		collectionId,
		collectionName,
		canManage,
		privateReason,
		onchanged,
	}: {
		open?: boolean;
		collectionId: string;
		collectionName: string;
		/** Creating and revoking the link needs EDIT_COLLECTIONS. */
		canManage: boolean;
		/** Set when the collection is private or inside a private one: such collections cannot be shared. */
		privateReason?: string;
		onchanged?: () => void;
	} = $props();

	const api = new CollectionApi(apiConfig());

	let share = $state<CollectionShareDto | null>(null);
	let busy = $state(false);
	let error = $state('');
	let confirmRevoke = $state(false);

	$effect(() => {
		if (open) load();
	});

	async function load() {
		error = '';
		confirmRevoke = false;
		try {
			share = await api.getCollectionShare({ collectionId });
		} catch (err) {
			error = await errorMessage(err, 'Could not load the link.');
		}
	}

	async function run(action: () => Promise<void>, fallback: string) {
		busy = true;
		error = '';
		try {
			await action();
			onchanged?.();
		} catch (err) {
			error = await errorMessage(err, fallback);
		} finally {
			busy = false;
			confirmRevoke = false;
		}
	}

	const create = () =>
		run(async () => {
			share = await api.shareCollection({ collectionId });
			await copy();
		}, 'Could not create the link.');

	const revoke = () =>
		run(async () => {
			await api.unshareCollection({ collectionId });
			share = { shared: false };
			showToast('Link revoked');
		}, 'Could not revoke the link.');

	async function copy() {
		if (!share?.token) return;
		showToast((await copyText(shareLink(share.token))) ? 'Link copied' : 'Copy the link from the field');
	}
</script>

<Modal title="Share “{collectionName}”" bind:open size="sm" autoclose={false}>
	<div class="flex flex-col gap-4 text-sm">
		<p class="flex gap-3 text-gray-600 dark:text-gray-300">
			<GlobeOutline class="mt-0.5 h-5 w-5 shrink-0 text-primary-500" />
			<span>
				Anybody with the link can see this collection and its sub-collections, without an account. Private
				sub-collections, the rest of the space and who added what stay hidden.
			</span>
		</p>

		{#if privateReason}
			<p class="rounded-md bg-amber-500/10 p-3 text-amber-700 dark:text-amber-300">{privateReason}</p>
		{:else if share?.shared && share.token}
			<div class="flex gap-2">
				<Input readonly value={shareLink(share.token)} class="font-mono text-xs" onfocus={(e) => (e.currentTarget as HTMLInputElement).select()} />
				<Button class="shrink-0" onclick={copy}>Copy</Button>
			</div>
			{#if confirmRevoke}
				<div class="flex flex-wrap items-center justify-end gap-2">
					<span class="me-auto text-gray-600 dark:text-gray-300">The link stops working for everyone.</span>
					<Button size="sm" color="light" onclick={() => (confirmRevoke = false)}>Cancel</Button>
					<Button size="sm" color="red" disabled={busy} onclick={revoke}>Revoke link</Button>
				</div>
			{:else}
				<div class="flex flex-wrap justify-end gap-2">
					<Button size="sm" color="light" disabled={busy} onclick={create} title="The current link stops working">
						New link
					</Button>
					<Button size="sm" color="light" disabled={busy} onclick={() => (confirmRevoke = true)}>Revoke</Button>
				</div>
			{/if}
		{:else if share?.shared}
			<p class="text-gray-600 dark:text-gray-300">This collection is shared publicly. Ask someone who can edit it for the link.</p>
		{:else if share && canManage}
			<div class="flex justify-end">
				<Button disabled={busy} onclick={create}>Create link</Button>
			</div>
		{:else if share}
			<p class="text-gray-600 dark:text-gray-300">This collection is not shared. People who can edit it can create a link.</p>
		{/if}

		{#if error}
			<p class="text-red-500 dark:text-red-400">{error}</p>
		{/if}
	</div>
</Modal>
