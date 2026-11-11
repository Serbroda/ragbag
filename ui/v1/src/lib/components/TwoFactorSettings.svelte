<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import type { TwoFactorSetupDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { session } from '../stores/spaces.svelte';
	import { showToast } from '../stores/toast.svelte';
	import { copyText } from '../utils/clipboard';
	import QrCode from './QrCode.svelte';

	/** The profile dialog: closing it forgets everything typed or shown here. */
	let { open }: { open: boolean } = $props();

	const userApi = new UserApi(apiConfig());

	let password = $state('');
	let code = $state('');
	let busy = $state(false);
	let error = $state('');
	/** While setting up: what the authenticator app needs, until the first code confirmed it. */
	let setup = $state<TwoFactorSetupDto | null>(null);
	/** Shown once after turning it on. */
	let recoveryCodes = $state<string[] | null>(null);
	let turningOff = $state(false);

	const enabled = $derived(session.me?.twoFactor ?? false);

	$effect(() => {
		if (open) return;
		password = code = error = '';
		setup = null;
		recoveryCodes = null;
		turningOff = false;
	});

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

	function start(e: Event) {
		e.preventDefault();
		run(async () => {
			setup = await userApi.setupTwoFactor({ twoFactorSetupRequest: { currentPassword: password } });
			password = '';
		}, 'Failed to start the setup.');
	}

	function turnOn(e: Event) {
		e.preventDefault();
		run(async () => {
			const result = await userApi.enableTwoFactor({ twoFactorCodeRequest: { code } });
			recoveryCodes = result.recoveryCodes;
			setup = null;
			code = '';
			if (session.me) session.me = { ...session.me, twoFactor: true };
		}, 'Failed to turn on two-factor authentication.');
	}

	function turnOff(e: Event) {
		e.preventDefault();
		run(async () => {
			await userApi.disableTwoFactor({ twoFactorDisableRequest: { currentPassword: password, code } });
			password = code = '';
			turningOff = false;
			recoveryCodes = null;
			if (session.me) session.me = { ...session.me, twoFactor: false };
			showToast('Two-factor authentication is off.');
		}, 'Failed to turn off two-factor authentication.');
	}

	async function copyCodes() {
		showToast((await copyText((recoveryCodes ?? []).join('\n'))) ? 'Recovery codes copied' : 'Could not copy the codes');
	}
</script>

<section>
	<Label class="mb-1">
		Two-factor authentication
		{#if enabled}
			<span class="ms-1 rounded-md bg-green-500/15 px-1.5 py-0.5 text-xs font-medium text-green-700 dark:text-green-300">On</span>
		{/if}
	</Label>
	<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
		{#if enabled}
			Signing in needs your password and the code of your authenticator app.
		{:else}
			Sign in with your password and a code from an authenticator app on your phone (Aegis, Google Authenticator,
			1Password, Bitwarden, …). Nobody gets in with your password alone.
		{/if}
	</p>

	{#if recoveryCodes}
		<div class="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3">
			<p class="text-sm font-medium text-gray-900 dark:text-white">Two-factor authentication is on. Keep these recovery codes.</p>
			<p class="mt-1 text-xs text-gray-600 dark:text-gray-300">
				Each one signs you in once without the app, e.g. after losing your phone. They are shown only now; without
				app and codes only an administrator can let you back in.
			</p>
			<ul class="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 font-mono text-sm text-gray-900 dark:text-white">
				{#each recoveryCodes as recoveryCode (recoveryCode)}
					<li>{recoveryCode}</li>
				{/each}
			</ul>
			<div class="mt-3 flex justify-end gap-2">
				<Button size="sm" color="light" onclick={copyCodes}>Copy</Button>
				<Button size="sm" onclick={() => (recoveryCodes = null)}>I saved them</Button>
			</div>
		</div>
	{:else if setup}
		<form class="flex flex-col gap-3" onsubmit={turnOn} autocomplete="off">
			<p class="text-sm text-gray-700 dark:text-gray-300">1. Scan this with your authenticator app:</p>
			<div class="flex flex-col items-center gap-2">
				<QrCode value={setup.uri} label="QR code for your authenticator app" />
				<p class="text-center text-xs text-gray-500 dark:text-gray-400">
					Cannot scan it? Type in this key:<br />
					<code class="select-all break-all text-gray-700 dark:text-gray-200">{setup.secret}</code>
				</p>
			</div>
			<p class="text-sm text-gray-700 dark:text-gray-300">2. Enter the 6 digits the app shows:</p>
			<Input
				inputmode="numeric"
				maxlength={7}
				placeholder="123 456"
				autocomplete="one-time-code"
				aria-label="Code of the authenticator app"
				bind:value={code}
				required
			/>
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end gap-2">
				<Button size="sm" color="light" onclick={() => ((setup = null), (code = error = ''))}>Cancel</Button>
				<Button type="submit" size="sm" disabled={busy || code.replace(/\s/g, '').length < 6}>
					{busy ? 'Checking…' : 'Turn on'}
				</Button>
			</div>
		</form>
	{:else if enabled}
		{#if turningOff}
			<form class="flex flex-col gap-2" onsubmit={turnOff} autocomplete="off">
				<Input type="password" placeholder="Current password" autocomplete="current-password" bind:value={password} required />
				<Input
					placeholder="Code of the app or a recovery code"
					autocomplete="one-time-code"
					aria-label="Code of the app or a recovery code"
					bind:value={code}
					required
				/>
				{#if error}
					<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
				{/if}
				<div class="flex justify-end gap-2">
					<Button size="sm" color="light" onclick={() => ((turningOff = false), (password = code = error = ''))}>Cancel</Button>
					<Button type="submit" size="sm" color="red" disabled={busy || !password || !code.trim()}>
						{busy ? 'Turning off…' : 'Turn off'}
					</Button>
				</div>
			</form>
		{:else}
			<div class="flex justify-end">
				<Button size="sm" color="light" onclick={() => (turningOff = true)}>Turn off…</Button>
			</div>
		{/if}
	{:else}
		<form class="flex flex-col gap-2" onsubmit={start}>
			<Input type="password" placeholder="Current password" autocomplete="current-password" bind:value={password} required />
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end">
				<Button type="submit" size="sm" disabled={busy || !password}>{busy ? 'Starting…' : 'Set up…'}</Button>
			</div>
		</form>
	{/if}
</section>
