<script lang="ts">
	import { InviteApi, SpaceApi, UserApi } from 'ragbag-frontend-sdk';
	import type {
		InviteDto,
		InviteRoleEnum,
		JoinRoleEnum,
		SpaceDto,
		SpaceMemberDto,
		SpaceMemberRoleEnum,
	} from 'ragbag-frontend-sdk';
	import { getContext } from 'svelte';
	import { Alert, Button, Input, Label, Modal, Select, Textarea, Toggle } from 'flowbite-svelte';
	import { apiConfig, appBase, errorMessage, fetchFeatureFlags } from '../api/client';
	import { navigate, route } from '../router';
	import RoleBadge from '../components/RoleBadge.svelte';
	import Avatar from '../components/Avatar.svelte';
	import ImagePicker from '../components/ImagePicker.svelte';
	import { copyText } from '../utils/clipboard';

	const spaceApi = new SpaceApi(apiConfig());
	const inviteApi = new InviteApi(apiConfig());
	const userApi = new UserApi(apiConfig());

	const reloadSpaces = getContext<(() => Promise<void>) | undefined>('reload-spaces');

	/** Most privileged first, mirrors SpaceMemberRole on the server. */
	const ROLES: SpaceMemberRoleEnum[] = ['OWNER', 'ADMIN', 'CONTRIBUTOR', 'VIEWER'];
	const INVITE_ROLES: InviteRoleEnum[] = ['ADMIN', 'CONTRIBUTOR', 'VIEWER'];
	const ROLE_LABELS: Record<SpaceMemberRoleEnum, string> = {
		OWNER: 'Owner',
		ADMIN: 'Admin',
		CONTRIBUTOR: 'Contributor',
		VIEWER: 'Viewer',
	};
	const rank = (role: SpaceMemberRoleEnum) => ROLES.indexOf(role);

	const spaceId = $derived(route.params.spaceId ?? '');

	let space = $state<SpaceDto | null>(null);
	let members = $state<SpaceMemberDto[]>([]);
	let invites = $state<InviteDto[]>([]);
	let currentUserId = $state<string | null>(null);
	let loading = $state(true);
	let loadError = $state('');

	const permissions = $derived(new Set(space?.permissions ?? []));
	const myRole = $derived<SpaceMemberRoleEnum>(space?.role ?? 'VIEWER');
	const canEditSpace = $derived(permissions.has('EDIT_SPACE'));
	const canManageMembers = $derived(permissions.has('MANAGE_MEMBERS'));
	const canDeleteSpace = $derived(permissions.has('DELETE_SPACE'));
	const assignableRoles = $derived(ROLES.filter((role) => rank(role) >= rank(myRole)));
	const assignableInviteRoles = $derived(INVITE_ROLES.filter((role) => rank(role) >= rank(myRole)));

	/** Owners manage everyone else; others only members ranked strictly below them. */
	function canManage(member: SpaceMemberDto): boolean {
		if (!canManageMembers || member.userId === currentUserId) return false;
		return myRole === 'OWNER' || rank(myRole) < rank(member.role);
	}

	$effect(() => {
		if (spaceId) load(spaceId);
	});

	async function load(id: string) {
		loading = true;
		loadError = '';
		try {
			const [loadedSpace, loadedMembers, me] = await Promise.all([
				spaceApi.getSpace({ spaceId: id }),
				spaceApi.getSpaceMembers({ spaceId: id }),
				userApi.me(),
			]);
			space = loadedSpace;
			members = loadedMembers;
			currentUserId = me.id;
			name = loadedSpace.name;
			description = loadedSpace.description ?? '';
			discoverable = loadedSpace.discoverable ?? false;
			joinRole = loadedSpace.joinRole ?? 'VIEWER';
			invites = loadedSpace.permissions?.includes('MANAGE_MEMBERS')
				? await inviteApi.getSpaceInvites({ spaceId: id })
				: [];
		} catch (err) {
			loadError = await errorMessage(err, 'Failed to load space settings.');
		} finally {
			loading = false;
		}
	}

	// --- General ---
	let name = $state('');
	let description = $state('');
	let savingGeneral = $state(false);
	let generalMessage = $state<{ color: 'green' | 'red'; text: string } | null>(null);

	async function saveGeneral(e: Event) {
		e.preventDefault();
		savingGeneral = true;
		generalMessage = null;
		try {
			space = await spaceApi.updateSpace({
				spaceId,
				updateSpaceDto: { name: name.trim(), description: description.trim() || undefined },
			});
			await reloadSpaces?.();
			generalMessage = { color: 'green', text: 'Saved.' };
		} catch (err) {
			generalMessage = { color: 'red', text: await errorMessage(err, 'Failed to save.') };
		} finally {
			savingGeneral = false;
		}
	}

	// --- Discovery ---
	let discoverable = $state(false);
	let joinRole = $state<JoinRoleEnum>('VIEWER');
	let savingDiscovery = $state(false);
	let discoveryMessage = $state<{ color: 'green' | 'red'; text: string } | null>(null);
	/** With open registration, anybody who can reach the instance can create an account and join. */
	let registrationOpen = $state(false);
	const discoveryChanged = $derived(
		discoverable !== (space?.discoverable ?? false) || joinRole !== (space?.joinRole ?? 'VIEWER'),
	);

	$effect(() => {
		fetchFeatureFlags()
			.then((flags) => (registrationOpen = flags['registration_enabled'] === 'true'))
			.catch(() => (registrationOpen = false));
	});

	async function saveDiscovery(e: Event) {
		e.preventDefault();
		if (!space) return;
		savingDiscovery = true;
		discoveryMessage = null;
		try {
			space = await spaceApi.updateSpace({
				spaceId,
				updateSpaceDto: { name: space.name, description: space.description, discoverable, joinRole },
			});
			discoveryMessage = { color: 'green', text: discoverable ? 'Everybody can find this space now.' : 'Saved.' };
		} catch (err) {
			discoveryMessage = { color: 'red', text: await errorMessage(err, 'Failed to save.') };
		} finally {
			savingDiscovery = false;
		}
	}

	// --- Members ---
	let memberError = $state('');
	let memberToRemove = $state<SpaceMemberDto | null>(null);

	async function changeRole(member: SpaceMemberDto, role: SpaceMemberRoleEnum) {
		memberError = '';
		try {
			const updated = await spaceApi.updateSpaceMember({
				spaceId,
				userId: member.userId,
				updateSpaceMemberDto: { role },
			});
			members = members.map((m) => (m.userId === updated.userId ? updated : m));
		} catch (err) {
			memberError = await errorMessage(err, 'Failed to change role.');
			// Re-sync the select with the server state
			members = await spaceApi.getSpaceMembers({ spaceId });
		}
	}

	async function removeMember() {
		if (!memberToRemove) return;
		memberError = '';
		try {
			await spaceApi.removeSpaceMember({ spaceId, userId: memberToRemove.userId });
			members = members.filter((m) => m.userId !== memberToRemove?.userId);
		} catch (err) {
			memberError = await errorMessage(err, 'Failed to remove member.');
		} finally {
			memberToRemove = null;
		}
	}

	// --- Invites ---
	let inviteRole = $state<InviteRoleEnum>('VIEWER');
	let inviteExpiryDays = $state('7');
	let inviteMaxUses = $state('');
	let creatingInvite = $state(false);
	let inviteError = $state('');
	let copiedToken = $state<string | null>(null);
	/** Shown for manual copying when the browser refuses clipboard access. */
	let manualCopyLink = $state<string | null>(null);

	function inviteLink(token: string): string {
		return `${window.location.origin}${appBase()}/invite/${token}`;
	}

	function isActive(invite: InviteDto): boolean {
		const expired = invite.expiresAt != null && invite.expiresAt.getTime() < Date.now();
		const usedUp = invite.maxUses != null && (invite.usedCount ?? 0) >= invite.maxUses;
		return !expired && !usedUp;
	}

	async function createInvite(e: Event) {
		e.preventDefault();
		creatingInvite = true;
		inviteError = '';
		try {
			const days = Number(inviteExpiryDays);
			const invite = await inviteApi.createInvite({
				createInviteRequest: {
					targetType: 'SPACE',
					targetId: spaceId,
					role: inviteRole,
					expiresAt: days > 0 ? new Date(Date.now() + days * 24 * 60 * 60 * 1000) : undefined,
					maxUses: inviteMaxUses ? Number(inviteMaxUses) : undefined,
				},
			});
			invites = [invite, ...invites];
			await copyLink(invite.token!);
		} catch (err) {
			inviteError = await errorMessage(err, 'Failed to create invite.');
		} finally {
			creatingInvite = false;
		}
	}

	async function copyLink(token: string) {
		if (!(await copyText(inviteLink(token)))) {
			manualCopyLink = inviteLink(token);
			return;
		}
		manualCopyLink = null;
		copiedToken = token;
		setTimeout(() => {
			if (copiedToken === token) copiedToken = null;
		}, 2000);
	}

	async function revokeInvite(token: string) {
		inviteError = '';
		try {
			await inviteApi.deleteInvite({ token });
			invites = invites.filter((i) => i.token !== token);
		} catch (err) {
			inviteError = await errorMessage(err, 'Failed to revoke invite.');
		}
	}

	// --- Danger zone ---
	let showLeaveModal = $state(false);
	let showDeleteModal = $state(false);
	let deleteConfirmation = $state('');
	let dangerError = $state('');
	let dangerBusy = $state(false);

	async function leaveSpace() {
		dangerBusy = true;
		dangerError = '';
		try {
			await spaceApi.leaveSpace({ spaceId });
			showLeaveModal = false;
			await reloadSpaces?.();
			navigate('/');
		} catch (err) {
			dangerError = await errorMessage(err, 'Failed to leave space.');
		} finally {
			dangerBusy = false;
		}
	}

	async function deleteSpace() {
		dangerBusy = true;
		dangerError = '';
		try {
			await spaceApi.deleteSpace({ spaceId });
			showDeleteModal = false;
			await reloadSpaces?.();
			navigate('/');
		} catch (err) {
			dangerError = await errorMessage(err, 'Failed to delete space.');
		} finally {
			dangerBusy = false;
		}
	}

	function formatDate(date: Date | undefined | null): string {
		return date ? date.toLocaleDateString(undefined, { dateStyle: 'medium' }) : 'Never';
	}
</script>

<div class="mx-auto max-w-3xl space-y-6">
	<h1 class="text-2xl font-bold tracking-tight">Space settings</h1>

	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if loadError}
		<Alert color="red">{loadError}</Alert>
	{:else if space}
		<!-- General -->
		<section class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
			<h2 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">General</h2>
			<div class="mb-4">
				<Label class="mb-3">Icon</Label>
				{#if canEditSpace}
					<ImagePicker
						name={space.name}
						imageId={space.iconId}
						shape="rounded-2xl"
						onupload={async (file) => {
							space = await spaceApi.setSpaceIcon({ spaceId, file });
							await reloadSpaces?.();
						}}
						onurl={async (url) => {
							space = await spaceApi.setSpaceIcon({ spaceId, url });
							await reloadSpaces?.();
						}}
						onremove={async () => {
							space = await spaceApi.removeSpaceIcon({ spaceId });
							await reloadSpaces?.();
						}}
					/>
				{:else}
					<Avatar name={space.name} imageId={space.iconId} class="h-20 w-20 rounded-2xl text-2xl" />
				{/if}
			</div>
			<form class="flex flex-col gap-4" onsubmit={saveGeneral}>
				<div>
					<Label for="space-name" class="mb-2">Name</Label>
					<Input id="space-name" bind:value={name} required maxlength={80} disabled={!canEditSpace} />
				</div>
				<div>
					<Label for="space-description" class="mb-2">Description</Label>
					<Textarea
						id="space-description"
						bind:value={description}
						rows={3}
						class="w-full"
						disabled={!canEditSpace}
					/>
				</div>
				{#if generalMessage}
					<Alert color={generalMessage.color}>{generalMessage.text}</Alert>
				{/if}
				{#if canEditSpace}
					<div class="flex justify-end">
						<Button type="submit" disabled={savingGeneral || !name.trim()}>
							{savingGeneral ? 'Saving...' : 'Save'}
						</Button>
					</div>
				{/if}
			</form>
		</section>

		<!-- Members -->
		<section class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
			<div class="mb-4 flex items-center justify-between">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Members</h2>
				<span class="text-sm text-gray-500 dark:text-gray-400">{members.length}</span>
			</div>
			{#if memberError}
				<Alert color="red" class="mb-4">{memberError}</Alert>
			{/if}
			<ul class="divide-y divide-gray-100 dark:divide-gray-800">
				{#each members as member (member.userId)}
					<li class="flex items-center gap-3 py-3">
						<Avatar name={member.username} imageId={member.avatarId} class="h-9 w-9 rounded-full text-sm" />
						<div class="min-w-0 flex-1">
							<p class="truncate font-medium text-gray-900 dark:text-white">
								{member.username}
								{#if member.userId === currentUserId}
									<span class="text-sm font-normal text-gray-500 dark:text-gray-400">(you)</span>
								{/if}
							</p>
						</div>
						{#if canManage(member)}
							<Select placeholder=""
								size="sm"
								class="w-36"
								value={member.role}
								aria-label="Role of {member.username}"
								onchange={(e: Event) =>
									changeRole(member, (e.currentTarget as HTMLSelectElement).value as SpaceMemberRoleEnum)}
							>
								{#each assignableRoles as role (role)}
									<option value={role}>{ROLE_LABELS[role]}</option>
								{/each}
							</Select>
							<Button size="xs" color="light" onclick={() => (memberToRemove = member)}>Remove</Button>
						{:else}
							<RoleBadge role={member.role} />
						{/if}
					</li>
				{/each}
			</ul>
		</section>

		<!-- Discovery -->
		{#if canManageMembers}
			<section class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
				<h2 class="mb-1 text-lg font-semibold text-gray-900 dark:text-white">Discovery</h2>
				<p class="mb-4 text-sm text-gray-500 dark:text-gray-400">
					Who can join this space. Private collections stay private either way.
				</p>
				<form class="flex flex-col gap-4" onsubmit={saveDiscovery}>
					<Toggle bind:checked={discoverable}>Public – listed under “Discover spaces”</Toggle>
					<p class="-mt-2 text-sm text-gray-500 dark:text-gray-400">
						{#if discoverable}
							Everybody with an account on this Ragbag can find the space and join it without an invite.
						{:else}
							Only people with an invite link can join.
						{/if}
					</p>
					{#if discoverable}
						<div class="sm:w-64">
							<Label for="join-role" class="mb-2">People who join become</Label>
							<Select id="join-role" placeholder="" bind:value={joinRole}>
								<option value="VIEWER">Viewers – they can read</option>
								<option value="CONTRIBUTOR">Contributors – they can add links</option>
							</Select>
						</div>
						{#if registrationOpen}
							<Alert color="yellow">
								Registration is open: if this Ragbag is reachable from the internet, anybody can create an
								account and join. Set <code>REGISTRATION_ENABLED=false</code> to keep it to your people.
							</Alert>
						{/if}
					{/if}
					{#if discoveryMessage}
						<Alert color={discoveryMessage.color}>{discoveryMessage.text}</Alert>
					{/if}
					<div class="flex justify-end">
						<Button type="submit" disabled={savingDiscovery || !discoveryChanged}>
							{savingDiscovery ? 'Saving...' : 'Save'}
						</Button>
					</div>
				</form>
			</section>
		{/if}

		<!-- Invites -->
		{#if canManageMembers}
			<section class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
				<h2 class="mb-1 text-lg font-semibold text-gray-900 dark:text-white">Invite links</h2>
				<p class="mb-4 text-sm text-gray-500 dark:text-gray-400">
					Anyone with the link can join this space with the selected role.
				</p>
				<form class="grid grid-cols-1 items-end gap-3 sm:grid-cols-4" onsubmit={createInvite}>
					<div>
						<Label for="invite-role" class="mb-2">Role</Label>
						<Select id="invite-role" placeholder="" bind:value={inviteRole}>
							{#each assignableInviteRoles as role (role)}
								<option value={role}>{ROLE_LABELS[role]}</option>
							{/each}
						</Select>
					</div>
					<div>
						<Label for="invite-expiry" class="mb-2">Expires</Label>
						<Select id="invite-expiry" placeholder="" bind:value={inviteExpiryDays}>
							<option value="1">after 1 day</option>
							<option value="7">after 7 days</option>
							<option value="30">after 30 days</option>
							<option value="0">never</option>
						</Select>
					</div>
					<div>
						<Label for="invite-uses" class="mb-2">Max uses</Label>
						<Select id="invite-uses" placeholder="" bind:value={inviteMaxUses}>
							<option value="">unlimited</option>
							<option value="1">1</option>
							<option value="5">5</option>
							<option value="10">10</option>
							<option value="25">25</option>
						</Select>
					</div>
					<Button type="submit" disabled={creatingInvite}>
						{creatingInvite ? 'Creating...' : 'Create link'}
					</Button>
				</form>
				{#if inviteError}
					<Alert color="red" class="mt-4">{inviteError}</Alert>
				{/if}
				{#if manualCopyLink}
					<div class="mt-4 rounded-lg bg-amber-500/10 p-3 ring-1 ring-inset ring-amber-500/30">
						<p class="mb-2 text-sm text-amber-700 dark:text-amber-300">
							Your browser blocked copying. Copy the link with Ctrl+C (⌘C on Mac):
						</p>
						<input
							readonly
							value={manualCopyLink}
							class="w-full rounded-lg border border-gray-300 bg-gray-50 px-3 py-2 font-mono text-xs text-gray-900 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							onfocus={(e) => e.currentTarget.select()}
							{@attach (node) => {
								node.focus();
								node.select();
							}}
						/>
					</div>
				{/if}

				{#if invites.length > 0}
					<ul class="mt-4 divide-y divide-gray-100 dark:divide-gray-800">
						{#each invites as invite (invite.token)}
							{@const active = isActive(invite)}
							<li class="flex flex-wrap items-center gap-3 py-3" class:opacity-60={!active}>
								{#if invite.role}
									<RoleBadge role={invite.role} />
								{/if}
								<div class="min-w-0 flex-1">
									<p
										class="truncate font-mono text-xs text-gray-700 dark:text-gray-300"
										title={inviteLink(invite.token!)}
									>
										…/invite/{invite.token!.slice(0, 10)}…
									</p>
									<p class="text-xs text-gray-500 dark:text-gray-400">
										by {invite.createdBy} · used {invite.usedCount ?? 0}{invite.maxUses
											? ` / ${invite.maxUses}`
											: ''} · expires {formatDate(invite.expiresAt)}
										{#if !active}· <span class="font-medium">inactive</span>{/if}
									</p>
								</div>
								{#if active}
									<Button size="xs" color="light" onclick={() => copyLink(invite.token!)}>
										{copiedToken === invite.token ? 'Copied!' : 'Copy'}
									</Button>
								{/if}
								<Button size="xs" color="red" outline onclick={() => revokeInvite(invite.token!)}>
									Revoke
								</Button>
							</li>
						{/each}
					</ul>
				{/if}
			</section>
		{/if}

		<!-- Danger zone -->
		<section class="rounded-xl border border-red-500/30 bg-white p-6 dark:bg-gray-900">
			<h2 class="mb-4 text-lg font-semibold text-red-600 dark:text-red-500">Danger zone</h2>
			{#if dangerError && !showLeaveModal && !showDeleteModal}
				<Alert color="red" class="mb-4">{dangerError}</Alert>
			{/if}
			<div class="flex flex-col gap-4">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div>
						<p class="font-medium text-gray-900 dark:text-white">Leave space</p>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							You lose access until someone invites you again.
						</p>
					</div>
					<Button color="light" onclick={() => ((dangerError = ''), (showLeaveModal = true))}>Leave</Button>
				</div>
				{#if canDeleteSpace}
					<div class="flex flex-wrap items-center justify-between gap-2">
						<div>
							<p class="font-medium text-gray-900 dark:text-white">Delete space</p>
							<p class="text-sm text-gray-500 dark:text-gray-400">
								Permanently deletes all collections, bookmarks and invites.
							</p>
						</div>
						<Button
							color="red"
							onclick={() => ((dangerError = ''), (deleteConfirmation = ''), (showDeleteModal = true))}
						>
							Delete
						</Button>
					</div>
				{/if}
			</div>
		</section>
	{/if}
</div>

<Modal title="Remove member" open={memberToRemove !== null} onclose={() => (memberToRemove = null)} size="sm">
	<p class="text-gray-700 dark:text-gray-300">
		Remove <strong>{memberToRemove?.username}</strong> from this space?
	</p>
	<div class="flex justify-end gap-2">
		<Button color="light" onclick={() => (memberToRemove = null)}>Cancel</Button>
		<Button color="red" onclick={removeMember}>Remove</Button>
	</div>
</Modal>

<Modal title="Leave space" bind:open={showLeaveModal} size="sm" autoclose={false}>
	<p class="text-gray-700 dark:text-gray-300">Leave <strong>{space?.name}</strong>?</p>
	{#if dangerError}
		<Alert color="red">{dangerError}</Alert>
	{/if}
	<div class="flex justify-end gap-2">
		<Button color="light" onclick={() => (showLeaveModal = false)}>Cancel</Button>
		<Button color="red" disabled={dangerBusy} onclick={leaveSpace}>Leave</Button>
	</div>
</Modal>

<Modal title="Delete space" bind:open={showDeleteModal} size="sm" autoclose={false}>
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			deleteSpace();
		}}
	>
		<p class="text-gray-700 dark:text-gray-300">
			This cannot be undone. Type <strong>{space?.name}</strong> to confirm.
		</p>
		<Input bind:value={deleteConfirmation} aria-label="Space name" />
		{#if dangerError}
			<Alert color="red">{dangerError}</Alert>
		{/if}
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (showDeleteModal = false)}>Cancel</Button>
			<Button type="submit" color="red" disabled={dangerBusy || deleteConfirmation !== space?.name}>
				Delete space
			</Button>
		</div>
	</form>
</Modal>
