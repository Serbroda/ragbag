<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import { Label, Modal } from 'flowbite-svelte';
	import { apiConfig } from '../api/client';
	import { session } from '../stores/spaces.svelte';
	import { LINK_VIEWS, preferences, setLinkView } from '../stores/preferences.svelte';
	import ImagePicker from './ImagePicker.svelte';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	const userApi = new UserApi(apiConfig());
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
