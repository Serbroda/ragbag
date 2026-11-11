<script lang="ts">
	import { PlanApi } from 'ragbag-frontend-sdk';
	import type { PlanDto, QuotasDto } from 'ragbag-frontend-sdk';
	import { ArrowLeftOutline, CheckOutline } from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { p } from '../router';
	import Logo from '../components/Logo.svelte';

	/** The own plan (PLANS_ENABLED): Free or Pro, and what counts against the limits. Pro is granted by the admins. */
	let plan = $state<PlanDto | null>(null);
	let error = $state('');

	new PlanApi(apiConfig())
		.getPlan()
		.then((loaded) => (plan = loaded))
		.catch(async (err) => (error = await errorMessage(err, 'Could not load your plan.')));

	function mb(bytes: number): string {
		return bytes < 1024 * 1024 ? `${Math.round(bytes / 1024)} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	}

	const limit = (value: number, unit = '') => (value > 0 ? `${value.toLocaleString()}${unit}` : 'Unlimited');

	/** The rows of the comparison and the usage bars; rows without any limit are left out. */
	function rows(dto: PlanDto) {
		const all = [
			{ label: 'Own spaces', key: 'spacesPerUser', used: dto.usage.spaces, show: (n: number) => n.toLocaleString() },
			{ label: 'Images', key: 'uploadMbPerUser', used: dto.usage.uploadBytes / 1024 / 1024, unit: ' MB', show: () => mb(dto.usage.uploadBytes) },
			{ label: 'Links', key: 'linksPerUser', used: dto.usage.links, show: (n: number) => n.toLocaleString() },
			{ label: 'Collections per space', key: 'collectionsPerSpace' },
		] as { label: string; key: keyof QuotasDto; used?: number; unit?: string; show?: (n: number) => string }[];
		return all.filter((row) => dto.free[row.key] > 0 || dto.pro[row.key] > 0);
	}

	const current = $derived(plan ? (plan.plan === 'PRO' ? plan.pro : plan.free) : null);
</script>

<svelte:head>
	<title>Plan · Ragbag</title>
</svelte:head>

<div class="min-h-dvh bg-white text-gray-900 dark:bg-gray-850 dark:text-gray-100">
	<header class="sticky top-0 z-20 border-b border-gray-200 bg-white/90 backdrop-blur dark:border-gray-950/60 dark:bg-gray-850/90">
		<div class="mx-auto flex h-12 w-full max-w-3xl items-center gap-3 px-4 md:px-8">
			<a
				href={p('/')}
				class="flex items-center gap-2 rounded-md px-1 py-1 text-sm text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
				title="Back to Ragbag"
			>
				<ArrowLeftOutline class="h-4 w-4" />
				<Logo size={24} />
			</a>
			<h1 class="font-semibold">Plan</h1>
		</div>
	</header>

	<main class="mx-auto flex w-full max-w-3xl flex-col gap-6 px-4 py-6 md:px-8">
		{#if error}
			<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
		{/if}

		{#if plan && current}
			<section class="rounded-lg border border-gray-200 p-4 dark:border-gray-800">
				<div class="flex items-center gap-2">
					<p class="text-lg font-semibold">{plan.plan === 'PRO' ? 'Pro' : 'Free'}</p>
					{#if plan.plan === 'PRO'}
						<span class="rounded-md bg-primary-500/15 px-1.5 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300">
							{plan.proSource === 'ADMIN' ? 'Instance admin' : 'Granted by the admins'}
						</span>
					{/if}
				</div>

				{#if rows(plan).some((row) => row.used !== undefined)}
					<div class="mt-4 flex flex-col gap-3">
						{#each rows(plan).filter((row) => row.used !== undefined) as row (row.key)}
							{@const max = current[row.key]}
							<div>
								<div class="flex justify-between text-sm">
									<span>{row.label}</span>
									<span class="text-gray-500 dark:text-gray-400">
										{row.show?.(row.used ?? 0)}{max > 0 ? ` of ${limit(max, row.unit)}` : ''}
									</span>
								</div>
								{#if max > 0}
									{@const share = Math.min(1, (row.used ?? 0) / max)}
									<div class="mt-1 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-800">
										<div
											class="h-full rounded-full {share >= 1 ? 'bg-red-500' : share > 0.8 ? 'bg-amber-500' : 'bg-primary-500'}"
											style="width: {share * 100}%"
										></div>
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<section>
				<h2 class="mb-2 text-sm font-semibold">Free and Pro</h2>
				<div class="overflow-hidden rounded-lg border border-gray-200 text-sm dark:border-gray-800">
					<div class="grid grid-cols-3 bg-gray-50 px-4 py-2 font-medium dark:bg-gray-900/40">
						<span></span>
						<span>Free</span>
						<span>Pro</span>
					</div>
					{#each rows(plan) as row (row.key)}
						<div class="grid grid-cols-3 border-t border-gray-100 px-4 py-2 dark:border-gray-800">
							<span class="text-gray-500 dark:text-gray-400">{row.label}</span>
							<span>{limit(plan.free[row.key], row.unit)}</span>
							<span>{limit(plan.pro[row.key], row.unit)}</span>
						</div>
					{/each}
					<div class="grid grid-cols-3 border-t border-gray-100 px-4 py-2 dark:border-gray-800">
						<span class="text-gray-500 dark:text-gray-400">API tokens for scripts and apps</span>
						<span>
							{#if plan.freeApiTokens}<CheckOutline class="h-4 w-4" />{:else}<span class="text-gray-400" aria-label="Not included">–</span>{/if}
						</span>
						<span><CheckOutline class="h-4 w-4" /></span>
					</div>
					<div class="grid grid-cols-3 border-t border-gray-100 px-4 py-2 dark:border-gray-800">
						<span class="text-gray-500 dark:text-gray-400">Joining spaces of others</span>
						<span><CheckOutline class="h-4 w-4" /></span>
						<span><CheckOutline class="h-4 w-4" /></span>
					</div>
				</div>
				<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
					Spaces of Pro owners have the Pro limits for everybody in them. Going back to Free keeps everything; you
					just cannot add beyond the Free limits.
					Pro cannot be bought: the administrators of this instance give it out.
				</p>
			</section>
		{:else if !error}
			<p class="text-gray-500 dark:text-gray-400">Loading…</p>
		{/if}

	</main>
</div>
