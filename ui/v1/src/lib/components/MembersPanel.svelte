<script lang="ts">
	import type { SpaceMemberDto, SpaceMemberRoleEnum } from 'ragbag-frontend-sdk';
	import { CloseOutline } from 'flowbite-svelte-icons';
	import { tick } from 'svelte';
	import Avatar from './Avatar.svelte';
	import { shownName } from '../utils/format';

	/**
	 * The members of a space on demand: opened from the faces in the dashboard header, over the content instead of
	 * a permanent column like Discord's. Managing them stays in the space settings.
	 */
	let {
		open = $bindable(false),
		members,
		currentUserId,
		onmanage,
	}: {
		open?: boolean;
		members: SpaceMemberDto[];
		currentUserId?: string;
		/** Only for those who may manage members: leads to the space settings */
		onmanage?: () => void;
	} = $props();

	const GROUPS: { role: SpaceMemberRoleEnum; label: string; plural: string }[] = [
		{ role: 'OWNER', label: 'Owner', plural: 'Owners' },
		{ role: 'ADMIN', label: 'Admin', plural: 'Admins' },
		{ role: 'CONTRIBUTOR', label: 'Contributor', plural: 'Contributors' },
		{ role: 'VIEWER', label: 'Viewer', plural: 'Viewers' },
	];
	/** Larger groups show as a grid of faces, names on hover, so 300 viewers stay one glance */
	const GRID_FROM = 13;
	const SEARCH_FROM = 20;

	let query = $state('');
	let search = $state<HTMLInputElement | null>(null);
	let closeButton = $state<HTMLButtonElement | null>(null);

	const filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return members;
		return members.filter((m) => shownName(m).toLowerCase().includes(q) || m.username.toLowerCase().includes(q));
	});
	const groups = $derived(
		GROUPS.map((group) => ({ ...group, people: filtered.filter((m) => m.role === group.role) })).filter(
			(group) => group.people.length > 0,
		),
	);

	$effect(() => {
		if (open) {
			query = '';
			tick().then(() => (search ?? closeButton)?.focus());
		}
	});

	const since = new Intl.DateTimeFormat(undefined, { month: 'short', year: 'numeric' });

	function details(member: SpaceMemberDto): string {
		const links = member.linkCount ?? 0;
		return `${links} link${links === 1 ? '' : 's'} · since ${since.format(member.joinedAt)}`;
	}

	function onkeydown(e: KeyboardEvent) {
		if (open && e.key === 'Escape') open = false;
	}
</script>

<svelte:window {onkeydown} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-[60] bg-gray-950/50" onclick={() => (open = false)}></div>
	<div
		class="members-panel fixed inset-y-0 right-0 z-[61] flex w-full max-w-xs flex-col border-s border-gray-200 bg-white shadow-2xl dark:border-gray-800 dark:bg-gray-900"
		role="dialog"
		aria-modal="true"
		aria-label="Members"
	>
		<div class="flex items-center justify-between gap-2 px-4 pb-2 pt-4">
			<h2 class="font-semibold text-gray-900 dark:text-white">
				Members <span class="ms-1 text-sm font-normal text-gray-500 dark:text-gray-400">{members.length}</span>
			</h2>
			<button
				bind:this={closeButton}
				type="button"
				class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-white"
				aria-label="Close"
				onclick={() => (open = false)}
			>
				<CloseOutline class="h-4 w-4" />
			</button>
		</div>
		{#if members.length >= SEARCH_FROM}
			<div class="px-4 pb-2">
				<input
					bind:this={search}
					bind:value={query}
					type="search"
					placeholder="Search members"
					class="w-full rounded-md border-0 bg-gray-100 px-3 py-2 text-sm text-gray-900 focus:ring-2 focus:ring-primary-500 dark:bg-gray-950 dark:text-white"
				/>
			</div>
		{/if}

		<div class="flex-1 overflow-y-auto px-2 pb-4">
			{#each groups as group (group.role)}
				<h3 class="px-2 pb-1 pt-4 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
					{group.people.length === 1 ? group.label : group.plural} — {group.people.length}
				</h3>
				{#if group.people.length >= GRID_FROM && !query.trim()}
					<ul class="flex flex-wrap gap-1.5 px-2 pt-1">
						{#each group.people as member (member.userId)}
							<li title="{shownName(member)} · {details(member)}">
								<Avatar name={shownName(member)} imageId={member.avatarId} class="h-9 w-9 rounded-full text-xs" />
								<span class="sr-only">{shownName(member)}</span>
							</li>
						{/each}
					</ul>
				{:else}
					<ul>
						{#each group.people as member (member.userId)}
							<li class="flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-800/60">
								<Avatar name={shownName(member)} imageId={member.avatarId} class="h-8 w-8 shrink-0 rounded-full text-xs" />
								<div class="min-w-0">
									<p class="truncate text-sm font-medium text-gray-900 dark:text-white">
										{shownName(member)}{#if member.displayName}<span class="ms-1 text-xs font-normal text-gray-500 dark:text-gray-400">@{member.username}</span>{/if}
										{#if member.userId === currentUserId}<span class="ms-1 text-xs font-normal text-gray-500 dark:text-gray-400">(you)</span>{/if}
									</p>
									<p class="truncate text-xs text-gray-500 dark:text-gray-400">{details(member)}</p>
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			{:else}
				<p class="px-2 py-6 text-center text-sm text-gray-500 dark:text-gray-400">No matches</p>
			{/each}
		</div>

		{#if onmanage}
			<div class="border-t border-gray-200 px-4 py-3 dark:border-gray-800">
				<button
					type="button"
					class="text-sm text-primary-600 hover:underline dark:text-primary-400"
					onclick={() => {
						open = false;
						onmanage?.();
					}}
				>
					Manage members →
				</button>
			</div>
		{/if}
	</div>
{/if}

<style>
	.members-panel {
		animation: slide-in 150ms ease-out;
	}
	@keyframes slide-in {
		from {
			transform: translateX(100%);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.members-panel {
			animation: none;
		}
	}
</style>
