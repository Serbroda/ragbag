<script lang="ts">
	import { InviteApi, SpaceApi } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { ArrowLeftOutline, PlusOutline, UserAddSolid } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';

	let {
		open = $bindable(false),
		ondone,
	}: {
		open?: boolean;
		/** Called with the id of the created or joined space. */
		ondone: (spaceId: string) => void | Promise<void>;
	} = $props();

	type Step = 'choose' | 'create' | 'join';
	let step = $state<Step>('choose');
	let name = $state('');
	let description = $state('');
	let invite = $state('');
	let error = $state('');
	let busy = $state(false);

	$effect(() => {
		if (open) {
			step = 'choose';
			name = '';
			description = '';
			invite = '';
			error = '';
		}
	});

	/** Accepts a full invite link (".../invite/<token>") or just the token. */
	function extractToken(value: string): string {
		const trimmed = value.trim();
		const match = trimmed.match(/\/invite\/([^/?#\s]+)/);
		return match ? match[1] : trimmed;
	}

	async function submit(e: Event) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			if (step === 'create') {
				const space = await new SpaceApi(apiConfig()).createSpace({
					createSpaceDto: { name: name.trim(), description: description.trim() || undefined },
				});
				await ondone(space.id);
			} else {
				const preview = await new InviteApi(apiConfig()).acceptInvite({ token: extractToken(invite) });
				await ondone(preview.targetId);
			}
			open = false;
		} catch (err) {
			error = await errorMessage(
				err,
				step === 'create' ? 'Failed to create space.' : 'This invite is invalid or has expired.',
			);
		} finally {
			busy = false;
		}
	}
</script>

<Modal bind:open size="sm" autoclose={false} title={step === 'join' ? 'Join a space' : 'Create a space'}>
	{#if step === 'choose'}
		<p class="text-center text-sm text-gray-500 dark:text-gray-400">
			A space is where you and your people collect links together.
		</p>
		<button
			type="button"
			class="flex w-full items-center gap-3 rounded-lg border border-gray-200 p-4 text-left hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
			onclick={() => (step = 'create')}
		>
			<span class="flex h-10 w-10 items-center justify-center rounded-full bg-green-500 text-white">
				<PlusOutline class="h-5 w-5" />
			</span>
			<span>
				<span class="block font-semibold text-gray-900 dark:text-white">Create my own</span>
				<span class="block text-sm text-gray-500 dark:text-gray-400">Start a new space and invite others</span>
			</span>
		</button>
		<button
			type="button"
			class="flex w-full items-center gap-3 rounded-lg border border-gray-200 p-4 text-left hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
			onclick={() => (step = 'join')}
		>
			<span class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-500 text-white">
				<UserAddSolid class="h-5 w-5" />
			</span>
			<span>
				<span class="block font-semibold text-gray-900 dark:text-white">Join a space</span>
				<span class="block text-sm text-gray-500 dark:text-gray-400">You have an invite link</span>
			</span>
		</button>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={submit}>
			{#if step === 'create'}
				<div>
					<Label for="new-space-name" class="mb-2">Name</Label>
					<Input id="new-space-name" bind:value={name} required maxlength={80} placeholder="e.g. Team Research" />
				</div>
				<div>
					<Label for="new-space-description" class="mb-2">Description (optional)</Label>
					<Input id="new-space-description" bind:value={description} />
				</div>
			{:else}
				<div>
					<Label for="invite-link" class="mb-2">Invite link</Label>
					<Input
						id="invite-link"
						bind:value={invite}
						required
						placeholder="https://…/invite/AbC123…"
					/>
					<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">Paste the full link or just the code.</p>
				</div>
			{/if}
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex items-center justify-between gap-2">
				<Button color="light" onclick={() => ((step = 'choose'), (error = ''))}>
					<ArrowLeftOutline class="me-1 h-4 w-4" /> Back
				</Button>
				<Button type="submit" disabled={busy || (step === 'create' ? !name.trim() : !invite.trim())}>
					{#if step === 'create'}
						{busy ? 'Creating...' : 'Create'}
					{:else}
						{busy ? 'Joining...' : 'Join space'}
					{/if}
				</Button>
			</div>
		</form>
	{/if}
</Modal>
