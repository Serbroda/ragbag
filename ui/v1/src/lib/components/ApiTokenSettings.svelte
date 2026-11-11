<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import type { ApiTokenCreatedDto, ApiTokenDto, ApiTokenScope } from 'ragbag-frontend-sdk';
	import { Button, Input, Label } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { p } from '../router';
	import { session } from '../stores/spaces.svelte';
	import { showToast } from '../stores/toast.svelte';
	import { copyText } from '../utils/clipboard';
	import { timeAgo } from '../utils/format';

	/** The profile dialog: closing it forgets everything typed or shown here. */
	let { open }: { open: boolean } = $props();

	const userApi = new UserApi(apiConfig());
	const EXPIRY = [
		{ days: 0, label: 'Never expires' },
		{ days: 30, label: '30 days' },
		{ days: 90, label: '90 days' },
		{ days: 365, label: '1 year' },
	];
	const selectClass =
		'rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white';

	let tokens = $state<ApiTokenDto[]>([]);
	let creating = $state(false);
	let name = $state('');
	let scope = $state<ApiTokenScope>('READ');
	let days = $state(0);
	let password = $state('');
	let busy = $state(false);
	let error = $state('');
	/** Shown once after creating: the server only keeps a hash. */
	let created = $state<ApiTokenCreatedDto | null>(null);

	/** With plans, API tokens are part of Pro: Free accounts only see (and revoke) what they have. */
	const allowed = $derived(session.me?.apiTokens ?? true);

	const example = $derived(`curl -H "Authorization: Bearer ${created?.token ?? ''}" ${window.location.origin}/api/v1/spaces`);

	$effect(() => {
		if (!open) {
			creating = false;
			name = password = error = '';
			created = null;
			return;
		}
		userApi.getApiTokens().then((loaded) => (tokens = loaded)).catch(() => {});
	});

	async function create(e: Event) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			created = await userApi.createApiToken({
				createApiTokenRequest: { name, scope, expiresInDays: days || undefined, currentPassword: password },
			});
			tokens = [created, ...tokens];
			creating = false;
			name = password = '';
		} catch (err) {
			error = await errorMessage(err, 'Failed to create the token.');
		} finally {
			busy = false;
		}
	}

	async function revoke(token: ApiTokenDto) {
		error = '';
		try {
			await userApi.deleteApiToken({ tokenId: token.id });
			tokens = tokens.filter((t) => t.id !== token.id);
			if (created?.id === token.id) created = null;
			showToast(`Token “${token.name}” revoked`);
		} catch (err) {
			error = await errorMessage(err, 'Failed to revoke the token.');
		}
	}

	async function copy(text: string, what: string) {
		showToast((await copyText(text)) ? `${what} copied` : 'Could not copy');
	}
</script>

<section>
	<Label class="mb-1">API tokens</Label>
	<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
		For scripts, shortcuts and apps that read or save links as you (<code>Authorization: Bearer …</code>). A token
		works with links and collections only: your account, members and space settings are out of its reach.
	</p>

	{#if created}
		<div class="mb-3 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3">
			<p class="text-sm font-medium text-gray-900 dark:text-white">Copy the token “{created.name}” now, it is not shown again.</p>
			<code class="mt-2 block select-all break-all rounded bg-white/60 p-2 text-xs text-gray-900 dark:bg-black/30 dark:text-white"
				>{created.token}</code
			>
			<p class="mt-2 text-xs text-gray-600 dark:text-gray-300">Try it:</p>
			<code class="mt-1 block select-all break-all text-xs text-gray-700 dark:text-gray-200">{example}</code>
			<div class="mt-3 flex justify-end gap-2">
				<Button size="sm" color="light" onclick={() => copy(example, 'Example')}>Copy example</Button>
				<Button size="sm" color="light" onclick={() => copy(created!.token, 'Token')}>Copy token</Button>
				<Button size="sm" onclick={() => (created = null)}>Done</Button>
			</div>
		</div>
	{/if}

	{#if tokens.length > 0}
		<ul class="mb-3 divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-gray-800 dark:border-gray-700">
			{#each tokens as token (token.id)}
				{@const expired = !!token.expiresAt && token.expiresAt < new Date()}
				<li class="flex items-center gap-3 px-3 py-2">
					<div class="min-w-0 flex-1">
						<p class="truncate text-sm font-medium text-gray-900 dark:text-white">
							{token.name}
							<span class="ms-1 rounded-md bg-gray-500/15 px-1.5 py-0.5 text-xs font-normal text-gray-600 dark:text-gray-300"
								>{token.scope === 'WRITE' ? 'Read & write' : 'Read only'}</span
							>
						</p>
						<p class="truncate text-xs {expired ? 'text-red-500 dark:text-red-400' : 'text-gray-500 dark:text-gray-400'}">
							{token.lastUsedAt ? `Last used ${timeAgo(token.lastUsedAt)}` : 'Never used'}
							· {expired
								? 'expired'
								: token.expiresAt
									? `expires ${token.expiresAt.toLocaleDateString()}`
									: 'never expires'}
						</p>
					</div>
					<Button size="xs" color="light" onclick={() => revoke(token)}>Revoke</Button>
				</li>
			{/each}
		</ul>
	{/if}

	{#if !allowed}
		<p class="rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-gray-900/40 dark:text-gray-300">
			API tokens are part of Pro{tokens.length > 0 ? ': yours rest until your account is Pro again' : ''}.
			<a href={p('/plan')} class="font-medium text-primary-600 hover:underline dark:text-primary-500">See your plan</a>
		</p>
	{:else if creating}
		<form class="flex flex-col gap-2" onsubmit={create}>
			<Input placeholder="What is it for? e.g. iPhone shortcut" maxlength={50} bind:value={name} required />
			<div class="grid grid-cols-2 gap-2">
				<select class={selectClass} aria-label="What the token may do" bind:value={scope}>
					<option value="READ">Read only</option>
					<option value="WRITE">Read & write</option>
				</select>
				<select class={selectClass} aria-label="When the token expires" bind:value={days}>
					{#each EXPIRY as option (option.days)}
						<option value={option.days}>{option.label}</option>
					{/each}
				</select>
			</div>
			<Input type="password" placeholder="Current password" autocomplete="current-password" bind:value={password} required />
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end gap-2">
				<Button size="sm" color="light" onclick={() => ((creating = false), (error = ''))}>Cancel</Button>
				<Button type="submit" size="sm" disabled={busy || !name.trim() || !password}>
					{busy ? 'Creating…' : 'Create token'}
				</Button>
			</div>
		</form>
	{:else}
		{#if error}
			<p class="mb-2 text-sm text-red-500 dark:text-red-400">{error}</p>
		{/if}
		<div class="flex justify-end">
			<Button size="sm" color="light" onclick={() => (creating = true)}>New token…</Button>
		</div>
	{/if}
</section>
