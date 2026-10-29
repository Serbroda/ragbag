<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { apiConfig, errorMessage } from '../api/client';
	import { unlock } from '../stores/lock.svelte';
	import { session } from '../stores/spaces.svelte';

	/**
	 * Unlocks the user's locked collections with the PIN, or sets the PIN first ("setup", needs the password).
	 */
	let {
		open = $bindable(false),
		mode,
		ondone,
	}: { open?: boolean; mode: 'unlock' | 'setup'; ondone?: () => void } = $props();

	let pin = $state('');
	let repeat = $state('');
	let password = $state('');
	let busy = $state(false);
	let error = $state('');

	$effect(() => {
		if (open) {
			// The modal focuses its first control itself; the PIN field (or the password) is what matters here
			const id = mode === 'setup' ? 'lock-password' : 'lock-pin';
			setTimeout(() => document.getElementById(id)?.focus(), 50);
			return;
		}
		// Never keep secrets around after the dialog was closed
		pin = repeat = password = error = '';
	});

	async function submit(e: Event) {
		e.preventDefault();
		error = '';
		if (mode === 'setup' && pin !== repeat) {
			error = 'The PINs do not match.';
			return;
		}
		busy = true;
		try {
			if (mode === 'unlock') {
				await unlock(pin);
			} else {
				await new UserApi(apiConfig()).setPin({ setPinDto: { currentPassword: password, pin } });
				if (session.me) session.me = { ...session.me, hasPin: true };
			}
			open = false;
			ondone?.();
		} catch (err) {
			error = await errorMessage(err, mode === 'unlock' ? 'Wrong PIN.' : 'Could not save the PIN.');
			pin = repeat = '';
		} finally {
			busy = false;
		}
	}
</script>

<Modal title={mode === 'unlock' ? 'Unlock' : 'Set a PIN'} bind:open size="xs" autoclose={false}>
	<form class="flex flex-col gap-4" onsubmit={submit} autocomplete="off">
		{#if mode === 'setup'}
			<p class="text-sm text-gray-600 dark:text-gray-300">
				Locked collections are hidden until you unlock them with this PIN. Only you are affected.
			</p>
			<div>
				<Label for="lock-password" class="mb-2">Your password</Label>
				<Input id="lock-password" type="password" autocomplete="current-password" bind:value={password} required />
			</div>
		{/if}
		<div>
			<Label for="lock-pin" class="mb-2">{mode === 'setup' ? 'New PIN (4–8 digits)' : 'PIN'}</Label>
			<input
				id="lock-pin"
				type="password"
				inputmode="numeric"
				autocomplete="off"
				pattern="[0-9]*"
				maxlength={8}
				class="w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-center font-mono text-xl tracking-[0.5em] text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
				bind:value={pin}
				required
			/>
		</div>
		{#if mode === 'setup'}
			<div>
				<Label for="lock-repeat" class="mb-2">Repeat PIN</Label>
				<input
					id="lock-repeat"
					type="password"
					inputmode="numeric"
					autocomplete="off"
					maxlength={8}
					class="w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-center font-mono text-xl tracking-[0.5em] text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
					bind:value={repeat}
					required
				/>
			</div>
		{/if}
		{#if error}
			<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
		{/if}
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (open = false)}>Cancel</Button>
			<Button type="submit" disabled={busy || pin.length < 4}>{mode === 'unlock' ? 'Unlock' : 'Save PIN'}</Button>
		</div>
	</form>
</Modal>
