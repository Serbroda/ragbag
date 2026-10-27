<script lang="ts">
	import { InviteApi } from 'ragbag-frontend-sdk';
	import type { InviteRoleEnum, SpaceMemberRoleEnum } from 'ragbag-frontend-sdk';
	import { untrack } from 'svelte';
	import { Button, Input, Label, Modal, Select } from 'flowbite-svelte';
	import { apiConfig, appBase, errorMessage } from '../api/client';
	import { copyText } from '../utils/clipboard';

	let {
		open = $bindable(false),
		spaceId,
		spaceName,
		myRole,
	}: { open?: boolean; spaceId: string; spaceName: string; myRole: SpaceMemberRoleEnum } = $props();

	const ROLES: InviteRoleEnum[] = ['ADMIN', 'CONTRIBUTOR', 'VIEWER'];
	const RANK: Record<SpaceMemberRoleEnum, number> = { OWNER: 0, ADMIN: 1, CONTRIBUTOR: 2, VIEWER: 3 };
	const LABELS: Record<InviteRoleEnum, string> = { ADMIN: 'Admin', CONTRIBUTOR: 'Contributor', VIEWER: 'Viewer' };
	const roles = $derived(ROLES.filter((role) => RANK[role] >= RANK[myRole]));

	let role = $state<InviteRoleEnum>('VIEWER');
	let days = $state('7');
	let link = $state('');
	let error = $state('');
	let busy = $state(false);
	let copied = $state(false);
	let copyFailed = $state(false);
	let linkInput = $state<HTMLInputElement>();

	$effect(() => {
		if (!open) return;
		// Only opening the dialog may trigger this; generate() reads role and days
		untrack(() => {
			role = 'VIEWER';
			days = '7';
			copied = false;
			copyFailed = false;
			generate();
		});
	});

	async function generate() {
		busy = true;
		error = '';
		link = '';
		try {
			const expiryDays = Number(days);
			const invite = await new InviteApi(apiConfig()).createInvite({
				createInviteRequest: {
					targetType: 'SPACE',
					targetId: spaceId,
					role,
					expiresAt: expiryDays > 0 ? new Date(Date.now() + expiryDays * 24 * 60 * 60 * 1000) : undefined,
				},
			});
			link = `${window.location.origin}${appBase()}/invite/${invite.token}`;
		} catch (err) {
			error = await errorMessage(err, 'Failed to create invite link.');
		} finally {
			busy = false;
		}
	}

	async function copy() {
		copyFailed = !(await copyText(link));
		if (copyFailed) {
			// Leave the link selected so Ctrl/Cmd+C copies it
			linkInput?.focus();
			linkInput?.select();
			return;
		}
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<Modal bind:open size="sm" title="Invite people to {spaceName}">
	<div class="grid grid-cols-2 gap-3">
		<div>
			<Label for="quick-invite-role" class="mb-2">Role</Label>
			<Select id="quick-invite-role" placeholder="" bind:value={role} onchange={generate}>
				{#each roles as r (r)}
					<option value={r}>{LABELS[r]}</option>
				{/each}
			</Select>
		</div>
		<div>
			<Label for="quick-invite-expiry" class="mb-2">Expires</Label>
			<Select id="quick-invite-expiry" placeholder="" bind:value={days} onchange={generate}>
				<option value="1">after 1 day</option>
				<option value="7">after 7 days</option>
				<option value="30">after 30 days</option>
				<option value="0">never</option>
			</Select>
		</div>
	</div>
	<div>
		<Label for="quick-invite-link" class="mb-2">Send this link to your friends</Label>
		<div class="flex gap-2">
			<Input
				id="quick-invite-link"
				bind:elementRef={linkInput}
				readonly
				value={busy ? 'Generating…' : link}
				class="font-mono text-xs"
				onfocus={(e: FocusEvent) => (e.currentTarget as HTMLInputElement).select()}
			/>
			<Button color={copied ? 'green' : 'primary'} disabled={!link} onclick={copy}>
				{copied ? 'Copied' : 'Copy'}
			</Button>
		</div>
		{#if copyFailed}
			<p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
				Your browser blocked copying. The link is selected, press Ctrl+C (⌘C on Mac).
			</p>
		{/if}
	</div>
	{#if error}
		<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
	{/if}
</Modal>
