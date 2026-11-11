<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { SpaceWebhookDto } from 'ragbag-frontend-sdk';
	import { Button, Input } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { showToast } from '../stores/toast.svelte';
	import SettingsItem from './SettingsItem.svelte';

	/**
	 * The Discord webhook of a space (members with EDIT_SPACE): new links are posted to its channel. The address
	 * is a secret, so it is shown shortened once it is saved.
	 */
	let { spaceId }: { spaceId: string } = $props();

	const spaceApi = new SpaceApi(apiConfig());

	let webhook = $state<SpaceWebhookDto | null>(null);
	let editing = $state(false);
	let url = $state('');
	let busy = $state(false);
	let error = $state('');

	$effect(() => {
		const id = spaceId;
		webhook = null;
		editing = false;
		url = error = '';
		spaceApi
			.getSpaceWebhook({ spaceId: id })
			.then((loaded) => id === spaceId && (webhook = loaded))
			.catch(async (err) => (error = await errorMessage(err, 'Could not load the integration.')));
	});

	const connected = $derived(!!webhook?.url);
	const broken = $derived(connected && !!webhook?.failedAt);
	/** Enough to recognize the webhook, not enough to post with it. */
	const shortened = $derived((webhook?.url ?? '').replace(/(\/api\/webhooks\/\d+\/).+$/, '$1…'));

	async function run(action: () => Promise<void>, fallback: string) {
		busy = true;
		error = '';
		try {
			await action();
		} catch (err) {
			error = await errorMessage(err, fallback);
		} finally {
			busy = false;
		}
	}

	function save(e: Event) {
		e.preventDefault();
		run(async () => {
			webhook = await spaceApi.setSpaceWebhook({ spaceId, setSpaceWebhookRequest: { url } });
			editing = false;
			url = '';
			showToast('Discord is connected. Send a test message to see it work.');
		}, 'Failed to save the webhook.');
	}

	function test() {
		run(async () => {
			await spaceApi.testSpaceWebhook({ spaceId });
			if (webhook) webhook = { ...webhook, failedAt: undefined };
			showToast('Test message sent: look into the channel.');
		}, 'The test message could not be sent.');
	}

	function remove() {
		run(async () => {
			await spaceApi.deleteSpaceWebhook({ spaceId });
			webhook = {};
			showToast('Discord is disconnected.');
		}, 'Failed to remove the webhook.');
	}
</script>

<SettingsItem
	title="Discord"
	status={webhook ? (broken ? 'Not working' : connected ? 'Connected' : 'Not connected') : undefined}
	tone={connected && !broken ? 'good' : 'neutral'}
	description="Posts every link somebody saves in this space to a Discord channel. Links in private collections and imported links are never posted."
	action={connected ? undefined : 'Connect…'}
	bind:open={editing}
>
	{#if connected && !editing}
		{#if broken}
			<p class="mb-3 rounded-lg bg-red-500/10 p-3 text-sm text-red-700 ring-1 ring-inset ring-red-500/20 dark:text-red-300">
				Discord no longer knows this webhook (was it deleted?), so nothing is posted. Set a new one, or send a
				test message if it should work again.
			</p>
		{/if}
		<div class="flex flex-wrap items-center gap-2">
			<code class="me-auto min-w-0 truncate text-xs text-gray-600 dark:text-gray-300">{shortened}</code>
			<Button size="sm" color="light" disabled={busy} onclick={test}>Send test message</Button>
			<Button size="sm" color="light" disabled={busy} onclick={() => ((editing = true), (error = ''))}>Change…</Button>
			<Button size="sm" color="light" disabled={busy} onclick={remove}>Disconnect</Button>
		</div>
		{#if error}
			<p class="mt-2 text-sm text-red-500 dark:text-red-400">{error}</p>
		{/if}
	{:else}
		<form class="flex flex-col gap-2" onsubmit={save}>
			<p class="text-sm text-gray-700 dark:text-gray-300">
				In Discord: channel settings → Integrations → Webhooks → New webhook → Copy webhook URL.
			</p>
			<!-- svelte-ignore a11y_autofocus -->
			<Input
				type="url"
				placeholder="https://discord.com/api/webhooks/…"
				aria-label="Discord webhook URL"
				autocomplete="off"
				bind:value={url}
				required
				autofocus
			/>
			<p class="text-xs text-gray-500 dark:text-gray-400">
				Treat the address like a password: whoever has it can post to that channel. Only people who may edit this
				space see it here.
			</p>
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end gap-2">
				<Button size="sm" color="light" onclick={() => ((editing = false), (url = error = ''))}>Cancel</Button>
				<Button type="submit" size="sm" disabled={busy || !url.trim()}>{busy ? 'Saving…' : 'Save'}</Button>
			</div>
		</form>
	{/if}
</SettingsItem>
