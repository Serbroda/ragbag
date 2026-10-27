<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { apiConfig, changePassword, errorMessage } from '../api/client';
	import { session } from '../stores/spaces.svelte';
	import { LINK_VIEWS, preferences, setLinkView } from '../stores/preferences.svelte';
	import ImagePicker from './ImagePicker.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const userApi = new UserApi(apiConfig());
	const MIN_PASSWORD_LENGTH = 8;

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordBusy = $state(false);
	let passwordMessage = $state<{ ok: boolean; text: string } | null>(null);

	$effect(() => {
		if (open) return;
		// Never keep passwords around after the dialog was closed
		currentPassword = newPassword = confirmPassword = '';
		passwordMessage = null;
	});

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
