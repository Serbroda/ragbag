<script lang="ts">
	import { AdminApi } from 'ragbag-frontend-sdk';
	import type { AbuseReportDto, ReportCategory, ResolveReportRequestActionEnum } from 'ragbag-frontend-sdk';
	import { Button, Textarea } from 'flowbite-svelte';
	import { apiConfig, errorMessage, shareLink } from '../api/client';
	import { session } from '../stores/spaces.svelte';

	/** Reports of public links: disable the link or dismiss the report; both sides are told by email. */
	let { onblock }: { onblock: (userId: string) => void } = $props();

	const api = new AdminApi(apiConfig());

	const CATEGORIES: Record<ReportCategory, string> = {
		ILLEGAL: 'Illegal content',
		CHILD_SAFETY: 'Child sexual abuse material',
		COPYRIGHT: 'Copyright',
		PERSONAL_DATA: 'Personal data',
		SPAM: 'Spam, scam or malware',
		OTHER: 'Other',
	};

	let reports = $state<AbuseReportDto[]>([]);
	let loading = $state(true);
	let error = $state('');
	let deciding = $state<string | null>(null);
	let note = $state('');
	let busy = $state(false);

	load();

	async function load() {
		loading = true;
		error = '';
		try {
			reports = await api.getReports();
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the reports.');
		} finally {
			loading = false;
		}
	}

	function link(report: AbuseReportDto): string {
		const token = report.currentToken ?? report.shareToken;
		const base = shareLink(token);
		return report.viewedCollectionId && report.viewedCollectionId !== report.collectionId
			? `${base}/${report.viewedCollectionId}`
			: base;
	}

	async function resolve(report: AbuseReportDto, action: ResolveReportRequestActionEnum) {
		busy = true;
		error = '';
		try {
			const updated = await api.resolveReport({
				reportId: report.id,
				resolveReportRequest: { action, note: note.trim() || undefined },
			});
			reports = reports.map((r) => (r.id === updated.id ? updated : r));
			deciding = null;
			note = '';
			if (session.me) {
				session.me = { ...session.me, openReports: reports.filter((r) => !r.resolvedAt).length };
			}
		} catch (err) {
			error = await errorMessage(err, 'Failed to save the decision.');
		} finally {
			busy = false;
		}
	}

	const RESOLUTIONS: Record<string, string> = {
		DISABLE_LINK: 'Link disabled',
		DISMISS: 'Dismissed',
		ENABLE_LINK: 'Link enabled again',
	};
</script>

<p class="text-sm text-gray-500 dark:text-gray-400">
	Anybody can report a public link. Look at it and decide: disabling stops the link for good (the space cannot
	share the collection again until you enable it). The reporter and the person who shared it learn the decision
	and your reasons by email. Resolved reports are deleted after 180 days.
</p>
{#if error}
	<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
{/if}
<ul class="-mx-2 max-h-[55vh] divide-y divide-gray-100 overflow-y-auto dark:divide-gray-800">
	{#each reports as report (report.id)}
		<li class="px-2 py-3 text-sm">
			<div class="flex flex-wrap items-baseline gap-x-2">
				<span class="font-medium text-gray-900 dark:text-white">{CATEGORIES[report.category]}</span>
				<span class="text-xs text-gray-500">{report.createdAt.toLocaleString()}</span>
				{#if report.resolvedAt}
					<span class="rounded-md bg-gray-500/15 px-1.5 py-0.5 text-xs text-gray-600 dark:text-gray-300">
						{RESOLUTIONS[report.resolution ?? ''] ?? 'Resolved'}{report.resolvedBy ? ` by ${report.resolvedBy}` : ''}
					</span>
				{:else}
					<span class="rounded-md bg-amber-500/15 px-1.5 py-0.5 text-xs text-amber-700 dark:text-amber-300">Open</span>
				{/if}
			</div>
			<p class="mt-1 text-gray-600 dark:text-gray-300">
				<a href={link(report)} target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline dark:text-primary-400"
					>{report.collectionName}</a
				>
				{#if report.spaceName}<span class="text-gray-500"> in {report.spaceName}</span>{/if}
				{#if report.sharedBy}<span class="text-gray-500"> · shared by {report.sharedBy}</span>{/if}
				{#if !report.collectionId}<span class="text-gray-500"> · collection deleted</span>
				{:else if report.linkDisabled}<span class="text-red-600 dark:text-red-400"> · link disabled</span>
				{:else if !report.currentToken}<span class="text-gray-500"> · no public link anymore</span>{/if}
			</p>
			<p class="mt-1 whitespace-pre-wrap break-words rounded-md bg-gray-50 p-2 text-gray-700 dark:bg-gray-800 dark:text-gray-200">
				{report.details}
			</p>
			<p class="mt-1 text-xs text-gray-500">
				{#if report.reporterName || report.reporterEmail}
					From {[report.reporterName, report.reporterEmail].filter(Boolean).join(', ')}
				{:else}
					Anonymous
				{/if}
				{#if report.resolutionNote}· Reasons: {report.resolutionNote}{/if}
			</p>

			{#if deciding === report.id}
				<div class="mt-2 flex flex-col gap-2">
					<Textarea
						rows={2}
						class="w-full"
						maxlength={2000}
						placeholder="Reasons for your decision (sent to the reporter and the person who shared it)"
						bind:value={note}
					/>
					<div class="flex flex-wrap justify-end gap-2">
						<Button size="xs" color="light" onclick={() => ((deciding = null), (note = ''))}>Cancel</Button>
						{#if report.linkDisabled}
							<Button size="xs" color="light" disabled={busy} onclick={() => resolve(report, 'ENABLE_LINK')}>
								Enable link again
							</Button>
						{:else}
							<Button size="xs" color="light" disabled={busy} onclick={() => resolve(report, 'DISMISS')}>
								Dismiss
							</Button>
							{#if report.currentToken}
								<Button size="xs" color="red" disabled={busy} onclick={() => resolve(report, 'DISABLE_LINK')}>
									Disable link
								</Button>
							{/if}
						{/if}
					</div>
				</div>
			{:else}
				<div class="mt-2 flex flex-wrap justify-end gap-2">
					{#if report.sharedById && report.sharedById !== session.me?.id}
						<Button size="xs" color="light" onclick={() => onblock(report.sharedById!)}>Block {report.sharedBy}…</Button>
					{/if}
					{#if !report.resolvedAt || report.linkDisabled}
						<Button size="xs" color={report.resolvedAt ? 'light' : 'primary'} onclick={() => ((deciding = report.id), (note = ''))}>
							{report.resolvedAt ? 'Reconsider…' : 'Decide…'}
						</Button>
					{/if}
				</div>
			{/if}
		</li>
	{:else}
		<li class="px-2 py-8 text-center text-sm text-gray-500">{loading ? 'Loading…' : 'No reports.'}</li>
	{/each}
</ul>
