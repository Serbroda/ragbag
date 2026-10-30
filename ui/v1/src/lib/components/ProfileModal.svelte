<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { apiConfig, appBase, changePassword, errorMessage } from '../api/client';
	import { session, setStartSpace } from '../stores/spaces.svelte';
	import { LINK_VIEWS, preferences, setLinkView } from '../stores/preferences.svelte';
	import { lockState } from '../stores/lock.svelte';
	import ImagePicker from './ImagePicker.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const userApi = new UserApi(apiConfig());
	const MIN_PASSWORD_LENGTH = 8;

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordBusy = $state(false);
	let passwordMessage = $state<{ ok: boolean; text: string } | null>(null);

	/** Saves the page that is open in the browser: drag it to the bookmarks bar. */
	const bookmarklet = $derived(
		`javascript:(()=>{window.open('${window.location.origin}${appBase()}/add?url='+encodeURIComponent(location.href)+'&title='+encodeURIComponent(document.title),'ragbag','width=560,height=760')})()`,
	);

	let pinPassword = $state('');
	let pin = $state('');
	let pinRepeat = $state('');
	let pinBusy = $state(false);
	let pinMessage = $state<{ ok: boolean; text: string } | null>(null);

	$effect(() => {
		if (open) return;
		// Never keep passwords around after the dialog was closed
		currentPassword = newPassword = confirmPassword = '';
		passwordMessage = null;
		pinPassword = pin = pinRepeat = '';
		pinMessage = null;
	});

	async function savePin(e: Event) {
		e.preventDefault();
		if (pin !== pinRepeat) {
			pinMessage = { ok: false, text: 'The PINs do not match.' };
			return;
		}
		pinBusy = true;
		pinMessage = null;
		try {
			await userApi.setPin({ setPinDto: { currentPassword: pinPassword, pin } });
			if (session.me) session.me = { ...session.me, hasPin: true };
			pinPassword = pin = pinRepeat = '';
			pinMessage = { ok: true, text: 'PIN saved.' };
		} catch (err) {
			pinMessage = { ok: false, text: await errorMessage(err, 'Failed to save the PIN.') };
		} finally {
			pinBusy = false;
		}
	}

	async function savePassword(e: Event) {
		e.preventDefault();
		if (newPassword !== confirmPassword) {
			passwordMessage = { ok: false, text: 'The new passwords do not match.' };
			return;
		}
		passwordBusy = true;
		passwordMessage = null;
		try {
			await changePassword(currentPassword, newPassword);
			currentPassword = newPassword = confirmPassword = '';
			passwordMessage = { ok: true, text: 'Password changed. Your other devices have been signed out.' };
		} catch (err) {
			passwordMessage = { ok: false, text: await errorMessage(err, 'Failed to change the password.') };
		} finally {
			passwordBusy = false;
		}
	}
</script>

<Modal bind:open size="sm" title="Profile & appearance">
	{#if session.me}
		<div class="flex flex-col gap-6">
			<section>
				<Label class="mb-3">Avatar</Label>
				<ImagePicker
					name={session.me.username}
					imageId={session.me.avatarId}
					onupload={async (file) => {
						session.me = await userApi.setAvatar({ file });
					}}
					onurl={async (url) => {
						session.me = await userApi.setAvatar({ url });
					}}
					onremove={async () => {
						session.me = await userApi.removeAvatar();
					}}
				/>
				<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
					Signed in as <strong>{session.me.username}</strong> ({session.me.email})
				</p>
			</section>

			<section>
				<Label class="mb-3">Change password</Label>
				<form class="flex flex-col gap-2" onsubmit={savePassword}>
					<!-- Lets password managers update the right entry -->
					<input type="text" autocomplete="username" value={session.me.username} hidden readonly />
					<Input
						type="password"
						placeholder="Current password"
						autocomplete="current-password"
						bind:value={currentPassword}
						required
					/>
					<Input
						type="password"
						placeholder="New password (at least {MIN_PASSWORD_LENGTH} characters)"
						autocomplete="new-password"
						minlength={MIN_PASSWORD_LENGTH}
						bind:value={newPassword}
						required
					/>
					<Input
						type="password"
						placeholder="Repeat new password"
						autocomplete="new-password"
						bind:value={confirmPassword}
						required
					/>
					{#if passwordMessage}
						<p class="text-sm {passwordMessage.ok ? 'text-green-600 dark:text-green-400' : 'text-red-500 dark:text-red-400'}">
							{passwordMessage.text}
						</p>
					{/if}
					<div class="flex justify-end">
						<Button
							type="submit"
							size="sm"
							disabled={passwordBusy || !currentPassword || newPassword.length < MIN_PASSWORD_LENGTH}
						>
							{passwordBusy ? 'Saving…' : 'Change password'}
						</Button>
					</div>
				</form>
			</section>

			<section>
				<Label class="mb-1">{session.me.hasPin ? 'Change PIN' : 'PIN for locked collections'}</Label>
				<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
					“Lock for me” hides a collection from your view (right-click it in the sidebar). The PIN unlocks them
					for a few minutes{lockState.unlocked ? ' – they are unlocked right now' : ''}.
				</p>
				<form class="flex flex-col gap-2" onsubmit={savePin} autocomplete="off">
					<Input type="password" placeholder="Current password" autocomplete="current-password" bind:value={pinPassword} required />
					<div class="grid grid-cols-2 gap-2">
						<Input type="password" inputmode="numeric" maxlength={8} placeholder="PIN (4–8 digits)" autocomplete="off" bind:value={pin} required />
						<Input type="password" inputmode="numeric" maxlength={8} placeholder="Repeat PIN" autocomplete="off" bind:value={pinRepeat} required />
					</div>
					{#if pinMessage}
						<p class="text-sm {pinMessage.ok ? 'text-green-600 dark:text-green-400' : 'text-red-500 dark:text-red-400'}">
							{pinMessage.text}
						</p>
					{/if}
					<div class="flex justify-end">
						<Button type="submit" size="sm" disabled={pinBusy || !pinPassword || pin.length < 4}>
							{pinBusy ? 'Saving…' : session.me.hasPin ? 'Change PIN' : 'Set PIN'}
						</Button>
					</div>
				</form>
			</section>

			<section>
				<Label for="start-space" class="mb-1">Open on start</Label>
				<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
					The space Ragbag opens with, on every device. You can also right-click a space in the bar.
				</p>
				<select
					id="start-space"
					class="w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
					value={session.me.startSpaceId ?? ''}
					onchange={(e) => setStartSpace((e.currentTarget as HTMLSelectElement).value || null)}
				>
					<option value="">The space I was in last</option>
					{#each session.spaces as space (space.id)}
						<option value={space.id}>{space.name}</option>
					{/each}
				</select>
			</section>

			<section>
				<Label class="mb-1">Save links from anywhere</Label>
				<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
					Drag this button to your browser's bookmarks bar. Clicking it on any page opens a small window to save
					that page to Ragbag. On Android, install Ragbag as an app (browser menu → “Install app”) to share links
					to it from other apps.
				</p>
				<!-- svelte-ignore a11y_invalid_attribute -->
				<a
					href={bookmarklet}
					class="inline-flex cursor-grab items-center gap-1.5 rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-semibold text-white shadow-sm"
					title="Drag me to the bookmarks bar"
					onclick={(e) => e.preventDefault()}>＋ Save to Ragbag</a
				>
			</section>

			<section>
				<Label class="mb-1">Link display</Label>
				<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">Saved for you on this device.</p>
				<div class="grid grid-cols-3 gap-2" role="radiogroup" aria-label="Link display">
					{#each LINK_VIEWS as view (view.value)}
						{@const active = preferences.linkView === view.value}
						<button
							type="button"
							role="radio"
							aria-checked={active}
							class="rounded-lg border px-3 py-2 text-left transition-colors {active
								? 'border-primary-500 bg-primary-500/10'
								: 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'}"
							onclick={() => setLinkView(view.value)}
						>
							<span class="block text-sm font-medium text-gray-900 dark:text-white">{view.label}</span>
							<span class="block text-xs text-gray-500 dark:text-gray-400">{view.description}</span>
						</button>
					{/each}
				</div>
			</section>
		</div>
	{/if}
</Modal>
