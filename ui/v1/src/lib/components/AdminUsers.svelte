<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AdminUserDto } from 'ragbag-frontend-sdk';
	import { Button, Input } from 'flowbite-svelte';
	import { DotsHorizontalOutline } from 'flowbite-svelte-icons';
	import { apiConfig, appBase, errorMessage } from '../api/client';
	import { copyText } from '../utils/clipboard';
	import { session } from '../stores/spaces.svelte';
	import Avatar from './Avatar.svelte';
	import { shownName } from '../utils/format';

	/** The accounts of the instance: reset links, blocking and deleting. `focus` opens one account's block form. */
	let { focus = null }: { focus?: string | null } = $props();

	const api = new AdminApi(apiConfig());

	let users = $state<AdminUserDto[]>([]);
	let filter = $state('');
	let loading = $state(false);
	let error = $state('');
	/** Reset link created here; only shown once, the server keeps a hash. */
	let created = $state<{ userId: string; link: string; expiresAt: Date } | null>(null);
	let copied = $state(false);
	let busyUserId = $state<string | null>(null);
	/** Blocking or deleting asks first, in the user's row. */
	let pending = $state<{ userId: string; kind: 'block' | 'delete' } | null>(null);
	let reason = $state('');
	/** The account whose "more" menu is open: everything but the reset link, destructive actions set apart. */
	let menuUserId = $state<string | null>(null);
	const menuItem = 'flex w-full rounded-md px-2 py-1.5 text-left hover:bg-primary-500 hover:text-white';
	const dangerItem = 'flex w-full rounded-md px-2 py-1.5 text-left text-red-400 hover:bg-red-600 hover:text-white';

	function hasMore(user: AdminUserDto): boolean {
		return (
			!!user.twoFactor || user.plan === 'FREE' || user.proSource === 'GRANTED' ||
			(!user.instanceAdmin && user.id !== session.me?.id)
		);
	}

	const visible = $derived(
		users.filter((user) =>
			`${user.username} ${user.displayName ?? ''} ${user.email}`.toLowerCase().includes(filter.trim().toLowerCase()),
		),
	);

	load();

	async function load() {
		loading = true;
		error = '';
		created = null;
		pending = null;
		filter = '';
		try {
			users = await api.getAllUsers();
			if (focus) ask(focus, 'block');
		} catch (err) {
			error = await errorMessage(err, 'Failed to load users.');
		} finally {
			loading = false;
		}
	}

	/** For somebody who lost their authenticator app and recovery codes: they sign in with the password alone again. */
	async function resetTwoFactor(user: AdminUserDto) {
		busyUserId = user.id;
		error = '';
		try {
			await api.resetTwoFactor({ userId: user.id });
			users = users.map((u) => (u.id === user.id ? { ...u, twoFactor: false } : u));
		} catch (err) {
			error = await errorMessage(err, 'Failed to reset the two-factor authentication.');
		} finally {
			busyUserId = null;
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

	/** Free or Pro (PLANS_ENABLED); admins are always Pro. */
	async function setPro(user: AdminUserDto, granted: boolean) {
		busyUserId = user.id;
		error = '';
		try {
			const updated = granted ? await api.grantPro({ userId: user.id }) : await api.revokePro({ userId: user.id });
			users = users.map((u) => (u.id === updated.id ? updated : u));
		} catch (err) {
			error = await errorMessage(err, 'Failed to change the plan.');
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

	async function copy() {
		if (created && (await copyText(created.link))) {
			copied = true;
			setTimeout(() => (copied = false), 2000);
		}
	}
</script>

<svelte:window onclick={() => (menuUserId = null)} />

<div class="flex flex-col gap-4">
	<p class="text-sm text-gray-500 dark:text-gray-400">
		Without email, forgotten passwords are reset with a link you send the person yourself. A link works once, for
		24 hours, and replaces older links of that user.
	</p>
	<Input type="search" placeholder="Filter by name or email" bind:value={filter} />
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
	<ul class="-mx-2 divide-y divide-gray-100 dark:divide-gray-800">
		{#each visible as user (user.id)}
			<li class="px-2 py-2.5">
				<div class="flex items-center gap-3">
					<Avatar name={shownName(user)} imageId={user.avatarId} class="h-9 w-9 rounded-full text-sm" />
					<div class="min-w-0 flex-1">
						<p class="truncate font-medium text-gray-900 dark:text-white">
							{shownName(user)}{#if user.displayName}<span class="ms-1 text-xs font-normal text-gray-500 dark:text-gray-400">@{user.username}</span>{/if}
							{#if user.id === session.me?.id}<span class="font-normal text-gray-500">(you)</span>{/if}
							{#if user.instanceAdmin}
								<span
									class="ms-1 rounded-md bg-primary-500/15 px-1.5 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300"
									>Admin</span
								>
							{/if}
							{#if user.plan === 'PRO' && !user.instanceAdmin}
								<span
									class="ms-1 rounded-md bg-amber-500/15 px-1.5 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-300"
									>Pro</span
								>
							{/if}
							{#if user.twoFactor}
								<span
									class="ms-1 rounded-md bg-green-500/15 px-1.5 py-0.5 text-xs font-medium text-green-700 dark:text-green-300"
									title="Signs in with an authenticator app as well">2FA</span
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
					<div class="flex shrink-0 items-center gap-1">
						<Button size="xs" color="light" disabled={busyUserId === user.id} onclick={() => createLink(user)}>
							Reset link
						</Button>
						{#if hasMore(user)}
							<div class="relative">
								<Button
									size="xs"
									color="light"
									aria-label="More actions for {user.username}"
									aria-haspopup="menu"
									aria-expanded={menuUserId === user.id}
									disabled={busyUserId === user.id}
									onclick={(e: MouseEvent) => {
										e.stopPropagation();
										menuUserId = menuUserId === user.id ? null : user.id;
									}}
								>
									<DotsHorizontalOutline class="h-4 w-4" />
								</Button>
								{#if menuUserId === user.id}
									<div
										role="menu"
										class="absolute right-0 top-[calc(100%+4px)] z-30 w-48 rounded-lg bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
									>
										{#if user.twoFactor}
											<button
												role="menuitem"
												class={menuItem}
												title="Lost app and recovery codes: the account signs in with its password alone again"
												onclick={() => resetTwoFactor(user)}>Reset two-factor</button
											>
										{/if}
										{#if user.plan === 'FREE'}
											<button role="menuitem" class={menuItem} onclick={() => setPro(user, true)}>Make Pro</button>
										{:else if user.proSource === 'GRANTED'}
											<button role="menuitem" class={menuItem} onclick={() => setPro(user, false)}>Remove Pro</button>
										{/if}
										{#if !user.instanceAdmin && user.id !== session.me?.id}
											{#if user.twoFactor || user.plan === 'FREE' || user.proSource === 'GRANTED'}
												<div class="my-1 h-px bg-gray-800"></div>
											{/if}
											{#if user.blockedAt}
												<button role="menuitem" class={menuItem} onclick={() => unblock(user)}>Unblock</button>
											{:else}
												<button role="menuitem" class={dangerItem} onclick={() => ask(user.id, 'block')}>Block…</button>
											{/if}
											<button role="menuitem" class={dangerItem} onclick={() => ask(user.id, 'delete')}>Delete…</button>
										{/if}
									</div>
								{/if}
							</div>
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
</div>
