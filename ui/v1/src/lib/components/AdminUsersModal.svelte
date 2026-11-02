<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AdminUserDto } from 'ragbag-frontend-sdk';
	import { untrack } from 'svelte';
	import { Button, Input, Modal } from 'flowbite-svelte';
	import { apiConfig, appBase, downloadBackup, errorMessage } from '../api/client';
	import { copyText } from '../utils/clipboard';
	import { session } from '../stores/spaces.svelte';
	import Avatar from './Avatar.svelte';
	import AdminReports from './AdminReports.svelte';

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
	let downloading = $state(false);
	let tab = $state<'users' | 'reports'>('users');
	/** Blocking or deleting asks first, in the user's row. */
	let pending = $state<{ userId: string; kind: 'block' | 'delete' } | null>(null);
	let reason = $state('');

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
		pending = null;
		filter = '';
		tab = session.me?.openReports ? 'reports' : 'users';
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

	function ask(userId: string, kind: 'block' | 'delete') {
		pending = { userId, kind };
		reason = '';
		created = null;
		tab = 'users';
	}

	async function block(user: AdminUserDto) {
		busyUserId = user.id;
		error = '';
		try {
			const updated = await api.blockUser({ userId: user.id, blockUserRequest: { reason: reason.trim() || undefined } });
			users = users.map((u) => (u.id === updated.id ? updated : u));
			pending = null;
		} catch (err) {
			error = await errorMessage(err, 'Failed to block the account.');
		} finally {
			busyUserId = null;
		}
	}

	async function unblock(user: AdminUserDto) {
		busyUserId = user.id;
		error = '';
		try {
			const updated = await api.unblockUser({ userId: user.id });
			users = users.map((u) => (u.id === updated.id ? updated : u));
		} catch (err) {
			error = await errorMessage(err, 'Failed to unblock the account.');
		} finally {
			busyUserId = null;
		}
	}

	async function remove(user: AdminUserDto) {
		busyUserId = user.id;
		error = '';
		try {
			await api.deleteUser({ userId: user.id });
			users = users.filter((u) => u.id !== user.id);
			pending = null;
		} catch (err) {
			error = await errorMessage(err, 'Failed to delete the account.');
		} finally {
			busyUserId = null;
		}
	}

	async function backup() {
		downloading = true;
		error = '';
		try {
			await downloadBackup();
		} catch (err) {
			error = await errorMessage(err, 'Failed to create the backup.');
		} finally {
			downloading = false;
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
	<div class="flex gap-1 border-b border-gray-100 dark:border-gray-800" role="tablist">
		{#each [['users', 'Accounts'], ['reports', 'Reports']] as [value, label] (value)}
			<button
				type="button"
				role="tab"
				aria-selected={tab === value}
				class="-mb-px border-b-2 px-3 py-2 text-sm font-medium {tab === value
					? 'border-primary-500 text-gray-900 dark:text-white'
					: 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}"
				onclick={() => (tab = value as 'users' | 'reports')}
			>
				{label}
				{#if value === 'reports' && session.me?.openReports}
					<span class="ms-1 rounded-full bg-red-500 px-1.5 text-xs text-white">{session.me.openReports}</span>
				{/if}
			</button>
		{/each}
	</div>
	{#if tab === 'reports'}
		<AdminReports onblock={(userId) => ask(userId, 'block')} />
	{:else}
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
								{#if user.blockedAt}
									<span
										class="ms-1 rounded-md bg-red-500/15 px-1.5 py-0.5 text-xs font-medium text-red-700 dark:text-red-300"
										title={user.blockedReason ?? ''}>Blocked</span
									>
								{/if}
							</p>
							<p class="truncate text-xs text-gray-500">
								{user.email} · {user.spaces ?? 0} space{user.spaces === 1 ? '' : 's'}
								{#if user.createdAt}· joined {user.createdAt.toLocaleDateString()}{/if}
							</p>
						</div>
						<div class="flex shrink-0 flex-wrap justify-end gap-1">
							<Button size="xs" color="light" disabled={busyUserId === user.id} onclick={() => createLink(user)}>
								Reset link
							</Button>
							{#if !user.instanceAdmin && user.id !== session.me?.id}
								{#if user.blockedAt}
									<Button size="xs" color="light" disabled={busyUserId === user.id} onclick={() => unblock(user)}>
										Unblock
									</Button>
								{:else}
									<Button size="xs" color="light" onclick={() => ask(user.id, 'block')}>Block…</Button>
								{/if}
								<Button size="xs" color="light" onclick={() => ask(user.id, 'delete')}>Delete…</Button>
							{/if}
						</div>
					</div>
					{#if pending?.userId === user.id && pending.kind === 'block'}
						<div class="mt-2 flex flex-col gap-2 rounded-lg bg-red-500/5 p-3 ring-1 ring-inset ring-red-500/20">
							<p class="text-xs text-gray-600 dark:text-gray-300">
								{user.username} is signed out, cannot sign in anymore and their public links stop working. Their
								spaces stay for the other members. The reason is shown at the login{user.email ? ' and sent by email' : ''}.
							</p>
							<Input size="sm" placeholder="Reason (optional)" maxlength={1000} bind:value={reason} />
							<div class="flex justify-end gap-2">
								<Button size="xs" color="light" onclick={() => (pending = null)}>Cancel</Button>
								<Button size="xs" color="red" disabled={busyUserId === user.id} onclick={() => block(user)}>
									Block {user.username}
								</Button>
							</div>
						</div>
					{:else if pending?.userId === user.id && pending.kind === 'delete'}
						<div class="mt-2 flex flex-col gap-2 rounded-lg bg-red-500/5 p-3 ring-1 ring-inset ring-red-500/20">
							<p class="text-xs text-gray-600 dark:text-gray-300">
								Deletes {user.username}'s account and personal data for good. Spaces without other members are
								deleted with it; spaces they are the only owner of go to the member with the highest role. Their links
								in shared spaces stay without their name.
							</p>
							<div class="flex justify-end gap-2">
								<Button size="xs" color="light" onclick={() => (pending = null)}>Cancel</Button>
								<Button size="xs" color="red" disabled={busyUserId === user.id} onclick={() => remove(user)}>
									Delete {user.username}
								</Button>
							</div>
						</div>
					{/if}
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
		<div class="flex flex-wrap items-center gap-3 border-t border-gray-100 pt-4 dark:border-gray-800">
			<p class="min-w-0 flex-1 text-xs text-gray-500 dark:text-gray-400">
				A backup holds everything, password hashes included: keep it safe. Restore it on the first login of a new,
				empty Ragbag (SQLite or PostgreSQL) or with <code>ragbag restore</code>.
			</p>
			<Button size="sm" color="light" disabled={downloading} onclick={backup}>
				{downloading ? 'Preparing…' : 'Download backup'}
			</Button>
		</div>
	{/if}
</Modal>
