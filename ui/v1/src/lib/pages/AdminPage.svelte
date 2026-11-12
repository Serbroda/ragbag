<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AdminOverviewDto, BackupsDto, TermsStateDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label } from 'flowbite-svelte';
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
	let creating = $state(false);
	let backups = $state<BackupsDto | null>(null);
	let terms = $state<TermsStateDto | null>(null);
	let termsOpen = $state(false);
	let termsDate = $state('');
	let termsBusy = $state(false);

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
			const admin = new AdminApi(apiConfig());
			[overview, terms] = await Promise.all([admin.getAdminOverview(), admin.getTermsState()]);
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the overview.');
		}
	}

	function open(value: Tab, account: string | null = null) {
		focus = account;
		navigate(value === 'overview' ? '/admin' : '/admin/:tab', { params: { tab: value } });
		if (value === 'overview') loadOverview();
	}

	$effect(() => {
		if (allowed && tab === 'backup') loadBackups();
	});

	async function loadBackups() {
		try {
			backups = await new AdminApi(apiConfig()).getBackups();
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the backups.');
		}
	}

	async function createBackup() {
		creating = true;
		error = '';
		try {
			backups = await new AdminApi(apiConfig()).createBackup();
		} catch (err) {
			error = await errorMessage(err, 'The backup failed.');
		} finally {
			creating = false;
		}
	}

	async function backup(stored?: string) {
		downloading = true;
		error = '';
		try {
			await downloadBackup(stored);
		} catch (err) {
			error = await errorMessage(err, 'Failed to create the backup.');
		} finally {
			downloading = false;
		}
	}

	/** Today, as the value of a date field. */
	function today(): string {
		const now = new Date();
		return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
	}

	function openTerms() {
		termsDate = today();
		termsOpen = true;
	}

	async function changeTerms(e: Event) {
		e.preventDefault();
		termsBusy = true;
		error = '';
		try {
			// A later day applies from its start (local time), today right away
			const [year, month, day] = termsDate.split('-').map(Number);
			const effectiveAt = termsDate > today() ? new Date(year, month - 1, day) : undefined;
			terms = await new AdminApi(apiConfig()).changeTerms({ changeTermsRequest: { effectiveAt } });
			termsOpen = false;
		} catch (err) {
			error = await errorMessage(err, 'Failed to record the change.');
		} finally {
			termsBusy = false;
		}
	}

	async function withdrawTerms() {
		termsBusy = true;
		error = '';
		try {
			terms = await new AdminApi(apiConfig()).withdrawTermsChange();
		} catch (err) {
			error = await errorMessage(err, 'Failed to withdraw the change.');
		} finally {
			termsBusy = false;
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
	const quotas = (links: number, spaces: number, collections: number, uploadMb: number) =>
		`${quota(links)} links and ${quota(spaces)} spaces per account, ${quota(collections)} collections per space, ${quota(uploadMb, ' MB')} images per account`;
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

					{#if terms?.terms}
						{@const t = terms}
						<h2 class="mb-2 mt-8 text-sm font-semibold">Terms of use</h2>
						<div class="rounded-lg border border-gray-200 p-4 text-sm dark:border-gray-800">
							<p>
								{#if t.inForceSince}
									In force since {t.inForceSince.toLocaleString()}: accepted by {t.accepted.toLocaleString()} of
									{t.accounts.toLocaleString()} accounts. The others are asked on their next visit, and their API tokens
									stop working until then.
								{:else}
									Accepted by {t.accepted.toLocaleString()} of {t.accounts.toLocaleString()} accounts (at registration).
								{/if}
							</p>
							{#if t.next}
								<p class="mt-2 text-amber-700 dark:text-amber-300">
									Changing on {t.next.toLocaleString()}: until then everybody sees a notice.
								</p>
							{/if}
							{#if termsOpen}
								<form class="mt-4 flex flex-col gap-3" onsubmit={changeTerms}>
									<p class="text-gray-500 dark:text-gray-400">
										Put the new text into <code>terms.md</code> (or at <code>TERMS_URL</code>) first. From the chosen day,
										every account has to accept the terms again before going on; a later day shows a notice until then.
										Small corrections (typos, the address) need no new acceptance.
									</p>
									<div class="max-w-48">
										<Label for="terms-date" class="mb-2">The new terms apply from</Label>
										<Input id="terms-date" type="date" min={today()} bind:value={termsDate} required />
									</div>
									<div class="flex gap-2">
										<Button size="sm" type="submit" disabled={termsBusy}>
											{termsDate > today() ? 'Announce' : 'Ask everybody now'}
										</Button>
										<Button size="sm" color="alternative" onclick={() => (termsOpen = false)}>Cancel</Button>
									</div>
								</form>
							{:else}
								<div class="mt-4 flex flex-wrap gap-2">
									<Button size="sm" color="light" onclick={openTerms}>Terms changed…</Button>
									{#if t.next}
										<Button size="sm" color="light" class="text-red-600 dark:text-red-400" disabled={termsBusy} onclick={withdrawTerms}>
											Withdraw the change
										</Button>
									{/if}
								</div>
							{/if}
						</div>
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
							['Backups (BACKUP_KEEP)', s.backupKeep ? `the newest ${s.backupKeep} kept${s.lastBackupAt ? `, last ${s.lastBackupAt.toLocaleString()}` : ', none yet'}` : 'off', s.backupKeep ? '' : 'Without them, a broken disk or a mistake loses everything'],
							['Logs (LOG_*)', `${s.logFile || 'stdout'}${s.logFile ? `, deleted after ${s.logMaxAgeDays} days` : ' (kept by journald or docker)'}${s.logAnonymizeIp ? ', IP addresses shortened' : ''}`, ''],
							[s.plans ? 'Free plan (QUOTA_*)' : 'Quotas (QUOTA_*)', quotas(s.quotaLinksPerUser, s.quotaSpacesPerUser, s.quotaCollectionsPerSpace, s.quotaUploadMbPerUser), ''],
							...(s.plans && s.proQuotas
								? [
										['Pro plan (QUOTA_PRO_*)', `${quotas(s.proQuotas.linksPerUser, s.proQuotas.spacesPerUser, s.proQuotas.collectionsPerSpace, s.proQuotas.uploadMbPerUser)}; ${o.grantedPro ?? 0} account${o.grantedPro === 1 ? '' : 's'} made Pro (plus the admins)`, ''],
									]
								: [['Plans (PLANS_ENABLED)', 'off: everybody has the same limits', '']]),
							['Donations (SUPPORT_URL)', s.supportUrl || 'none', ''],
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
						<Button disabled={downloading} onclick={() => backup()}>{downloading ? 'Preparing…' : 'Download backup'}</Button>
					</section>
					<section>
						<h2 class="mb-1 font-semibold">Automatic backups</h2>
						{#if backups?.enabled}
							<p class="mb-3 text-gray-500 dark:text-gray-400">
								Every {backups.intervalHours} hours into <code>{backups.dir}</code>, the newest {backups.keep} are kept.
								{#if backups.nextAt}Next one {backups.nextAt.toLocaleString()}.{/if}
								Copy them to another machine now and then: a disk failure takes them along otherwise.
							</p>
						{:else if backups}
							<p class="mb-3 text-amber-700 dark:text-amber-300">
								Off. Set <code>BACKUP_KEEP</code> (e.g. <code>7</code>) and restart, and Ragbag backs itself up every
								<code>BACKUP_INTERVAL_HOURS</code> (24) into <code>{backups.dir}</code> (<code>BACKUP_DIR</code>).
							</p>
						{/if}
						{#if backups && backups.files.length > 0}
							<ul class="mb-3 divide-y divide-gray-100 rounded-lg border border-gray-200 dark:divide-gray-800 dark:border-gray-800">
								{#each backups.files as file (file.name)}
									<li class="flex items-center gap-3 px-3 py-2">
										<span class="min-w-0 flex-1 truncate">
											{file.createdAt.toLocaleString()}
											<span class="text-xs text-gray-500">· {bytes(file.size)}</span>
										</span>
										<Button size="xs" color="light" disabled={downloading} onclick={() => backup(file.name)}>Download</Button>
									</li>
								{/each}
							</ul>
						{/if}
						<Button size="sm" color="light" disabled={creating} onclick={createBackup}>
							{creating ? 'Backing up…' : 'Back up now'}
						</Button>
						<details class="mt-4">
							<summary class="cursor-pointer text-gray-500 dark:text-gray-400">Or from outside, e.g. by cron</summary>
							<p class="my-2 text-gray-500 dark:text-gray-400">
								<code>ragbag backup FILE</code> writes the same file and only needs <code>DATABASE_URL</code>:
							</p>
							<pre class="overflow-x-auto rounded-lg bg-gray-100 p-3 text-xs dark:bg-gray-950/70"><code
									># systemd service (install.sh), while the service runs
sudo sh -c 'set -a; . /etc/ragbag/ragbag.env; ragbag backup /var/backups/ragbag-$(date +%F).json.gz'

# Docker
docker compose exec -T app ragbag backup &gt; ragbag-$(date +%F).json.gz</code
								></pre>
						</details>
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
