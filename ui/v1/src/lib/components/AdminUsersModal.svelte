<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AdminUserDto } from 'ragbag-frontend-sdk';
	import { untrack } from 'svelte';
	import { Button, Input, Modal } from 'flowbite-svelte';
	import { apiConfig, appBase, errorMessage } from '../api/client';
	import { copyText } from '../utils/clipboard';
	import { session } from '../stores/spaces.svelte';
	import Avatar from './Avatar.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const api = new AdminApi(apiConfig());

	let users = $state<AdminUserDto[]>([]);
	let filter = $state('');
	let loading = $state(false);
	let error = $state('');
	/** Reset link created in this dialog; only shown once, the server keeps a hash. */
	let created = $state<{ userId: string; link: string; expiresAt: Date } | null>(null);
	let copied = $state(false);
	let busyUserId = $state<string | null>(null);

	const visible = $derived(
		users.filter((user) =>
			`${user.username} ${user.email}`.toLowerCase().includes(filter.trim().toLowerCase()),
		),
	);

	$effect(() => {
		if (!open) return;
		untrack(load);
	});

	async function load() {
		loading = true;
		error = '';
		created = null;
		filter = '';
		try {
			users = await api.getAllUsers();
		} catch (err) {
			error = await errorMessage(err, 'Failed to load users.');
		} finally {
			loading = false;
		}
	}

	async function createLink(user: AdminUserDto) {
		busyUserId = user.id;
		error = '';
		copied = false;
		try {
			const reset = await api.createPasswordReset({ userId: user.id });
			created = {
				userId: user.id,
				link: `${window.location.origin}${appBase()}/reset-password/${reset.token}`,
				expiresAt: reset.expiresAt,
			};
		} catch (err) {
			error = await errorMessage(err, 'Failed to create a reset link.');
		} finally {
			busyUserId = null;
		}
	}

	async function copy() {
		if (created && (await copyText(created.link))) {
			copied = true;
			setTimeout(() => (copied = false), 2000);
		}
	}
</script>

<Modal bind:open size="lg" title="Users of this Ragbag">
	<p class="text-sm text-gray-500 dark:text-gray-400">
		Without email, forgotten passwords are reset with a link you send the person yourself. A link works once, for
		24 hours, and replaces older links of that user.
	</p>
	<Input type="search" placeholder="Filter by name or email" bind:value={filter} />
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
	<ul class="-mx-2 max-h-[50vh] divide-y divide-gray-100 overflow-y-auto dark:divide-gray-800">
		{#each visible as user (user.id)}
			<li class="px-2 py-2.5">
				<div class="flex items-center gap-3">
					<Avatar name={user.username} imageId={user.avatarId} class="h-9 w-9 rounded-full text-sm" />
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium text-gray-900 dark:text-white">
							{user.username}
							{#if user.id === session.me?.id}<span class="font-normal text-gray-500">(you)</span>{/if}
							{#if user.instanceAdmin}
								<span
									class="ms-1 rounded-md bg-primary-500/15 px-1.5 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300"
									>Admin</span
								>
							{/if}
						</p>
						<p class="truncate text-xs text-gray-500">
							{user.email} · {user.spaces ?? 0} space{user.spaces === 1 ? '' : 's'}
							{#if user.createdAt}· joined {user.createdAt.toLocaleDateString()}{/if}
						</p>
					</div>
					<Button size="xs" color="light" disabled={busyUserId === user.id} onclick={() => createLink(user)}>
						{busyUserId === user.id ? 'Creating…' : 'Reset link'}
					</Button>
				</div>
				{#if created?.userId === user.id}
					<div class="mt-2 rounded-lg bg-amber-500/10 p-3 ring-1 ring-inset ring-amber-500/30">
						<p class="mb-2 text-xs text-amber-700 dark:text-amber-300">
							Send this link to {user.username}. It is shown only now and expires
							{created.expiresAt.toLocaleString()}.
						</p>
						<div class="flex gap-2">
							<Input
								readonly
								value={created.link}
								class="font-mono text-xs"
								onfocus={(e: FocusEvent) => (e.currentTarget as HTMLInputElement).select()}
							/>
							<Button size="sm" color={copied ? 'green' : 'primary'} onclick={copy}>
								{copied ? 'Copied' : 'Copy'}
							</Button>
						</div>
					</div>
				{/if}
			</li>
		{:else}
			<li class="px-2 py-8 text-center text-sm text-gray-500">{loading ? 'Loading…' : 'No users found.'}</li>
		{/each}
	</ul>
</Modal>
