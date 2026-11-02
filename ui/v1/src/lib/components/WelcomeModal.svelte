<script lang="ts">
	import type { SpaceDto } from 'ragbag-frontend-sdk';
	import { Button, Modal } from 'flowbite-svelte';
	import Avatar from './Avatar.svelte';

	/**
	 * The welcome of a space: the introduction for the owner of a new account's first space, else the space's
	 * welcome message (Markdown rendered by the server, without raw HTML).
	 */
	let {
		open = $bindable(false),
		space,
		intro,
		oninvite,
		onsettings,
	}: {
		open?: boolean;
		space: SpaceDto;
		/** The owner's introduction instead of the welcome message */
		intro: boolean;
		oninvite?: () => void;
		onsettings?: () => void;
	} = $props();

	const canInvite = $derived(space.permissions?.includes('MANAGE_MEMBERS') ?? false);

	/** Links of the message lead away from Ragbag: open them in a new tab. */
	function externalLinks(node: HTMLElement) {
		const update = () => {
			for (const link of node.querySelectorAll('a')) {
				link.target = '_blank';
				link.rel = 'noopener noreferrer';
			}
		};
		update();
		const observer = new MutationObserver(update);
		observer.observe(node, { childList: true, subtree: true });
		return { destroy: () => observer.disconnect() };
	}

	function then(action?: () => void) {
		open = false;
		action?.();
	}
</script>

<Modal bind:open size="md">
	<div class="flex flex-col items-center text-center">
		<Avatar name={space.name} imageId={space.iconId} class="mb-4 h-20 w-20 rounded-2xl text-2xl" />
		<p class="text-sm text-gray-500 dark:text-gray-400">{intro ? 'Welcome to your own space' : 'Welcome to'}</p>
		<h2 class="text-xl font-semibold text-gray-900 dark:text-white">{space.name}</h2>
	</div>

	{#if intro}
		<ul class="space-y-3 text-sm text-gray-700 dark:text-gray-300">
			<li>
				<span class="font-semibold text-gray-900 dark:text-white">Collect links</span> in collections (the sidebar),
				nested as deep as you like. Private collections only you and the people you pick can see.
			</li>
			<li>
				<span class="font-semibold text-gray-900 dark:text-white">Share the space</span> by inviting people with a
				link: as viewers, or as contributors who add links of their own.
			</li>
			<li>
				<span class="font-semibold text-gray-900 dark:text-white">Make it yours:</span> we picked a random planet as
				its icon. Icon, name and a welcome message for new members are in the space settings.
			</li>
			<li>
				<span class="font-semibold text-gray-900 dark:text-white">Save from anywhere</span> with the bookmarklet in
				your profile, or the share menu of your phone once Ragbag is installed.
			</li>
		</ul>
		<div class="flex flex-wrap justify-end gap-2">
			{#if canInvite}
				<Button color="light" onclick={() => then(oninvite)}>Invite people</Button>
			{/if}
			<Button color="light" onclick={() => then(onsettings)}>Space settings</Button>
			<Button onclick={() => then()}>Let's go</Button>
		</div>
	{:else}
		<div class="markdown max-h-[50vh] overflow-y-auto" use:externalLinks>
			{@html space.welcomeHtml ?? ''}
		</div>
		<div class="flex justify-end">
			<Button onclick={() => then()}>Got it</Button>
		</div>
	{/if}
</Modal>
