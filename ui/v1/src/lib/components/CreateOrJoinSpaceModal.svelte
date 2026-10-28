<script lang="ts">
	import { InviteApi, SpaceApi } from 'ragbag-frontend-sdk';
	import type { DiscoverableSpaceDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { ArrowLeftOutline, GlobeOutline, PlusOutline, UserAddSolid } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import Avatar from './Avatar.svelte';

	type Step = 'choose' | 'create' | 'join' | 'discover';

	let {
		open = $bindable(false),
		start = 'choose',
		ondone,
	}: {
		open?: boolean;
		/** The step shown when the modal opens, e.g. 'discover' for the space bar's discover button. */
		start?: Step;
		/** Called with the id of the created or joined space. */
		ondone: (spaceId: string) => void | Promise<void>;
	} = $props();

	let step = $state<Step>('choose');
	let name = $state('');
	let description = $state('');
	let invite = $state('');
	let error = $state('');
	let busy = $state(false);

	$effect(() => {
		if (open) {
			step = start;
			name = '';
			description = '';
			invite = '';
			error = '';
			query = '';
		}
	});

	// --- Discover ---
	const TITLES: Record<Step, string> = {
		choose: 'Create a space',
		create: 'Create a space',
		join: 'Join a space',
		discover: 'Discover spaces',
	};
	const JOIN_ROLE_LABELS = { VIEWER: 'join as viewer', CONTRIBUTOR: 'join as contributor' } as const;
	let query = $state('');
	let discovered = $state<DiscoverableSpaceDto[]>([]);
	let searching = $state(false);
	let joiningId = $state<string | null>(null);
	let searchTimer: ReturnType<typeof setTimeout> | undefined;

	$effect(() => {
		if (!open || step !== 'discover') return;
		const q = query.trim();
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => discover(q), q ? 250 : 0);
		return () => clearTimeout(searchTimer);
	});

	async function discover(q: string) {
		searching = true;
		error = '';
		try {
			const result = await new SpaceApi(apiConfig()).discoverSpaces({ q: q || undefined });
			// Ignore answers to queries the user has typed past
			if (q === query.trim()) discovered = result;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load spaces.');
		} finally {
			searching = false;
		}
	}

	async function joinDiscovered(space: DiscoverableSpaceDto) {
		joiningId = space.id;
		error = '';
		try {
			if (!space.role) await new SpaceApi(apiConfig()).joinSpace({ spaceId: space.id });
			await ondone(space.id);
			open = false;
		} catch (err) {
			error = await errorMessage(err, 'Failed to join space.');
		} finally {
			joiningId = null;
		}
	}

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

<Modal bind:open size={step === 'discover' ? 'md' : 'sm'} autoclose={false} title={TITLES[step]}>
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
		<button
			type="button"
			class="flex w-full items-center gap-3 rounded-lg border border-gray-200 p-4 text-left hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-700"
			onclick={() => (step = 'discover')}
		>
			<span class="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500 text-white">
				<GlobeOutline class="h-5 w-5" />
			</span>
			<span>
				<span class="block font-semibold text-gray-900 dark:text-white">Discover spaces</span>
				<span class="block text-sm text-gray-500 dark:text-gray-400">Browse public spaces and join one</span>
			</span>
		</button>
	{:else if step === 'discover'}
		<div class="flex flex-col gap-4">
			<Input type="search" bind:value={query} placeholder="Search by name or description" aria-label="Search spaces" />
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			{#if discovered.length === 0}
				<p class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
					{#if searching}
						Searching...
					{:else if query.trim()}
						No public space matches “{query.trim()}”.
					{:else}
						There are no public spaces yet. Owners and admins can make a space public in its settings.
					{/if}
				</p>
			{:else}
				<ul class="-mx-2 max-h-[55vh] overflow-y-auto">
					{#each discovered as space (space.id)}
						<li class="flex items-center gap-3 rounded-lg px-2 py-3 hover:bg-gray-50 dark:hover:bg-gray-700/50">
							<Avatar name={space.name} imageId={space.iconId} class="h-12 w-12 rounded-2xl text-sm" />
							<div class="min-w-0 flex-1">
								<p class="truncate font-semibold text-gray-900 dark:text-white">{space.name}</p>
								{#if space.description}
									<p class="line-clamp-2 text-sm text-gray-500 dark:text-gray-400">{space.description}</p>
								{/if}
								<p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
									{space.memberCount}
									{space.memberCount === 1 ? 'member' : 'members'} ·
									{space.role ? 'you are a member' : JOIN_ROLE_LABELS[space.joinRole]}
								</p>
							</div>
							<Button
								size="sm"
								color={space.role ? 'light' : 'primary'}
								disabled={joiningId !== null}
								onclick={() => joinDiscovered(space)}
							>
								{#if joiningId === space.id}
									Joining...
								{:else}
									{space.role ? 'Open' : 'Join'}
								{/if}
							</Button>
						</li>
					{/each}
				</ul>
			{/if}
			{#if start !== 'discover'}
				<div>
					<Button color="light" onclick={() => ((step = 'choose'), (error = ''))}>
						<ArrowLeftOutline class="me-1 h-4 w-4" /> Back
					</Button>
				</div>
			{/if}
		</div>
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
