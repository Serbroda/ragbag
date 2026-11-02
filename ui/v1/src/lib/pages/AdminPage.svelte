<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AdminOverviewDto } from 'ragbag-frontend-sdk';
	import { Button } from 'flowbite-svelte';
	import { ArrowLeftOutline } from 'flowbite-svelte-icons';
	import { apiConfig, downloadBackup, errorMessage } from '../api/client';
	import { navigate, p, route } from '../router';
	import { loadMe, session } from '../stores/spaces.svelte';
	import AdminUsers from '../components/AdminUsers.svelte';
	import AdminReports from '../components/AdminReports.svelte';
	import Logo from '../components/Logo.svelte';

	/** Everything instance admins do, outside of any space: overview, accounts, reports and backups. */
	const TABS = [
		{ value: 'overview', label: 'Overview' },
		{ value: 'accounts', label: 'Accounts' },
		{ value: 'reports', label: 'Reports' },
		{ value: 'backup', label: 'Backup' },
	] as const;
	type Tab = (typeof TABS)[number]['value'];

	const tab = $derived<Tab>(TABS.find((t) => t.value === route.params.tab)?.value ?? 'overview');
	/** Set by "Block …" on a report: the accounts tab opens that account's block form. */
	let focus = $state<string | null>(null);

	let allowed = $state<boolean | null>(null);
	let overview = $state<AdminOverviewDto | null>(null);
	let error = $state('');
	let downloading = $state(false);

	loadMe()
		.then((me) => {
			allowed = !!me.instanceAdmin;
			if (allowed) loadOverview();
		})
		.catch(async (err) => {
			allowed = false;
			error = await errorMessage(err, 'Could not load your account.');
		});

	async function loadOverview() {
		try {
			overview = await new AdminApi(apiConfig()).getAdminOverview();
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the overview.');
		}
	}

	function open(value: Tab, account: string | null = null) {
		focus = account;
		navigate(value === 'overview' ? '/admin' : '/admin/:tab', { params: { tab: value } });
		if (value === 'overview') loadOverview();
	}

	async function backup() {
		downloading = true;
		error = '';
		try {
			await downloadBackup();
		} catch (err) {
			error = await errorMessage(err, 'Failed to create the backup.');
		} finally {
			downloading = false;
		}
	}

	function bytes(value: number): string {
		if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`;
		if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`;
		return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`;
	}

	function legalSummary(settings: AdminOverviewDto['settings']): string {
		const pages = [
			['imprint', settings.legalImprint],
			['privacy policy', settings.legalPrivacy],
			['terms of use', settings.legalTerms],
		].filter(([, link]) => link);
		return pages.length ? pages.map(([name, link]) => `${name} (${link})`).join(', ') : 'none';
	}

	function legalWarning(settings: AdminOverviewDto['settings']): string {
		if (settings.legalImprint && settings.legalPrivacy) return '';
		return 'A public instance needs at least an imprint and a privacy policy (in Germany § 5 DDG, Art. 13 GDPR); templates are in docs/legal of the repository';
	}

	const quota = (value: number, unit = '') => (value > 0 ? `${value.toLocaleString()}${unit}` : 'unlimited');
</script>

<svelte:head>
	<title>Administration · Ragbag</title>
</svelte:head>

<div class="min-h-dvh bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100">
	<header class="sticky top-0 z-20 border-b border-gray-200 bg-white/90 backdrop-blur dark:border-gray-950/60 dark:bg-gray-850/90">
		<div class="mx-auto flex h-12 w-full max-w-5xl items-center gap-3 px-4 md:px-8">
			<a
				href={p('/')}
				class="flex items-center gap-2 rounded-md px-1 py-1 text-sm text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
				title="Back to Ragbag"
			>
				<ArrowLeftOutline class="h-4 w-4" />
				<Logo size={24} />
			</a>
			<h1 class="font-semibold">Administration</h1>
		</div>
	</header>

	<main class="mx-auto w-full max-w-5xl px-4 py-6 md:px-8">
		{#if allowed === null}
			<p class="text-gray-500 dark:text-gray-400">Loading…</p>
		{:else if !allowed}
			<div class="mx-auto mt-16 max-w-md text-center">
				<p class="text-lg font-semibold">Only for instance admins</p>
				<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
					{error || 'Admins are the accounts listed in ADMIN_USERS.'}
				</p>
			</div>
		{:else}
			<div class="mb-6 flex gap-1 overflow-x-auto border-b border-gray-200 dark:border-gray-800" role="tablist">
				{#each TABS as item (item.value)}
					<button
						type="button"
						role="tab"
						aria-selected={tab === item.value}
						class="-mb-px shrink-0 border-b-2 px-3 py-2 text-sm font-medium {tab === item.value
							? 'border-primary-500 text-gray-900 dark:text-white'
							: 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300'}"
						onclick={() => open(item.value)}
					>
						{item.label}
						{#if item.value === 'reports' && session.me?.openReports}
							<span class="ms-1 rounded-full bg-red-500 px-1.5 text-xs text-white">{session.me.openReports}</span>
						{/if}
					</button>
				{/each}
			</div>

			{#if error}
				<p class="mb-4 text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}

			{#if tab === 'overview'}
				{#if overview}
					{@const o = overview}
					{@const s = o.settings}
					<div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
						{#each [['Accounts', o.users.toLocaleString(), [o.blockedUsers ? `${o.blockedUsers} blocked` : '', o.unconfirmedUsers ? `${o.unconfirmedUsers} unconfirmed` : ''].filter(Boolean).join(' · ')], ['Spaces', o.spaces.toLocaleString(), `${o.collections.toLocaleString()} collections`], ['Links', o.bookmarks.toLocaleString(), `${o.publicLinks} public link${o.publicLinks === 1 ? '' : 's'}`], ['Images', bytes(o.imageBytes), 'avatars, icons, previews']] as [label, value, hint] (label)}
							<div class="rounded-lg border border-gray-200 p-4 dark:border-gray-800">
								<p class="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
								<p class="mt-1 text-2xl font-semibold">{value}</p>
								<p class="mt-0.5 truncate text-xs text-gray-500">{hint || ' '}</p>
							</div>
						{/each}
					</div>

					{#if o.openReports > 0}
						<button
							type="button"
							class="mt-4 w-full rounded-lg bg-amber-500/10 p-3 text-left text-sm text-amber-700 ring-1 ring-inset ring-amber-500/30 hover:bg-amber-500/15 dark:text-amber-300"
							onclick={() => open('reports')}
						>
							{o.openReports} open report{o.openReports === 1 ? '' : 's'} of public links: decide now →
						</button>
					{/if}

					<h2 class="mb-2 mt-8 text-sm font-semibold">Configuration</h2>
					<p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
						Set by environment variables (<code>/etc/ragbag/ragbag.env</code>, <code>.env</code> or the compose file);
						changes need a restart.
					</p>
					<dl class="divide-y divide-gray-100 rounded-lg border border-gray-200 text-sm dark:divide-gray-800 dark:border-gray-800">
						{#each [
							['Version', `${o.version} · ${o.database === 'postgres' ? 'PostgreSQL' : 'SQLite'} · schema ${o.schemaVersion}`, ''],
							['Address (PUBLIC_URL)', s.publicUrl || 'not set', s.publicUrl ? '' : 'Links in emails need it'],
							['Registration (REGISTRATION_ENABLED)', s.registrationOpen ? 'open' : 'closed', ''],
							['Email (SMTP_HOST)', s.email ? 'on: confirmation, password reset, reports' : 'off', s.email ? '' : 'Without it, new accounts are not confirmed and admins hand out reset links'],
							['Rate limits (RATE_LIMIT_ENABLED)', s.rateLimits ? 'on' : 'off', s.rateLimits ? '' : 'Turn them on for a public instance'],
							['Own network (ALLOW_PRIVATE_HOSTS)', s.allowPrivateHosts ? 'allowed' : 'refused', s.allowPrivateHosts && s.registrationOpen ? 'With open registration, anybody can make the server load addresses of your network' : ''],
							['Admins (ADMIN_USERS)', s.adminUsers.length ? s.adminUsers.join(', ') : 'the account created on the first start', ''],
							['Legal pages (DATA_DIR/legal, *_URL)', legalSummary(s), legalWarning(s)],
							['Quotas (QUOTA_*)', `${quota(s.quotaLinksPerUser)} links and ${quota(s.quotaSpacesPerUser)} spaces per account, ${quota(s.quotaCollectionsPerSpace)} collections per space, ${quota(s.quotaUploadMbPerUser, ' MB')} images per account`, ''],
						] as [label, value, warning] (label)}
							<div class="flex flex-col gap-0.5 px-4 py-2.5 sm:flex-row sm:gap-4">
								<dt class="shrink-0 text-gray-500 sm:w-72">{label}</dt>
								<dd class="min-w-0">
									<span class="break-words">{value}</span>
									{#if warning}<span class="block text-xs text-amber-600 dark:text-amber-400">{warning}</span>{/if}
								</dd>
							</div>
						{/each}
					</dl>
				{:else}
					<p class="text-gray-500 dark:text-gray-400">Loading…</p>
				{/if}
			{:else if tab === 'accounts'}
				<AdminUsers {focus} />
			{:else if tab === 'reports'}
				<div class="flex flex-col gap-4">
					<AdminReports onblock={(userId) => open('accounts', userId)} />
				</div>
			{:else}
				<div class="flex max-w-2xl flex-col gap-6 text-sm">
					<section>
						<h2 class="mb-1 font-semibold">Download a backup</h2>
						<p class="mb-3 text-gray-500 dark:text-gray-400">
							One file with everything: accounts with their password hashes, spaces, collections, links, images
							and public links. Keep it safe. It restores on the first login of a new, empty Ragbag – SQLite or
							PostgreSQL, also of a newer version.
						</p>
						<Button disabled={downloading} onclick={backup}>{downloading ? 'Preparing…' : 'Download backup'}</Button>
					</section>
					<section>
						<h2 class="mb-1 font-semibold">Automatic backups</h2>
						<p class="mb-2 text-gray-500 dark:text-gray-400">
							On the server, <code>ragbag backup FILE</code> writes the same file (it only needs
							<code>DATABASE_URL</code>), e.g. every night from cron or a systemd timer:
						</p>
						<pre class="overflow-x-auto rounded-lg bg-gray-100 p-3 text-xs dark:bg-gray-950/70"><code
								># systemd service (install.sh), while the service runs
sudo sh -c 'set -a; . /etc/ragbag/ragbag.env; ragbag backup /var/backups/ragbag-$(date +%F).json.gz'

# Docker
docker compose exec -T app ragbag backup &gt; ragbag-$(date +%F).json.gz</code
							></pre>
					</section>
					<section>
						<h2 class="mb-1 font-semibold">Restoring</h2>
						<p class="text-gray-500 dark:text-gray-400">
							Install Ragbag on the new server with an empty database, sign in with the password in
							<code>initial-admin-password</code> and choose “Restore a backup”, or run
							<code>ragbag restore FILE</code> before the first start.
						</p>
					</section>
				</div>
			{/if}
		{/if}
	</main>
</div>
