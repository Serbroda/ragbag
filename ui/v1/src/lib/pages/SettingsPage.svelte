<script lang="ts">
	import { UserApi } from 'ragbag-frontend-sdk';
	import type { Component } from 'svelte';
	import { Button, Input } from 'flowbite-svelte';
	import {
		ArrowLeftOutline,
		CodeOutline,
		LockOutline,
		PaletteOutline,
		UserCircleOutline,
		UserSettingsOutline,
	} from 'flowbite-svelte-icons';
	import {
		apiConfig, appBase, changePassword, deleteAccount, errorMessage, fetchFeatureFlags, logoutEverywhere,
	} from '../api/client';
	import { navigate, p, route } from '../router';
	import { loadMe, loadSpaces, resetSession, session, setStartSpace } from '../stores/spaces.svelte';
	import { showToast } from '../stores/toast.svelte';
	import { LINK_VIEWS, preferences, setLinkView } from '../stores/preferences.svelte';
	import { lockState, resetLock } from '../stores/lock.svelte';
	import { shownName } from '../utils/format';
	import ImagePicker from '../components/ImagePicker.svelte';
	import Logo from '../components/Logo.svelte';
	import SettingsItem from '../components/SettingsItem.svelte';
	import TwoFactorSettings from '../components/TwoFactorSettings.svelte';
	import ApiTokenSettings from '../components/ApiTokenSettings.svelte';
	import Toasts from '../components/Toasts.svelte';

	/**
	 * Everything personal, outside of any space. Each tab is a list of settings that show how they are set; forms
	 * (and the password some of them need) only appear for the one being changed.
	 */
	const TABS = [
		{ value: 'profile', label: 'Profile', icon: UserCircleOutline },
		{ value: 'security', label: 'Security', icon: LockOutline },
		{ value: 'appearance', label: 'Appearance', icon: PaletteOutline },
		{ value: 'apps', label: 'Apps & API', icon: CodeOutline },
		{ value: 'account', label: 'Account', icon: UserSettingsOutline },
	] as const satisfies readonly { value: string; label: string; icon: Component<{ class?: string }> }[];
	type Tab = (typeof TABS)[number]['value'];

	const tab = $derived<Tab>(TABS.find((t) => t.value === route.params.tab)?.value ?? 'profile');

	function open(value: Tab) {
		navigate(value === 'profile' ? '/settings' : '/settings/:tab', { params: { tab: value } });
	}

	const userApi = new UserApi(apiConfig());
	const MIN_PASSWORD_LENGTH = 8;
	let loadError = $state('');
	let plansEnabled = $state(false);

	// Opened directly (reload, link): the session is not there yet
	Promise.all([loadMe(), session.loaded ? Promise.resolve() : loadSpaces()]).catch(async (err) => {
		loadError = await errorMessage(err, 'Could not load your account.');
	});
	fetchFeatureFlags().then((flags) => (plansEnabled = flags['plans'] === 'true')).catch(() => {});

	// --- Profile ---
	let displayName = $state('');
	let nameBusy = $state(false);
	let nameMessage = $state<{ ok: boolean; text: string } | null>(null);
	let nameLoaded = false;
	$effect(() => {
		if (session.me && !nameLoaded) {
			nameLoaded = true;
			displayName = session.me.displayName ?? '';
		}
	});

	async function saveDisplayName(e: Event) {
		e.preventDefault();
		nameBusy = true;
		nameMessage = null;
		try {
			session.me = await userApi.setDisplayName({ displayNameDto: { displayName } });
			displayName = session.me.displayName ?? '';
			nameMessage = { ok: true, text: 'Saved.' };
		} catch (err) {
			nameMessage = { ok: false, text: await errorMessage(err, 'Failed to save the display name.') };
		} finally {
			nameBusy = false;
		}
	}

	// --- Security ---
	let passwordOpen = $state(false);
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordBusy = $state(false);
	let passwordError = $state('');

	async function savePassword(e: Event) {
		e.preventDefault();
		if (newPassword !== confirmPassword) {
			passwordError = 'The new passwords do not match.';
			return;
		}
		passwordBusy = true;
		passwordError = '';
		try {
			await changePassword(currentPassword, newPassword);
			closePassword();
			showToast('Password changed. Your other devices have been signed out.');
		} catch (err) {
			passwordError = await errorMessage(err, 'Failed to change the password.');
		} finally {
			passwordBusy = false;
		}
	}

	function closePassword() {
		passwordOpen = false;
		currentPassword = newPassword = confirmPassword = passwordError = '';
	}

	let pinOpen = $state(false);
	let pinPassword = $state('');
	let pin = $state('');
	let pinRepeat = $state('');
	let pinBusy = $state(false);
	let pinError = $state('');

	async function savePin(e: Event) {
		e.preventDefault();
		if (pin !== pinRepeat) {
			pinError = 'The PINs do not match.';
			return;
		}
		pinBusy = true;
		pinError = '';
		try {
			await userApi.setPin({ setPinDto: { currentPassword: pinPassword, pin } });
			if (session.me) session.me = { ...session.me, hasPin: true };
			closePin();
			showToast('PIN saved.');
		} catch (err) {
			pinError = await errorMessage(err, 'Failed to save the PIN.');
		} finally {
			pinBusy = false;
		}
	}

	function closePin() {
		pinOpen = false;
		pinPassword = pin = pinRepeat = pinError = '';
	}

	let signOutOpen = $state(false);

	async function signOutEverywhere() {
		resetLock();
		await logoutEverywhere().catch(() => {});
		resetSession();
		navigate('/login');
	}

	// --- Apps ---
	/** Saves the page that is open in the browser: drag it to the bookmarks bar. */
	const bookmarklet = $derived(
		`javascript:(()=>{window.open('${window.location.origin}${appBase()}/add?url='+encodeURIComponent(location.href)+'&title='+encodeURIComponent(document.title),'ragbag','width=560,height=760')})()`,
	);

	// --- Account ---
	let deleteOpen = $state(false);
	let deletePassword = $state('');
	let deleteBusy = $state(false);
	let deleteError = $state('');

	async function removeAccount(e: Event) {
		e.preventDefault();
		deleteBusy = true;
		deleteError = '';
		try {
			await deleteAccount(deletePassword);
			resetSession();
			navigate('/login');
			showToast('Your account was deleted.');
		} catch (err) {
			deleteError = await errorMessage(err, 'Failed to delete the account.');
		} finally {
			deleteBusy = false;
		}
	}

	// Leaving a tab forgets what was typed there, passwords above all
	$effect(() => {
		if (tab !== 'security') {
			closePassword();
			closePin();
			signOutOpen = false;
		}
		if (tab !== 'account') {
			deleteOpen = false;
			deletePassword = deleteError = '';
		}
	});

	const card = 'rounded-lg border border-gray-200 p-4 dark:border-gray-800 sm:p-5';
	const rows = 'divide-y divide-gray-100 dark:divide-gray-800';
	const selectClass =
		'w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white';
</script>

<svelte:head>
	<title>Settings · Ragbag</title>
</svelte:head>

<div class="min-h-dvh bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100">
	<header class="sticky top-0 z-20 border-b border-gray-200 bg-white/90 backdrop-blur dark:border-gray-950/60 dark:bg-gray-850/90">
		<div class="mx-auto flex h-12 w-full max-w-4xl items-center gap-3 px-4 md:px-8">
			<a
				href={p('/')}
				class="flex items-center gap-2 rounded-md px-1 py-1 text-sm text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
				title="Back to Ragbag"
			>
				<ArrowLeftOutline class="h-4 w-4" />
				<Logo size={24} />
			</a>
			<h1 class="font-semibold">Settings</h1>
		</div>
	</header>

	<main class="mx-auto flex w-full max-w-4xl flex-col gap-4 px-4 py-6 md:flex-row md:gap-8 md:px-8">
		<!-- Tabs: a scrolling row on phones, a list next to the content on wider screens -->
		<nav
			class="-mx-4 flex shrink-0 gap-1 overflow-x-auto border-b border-gray-200 px-4 dark:border-gray-800 md:mx-0 md:w-44 md:flex-col md:overflow-visible md:border-b-0 md:px-0"
			aria-label="Settings"
		>
			{#each TABS as item (item.value)}
				{@const Icon = item.icon}
				<button
					type="button"
					aria-current={tab === item.value ? 'page' : undefined}
					class="-mb-px flex shrink-0 items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium md:mb-0 md:rounded-md md:border-b-0 {tab ===
					item.value
						? 'border-primary-500 text-gray-900 dark:text-white md:bg-gray-100 md:dark:bg-gray-800'
						: 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 md:hover:bg-gray-100/60 md:dark:hover:bg-gray-800/50'}"
					onclick={() => open(item.value)}
				>
					<Icon class="hidden h-4 w-4 md:block" />
					{item.label}
				</button>
			{/each}
		</nav>

		<div class="min-w-0 flex-1">
			{#if loadError}
				<p class="text-sm text-red-500 dark:text-red-400">{loadError}</p>
			{:else if !session.me}
				<p class="text-gray-500 dark:text-gray-400">Loading…</p>
			{:else if tab === 'profile'}
				<section class="{card} {rows}">
					<SettingsItem title="Avatar">
						<ImagePicker
							name={shownName(session.me)}
							imageId={session.me.avatarId}
							onupload={async (file) => {
								session.me = await userApi.setAvatar({ file });
							}}
							onremove={async () => {
								session.me = await userApi.removeAvatar();
							}}
						/>
					</SettingsItem>
					<SettingsItem
						title="Display name"
						description="How others see you, e.g. with capitals and spaces. You keep signing in with your username."
					>
						<form class="flex gap-2" onsubmit={saveDisplayName}>
							<Input
								bind:value={displayName}
								maxlength={50}
								placeholder={session.me.username}
								aria-label="Display name"
								class="flex-1"
							/>
							<Button type="submit" color="light" disabled={nameBusy || displayName.trim() === (session.me.displayName ?? '')}>
								{nameBusy ? 'Saving…' : 'Save'}
							</Button>
						</form>
						{#if nameMessage}
							<p class="mt-2 text-xs {nameMessage.ok ? 'text-green-600 dark:text-green-400' : 'text-red-500 dark:text-red-400'}">
								{nameMessage.text}
							</p>
						{/if}
					</SettingsItem>
					<SettingsItem title="Username" description="What you sign in with; it cannot be changed.">
						<p class="text-sm text-gray-700 dark:text-gray-300">
							<span class="font-medium">{session.me.username}</span>
							<span class="text-gray-500 dark:text-gray-400">· {session.me.email}</span>
						</p>
					</SettingsItem>
				</section>
			{:else if tab === 'security'}
				<section class="{card} {rows}">
					<SettingsItem
						title="Password"
						description="Changing it signs out your other devices."
						action="Change…"
						bind:open={passwordOpen}
					>
						<form class="flex flex-col gap-2" onsubmit={savePassword}>
							<!-- Lets password managers update the right entry -->
							<input type="text" autocomplete="username" value={session.me.username} hidden readonly />
							<!-- svelte-ignore a11y_autofocus -->
							<Input
								type="password"
								placeholder="Current password"
								autocomplete="current-password"
								bind:value={currentPassword}
								required
								autofocus
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
							{#if passwordError}
								<p class="text-sm text-red-500 dark:text-red-400">{passwordError}</p>
							{/if}
							<div class="flex justify-end gap-2">
								<Button size="sm" color="light" onclick={closePassword}>Cancel</Button>
								<Button
									type="submit"
									size="sm"
									disabled={passwordBusy || !currentPassword || newPassword.length < MIN_PASSWORD_LENGTH}
								>
									{passwordBusy ? 'Saving…' : 'Change password'}
								</Button>
							</div>
						</form>
					</SettingsItem>

					<TwoFactorSettings open={tab === 'security'} />

					<SettingsItem
						title="PIN for locked collections"
						status={session.me.hasPin ? 'Set' : 'Not set'}
						tone={session.me.hasPin ? 'good' : 'neutral'}
						description="“Lock for me” hides a collection from your view (right-click it in the sidebar). The PIN unlocks them for a few minutes{lockState.unlocked
							? ' – they are unlocked right now'
							: ''}."
						action={session.me.hasPin ? 'Change…' : 'Set a PIN…'}
						bind:open={pinOpen}
					>
						<form class="flex flex-col gap-2" onsubmit={savePin} autocomplete="off">
							<div class="grid grid-cols-2 gap-2">
								<!-- svelte-ignore a11y_autofocus -->
								<Input
									type="password"
									inputmode="numeric"
									maxlength={8}
									placeholder="PIN (4–8 digits)"
									autocomplete="off"
									bind:value={pin}
									required
									autofocus
								/>
								<Input
									type="password"
									inputmode="numeric"
									maxlength={8}
									placeholder="Repeat PIN"
									autocomplete="off"
									bind:value={pinRepeat}
									required
								/>
							</div>
							<Input
								type="password"
								placeholder="Confirm with your password"
								autocomplete="current-password"
								bind:value={pinPassword}
								required
							/>
							{#if pinError}
								<p class="text-sm text-red-500 dark:text-red-400">{pinError}</p>
							{/if}
							<div class="flex justify-end gap-2">
								<Button size="sm" color="light" onclick={closePin}>Cancel</Button>
								<Button type="submit" size="sm" disabled={pinBusy || !pinPassword || pin.length < 4}>
									{pinBusy ? 'Saving…' : 'Save PIN'}
								</Button>
							</div>
						</form>
					</SettingsItem>

					<SettingsItem
						title="Sign out everywhere"
						description="Ends your sessions on all devices, this one included. API tokens keep working."
						action="Sign out…"
						bind:open={signOutOpen}
					>
						<div class="flex items-center justify-end gap-2">
							<span class="me-auto text-sm text-gray-700 dark:text-gray-300">You will have to sign in again.</span>
							<Button size="sm" color="light" onclick={() => (signOutOpen = false)}>Cancel</Button>
							<Button size="sm" color="red" onclick={signOutEverywhere}>Sign out everywhere</Button>
						</div>
					</SettingsItem>
				</section>
			{:else if tab === 'appearance'}
				<section class="{card} {rows}">
					<SettingsItem title="Link display" description="How links are shown; saved for you on this device.">
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
					</SettingsItem>
					<SettingsItem
						title="Open on start"
						description="The space Ragbag opens with, on every device. You can also right-click a space in the bar."
					>
						<select
							class={selectClass}
							aria-label="Open on start"
							value={session.me.startSpaceId ?? ''}
							onchange={(e) => setStartSpace((e.currentTarget as HTMLSelectElement).value || null)}
						>
							<option value="">The space I was in last</option>
							{#each session.spaces as space (space.id)}
								<option value={space.id}>{space.name}</option>
							{/each}
						</select>
					</SettingsItem>
				</section>
			{:else if tab === 'apps'}
				<section class="{card} {rows}">
					<SettingsItem
						title="Save links from anywhere"
						description="Drag this button to your browser's bookmarks bar. Clicking it on any page opens a small window to save that page to Ragbag. On Android, install Ragbag as an app (browser menu → “Install app”) to share links to it from other apps."
					>
						<!-- svelte-ignore a11y_invalid_attribute -->
						<a
							href={bookmarklet}
							class="inline-flex cursor-grab items-center gap-1.5 rounded-lg bg-primary-500 px-3 py-1.5 text-sm font-semibold text-white shadow-sm"
							title="Drag me to the bookmarks bar"
							onclick={(e) => e.preventDefault()}>＋ Save to Ragbag</a
						>
					</SettingsItem>
					<ApiTokenSettings open={tab === 'apps'} />
				</section>
			{:else}
				<section class="{card} {rows}">
					{#if plansEnabled}
						<SettingsItem title="Plan" description="Your plan, what counts against its limits and what Pro adds.">
							<a href={p('/plan')} class="text-sm font-medium text-primary-600 hover:underline dark:text-primary-500">
								See your plan
							</a>
						</SettingsItem>
					{/if}
					<SettingsItem
						title="Delete account"
						description="Deletes your account and personal data for good. Spaces only you are a member of are deleted with everything in them (export collections first if you want to keep them); links you added to spaces shared with others stay there without your name. Spaces you are the only owner of need another owner first."
						action="Delete account…"
						danger
						bind:open={deleteOpen}
					>
						<form class="flex flex-col gap-2" onsubmit={removeAccount}>
							<input type="text" autocomplete="username" value={session.me.username} hidden readonly />
							<!-- svelte-ignore a11y_autofocus -->
							<Input
								type="password"
								placeholder="Confirm with your password"
								autocomplete="current-password"
								bind:value={deletePassword}
								required
								autofocus
							/>
							{#if deleteError}
								<p class="text-sm text-red-500 dark:text-red-400">{deleteError}</p>
							{/if}
							<div class="flex justify-end gap-2">
								<Button size="sm" color="light" onclick={() => ((deleteOpen = false), (deletePassword = deleteError = ''))}>
									Cancel
								</Button>
								<Button type="submit" size="sm" color="red" disabled={deleteBusy || !deletePassword}>
									{deleteBusy ? 'Deleting…' : 'Delete my account'}
								</Button>
							</div>
						</form>
					</SettingsItem>
				</section>
			{/if}
		</div>
	</main>
</div>

<Toasts />
