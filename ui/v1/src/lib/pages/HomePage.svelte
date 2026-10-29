<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { BookmarkDto, DashboardDto, DashboardSectionTypeEnum } from 'ragbag-frontend-sdk';
	import { getContext, untrack } from 'svelte';
	import { flip } from 'svelte/animate';
	import { dndzone, dragHandle, dragHandleZone } from 'svelte-dnd-action';
	import { Alert, Button, Input, Modal } from 'flowbite-svelte';
	import {
		ClockSolid,
		CloseOutline,
		GridPlusOutline,
		LinkOutline,
		PenOutline,
		PlusOutline,
		StarSolid,
		TrashBinOutline,
	} from 'flowbite-svelte-icons';
	import { apiConfig, errorMessage } from '../api/client';
	import { navigate, route } from '../router';
	import LinkTile from '../components/LinkTile.svelte';
	import LinkPickerModal from '../components/LinkPickerModal.svelte';
	import FavoriteButton from '../components/FavoriteButton.svelte';
	import { dashboardApi } from '../stores/dashboard';
	import { session } from '../stores/spaces.svelte';
	import { localId } from '../utils/id';

	const FLIP_MS = 180;

	const openInvite = getContext<(() => void) | undefined>('open-invite');

	const spaceId = $derived(route.params.spaceId ?? '');
	const space = $derived(session.spaces.find((s) => s.id === spaceId) ?? null);

	let dashboard = $state<DashboardDto | null>(null);
	let memberCount = $state<number | null>(null);
	let loading = $state(true);
	let error = $state('');

	$effect(() => {
		const id = spaceId;
		// Only the space id may trigger a reload; load() itself reads and writes the dashboard state
		untrack(() => {
			editing = false;
			dashboard = null;
			if (id) load(id);
		});
	});

	async function load(id: string) {
		loading = dashboard === null;
		error = '';
		try {
			const [loaded, members] = await Promise.all([
				dashboardApi().getDashboard({ spaceId: id }),
				new SpaceApi(apiConfig()).getSpaceMembers({ spaceId: id }),
			]);
			if (id !== spaceId) return;
			dashboard = loaded;
			memberCount = members.length;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load the dashboard.');
		} finally {
			loading = false;
		}
	}

	const greeting = $derived.by(() => {
		const hour = new Date().getHours();
		if (hour < 12) return 'Good morning';
		if (hour < 18) return 'Good afternoon';
		return 'Good evening';
	});

	const TYPE_META: Record<DashboardSectionTypeEnum, { label: string; icon: typeof StarSolid; hint: string }> = {
		LINKS: { label: 'Links', icon: LinkOutline, hint: 'Links you pick and arrange yourself' },
		FAVORITES: { label: 'Favorites', icon: StarSolid, hint: 'Shows your starred links automatically' },
		RECENT: { label: 'Recently added', icon: ClockSolid, hint: 'Shows the newest links of the space automatically' },
	};

	// --- Edit mode ---------------------------------------------------------
	// Draft items get their own ids: the same link may sit in several sections, and the
	// drag-and-drop library needs ids that are unique across all zones.

	type DraftItem = { id: string; bookmark: BookmarkDto };
	type DraftSection = { id: string; title: string; type: DashboardSectionTypeEnum; items: DraftItem[] };

	let editing = $state(false);
	let draft = $state<DraftSection[]>([]);
	let saving = $state(false);
	let saveError = $state('');
	let pickerSection = $state<DraftSection | null>(null);
	let pickerOpen = $state(false);
	let showReset = $state(false);

	const uid = localId;

	function startEditing() {
		if (!dashboard) return;
		draft = dashboard.sections.map((section) => ({
			id: uid(),
			title: section.title,
			type: section.type,
			items: section.type === 'LINKS' ? section.bookmarks.map((bookmark) => ({ id: uid(), bookmark })) : [],
		}));
		saveError = '';
		editing = true;
	}

	function addSection(type: DashboardSectionTypeEnum) {
		const title = type === 'LINKS' ? 'New section' : TYPE_META[type].label;
		draft = [...draft, { id: uid(), title, type, items: [] }];
	}

	function removeSection(sectionId: string) {
		draft = draft.filter((s) => s.id !== sectionId);
	}

	function removeItem(section: DraftSection, itemId: string) {
		section.items = section.items.filter((item) => item.id !== itemId);
	}

	function openPicker(section: DraftSection) {
		pickerSection = section;
		pickerOpen = true;
	}

	function addLinks(bookmarks: BookmarkDto[]) {
		if (!pickerSection) return;
		pickerSection.items = [...pickerSection.items, ...bookmarks.map((bookmark) => ({ id: uid(), bookmark }))];
	}

	async function save() {
		if (draft.some((s) => !s.title.trim())) {
			saveError = 'Every section needs a title.';
			return;
		}
		saving = true;
		saveError = '';
		try {
			dashboard = await dashboardApi().updateDashboard({
				spaceId,
				updateDashboardDto: {
					sections: draft.map((s) => ({
						title: s.title.trim(),
						type: s.type,
						bookmarkIds: s.type === 'LINKS' ? s.items.map((item) => item.bookmark.id) : undefined,
					})),
				},
			});
			editing = false;
		} catch (err) {
			saveError = await errorMessage(err, 'Failed to save the dashboard.');
		} finally {
			saving = false;
		}
	}

	async function resetToDefault() {
		showReset = false;
		try {
			await dashboardApi().resetDashboard({ spaceId });
			editing = false;
			await load(spaceId);
		} catch (err) {
			saveError = await errorMessage(err, 'Failed to reset the dashboard.');
		}
	}
</script>

<div>
	<header class="mb-8 flex flex-wrap items-end justify-between gap-4">
		<div class="min-w-0">
			<p class="text-sm text-gray-500 dark:text-gray-400">
				{greeting}{session.me ? `, ${session.me.username}` : ''} 👋
			</p>
			<h1 class="truncate text-3xl font-bold tracking-tight">{space?.name ?? 'Dashboard'}</h1>
			{#if space?.description}
				<p class="mt-1 text-gray-600 dark:text-gray-300">{space.description}</p>
			{/if}
			<div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-gray-500 dark:text-gray-400">
				{#if memberCount !== null}
					<button
						type="button"
						class="hover:text-gray-700 hover:underline dark:hover:text-gray-200"
						title="Show members in space settings"
						onclick={() => navigate('/space/:spaceId/settings', { params: { spaceId } })}
					>
						{memberCount} member{memberCount === 1 ? '' : 's'}
					</button>
				{/if}
				{#if space?.role}
					<span>You are <span class="font-medium capitalize">{space.role.toLowerCase()}</span></span>
				{/if}
				{#if space?.permissions?.includes('MANAGE_MEMBERS')}
					<button type="button" class="text-primary-600 hover:underline dark:text-primary-400" onclick={() => openInvite?.()}>
						Invite people
					</button>
				{/if}
			</div>
		</div>
		{#if dashboard}
			<div class="flex shrink-0 gap-2">
				{#if editing}
					<Button color="alternative" onclick={() => (editing = false)} disabled={saving}>Cancel</Button>
					<Button onclick={save} disabled={saving}>{saving ? 'Saving…' : 'Save dashboard'}</Button>
				{:else}
					<Button color="alternative" onclick={startEditing}>
						<PenOutline class="me-2 h-4 w-4" /> Customize
					</Button>
				{/if}
			</div>
		{/if}
	</header>

	{#if loading}
		<p class="text-gray-500 dark:text-gray-400">Loading...</p>
	{:else if error}
		<Alert color="red">{error}</Alert>
	{:else if dashboard && !editing}
		<!-- View mode -->
		<div class="space-y-10">
			{#each dashboard.sections as section, index (section.id ?? index)}
				{@const meta = TYPE_META[section.type]}
				<section>
					<h2 class="mb-3 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
						<meta.icon class="h-3.5 w-3.5" />
						{section.title}
						<span class="font-normal normal-case tracking-normal text-gray-400">· {section.bookmarks.length}</span>
					</h2>
					{#if section.bookmarks.length === 0}
						<div class="rounded-xl border-2 border-dashed border-gray-200 px-4 py-6 text-center text-sm text-gray-500 dark:border-gray-800">
							{#if section.type === 'FAVORITES'}
								Star links to see them here.
							{:else if section.type === 'RECENT'}
								Nothing has been added to this space yet.
							{:else}
								No links yet. Use
								<button type="button" class="text-primary-600 hover:underline dark:text-primary-400" onclick={startEditing}>
									Customize
								</button>
								to add some.
							{/if}
						</div>
					{:else}
						<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
							{#each section.bookmarks as bookmark (bookmark.id)}
								<LinkTile {bookmark}>
									{#snippet overlay()}
										<FavoriteButton
											bookmarkId={bookmark.id}
											favorite={bookmark.favorite ?? false}
											class={bookmark.favorite ? '' : 'opacity-0 focus:opacity-100 group-hover:opacity-100'}
											onchange={() => load(spaceId)}
										/>
									{/snippet}
								</LinkTile>
							{/each}
						</div>
					{/if}
				</section>
			{/each}
			{#if dashboard.sections.length === 0}
				<div class="rounded-xl border-2 border-dashed border-gray-200 py-14 text-center dark:border-gray-800">
					<p class="font-medium">Your dashboard is empty</p>
					<p class="mt-1 text-sm text-gray-500">Add sections with your most important links.</p>
					<Button class="mt-4" onclick={startEditing}>Customize</Button>
				</div>
			{/if}
		</div>
	{:else if editing}
		<!-- Edit mode: sections are reordered by their handle, links by dragging the tiles -->
		<p class="mb-4 text-sm text-gray-500 dark:text-gray-400">
			Drag sections by their handle and links between sections. Your dashboard is personal, nobody else sees it.
		</p>
		{#if saveError}
			<Alert color="red" class="mb-4">{saveError}</Alert>
		{/if}

		<div
			class="space-y-4"
			use:dragHandleZone={{ items: draft, flipDurationMs: FLIP_MS, type: 'sections', dropTargetStyle: {} }}
			onconsider={(e) => (draft = e.detail.items as DraftSection[])}
			onfinalize={(e) => (draft = e.detail.items as DraftSection[])}
		>
			{#each draft as section (section.id)}
				{@const meta = TYPE_META[section.type]}
				<section
					animate:flip={{ duration: FLIP_MS }}
					class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-gray-800 dark:bg-gray-900/60"
				>
					<div class="mb-3 flex items-center gap-2">
						<div
							use:dragHandle
							aria-label="Drag to reorder section {section.title}"
							class="cursor-grab rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-200"
						>
							<svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
								<circle cx="5" cy="3" r="1.4" /><circle cx="11" cy="3" r="1.4" /><circle cx="5" cy="8" r="1.4" />
								<circle cx="11" cy="8" r="1.4" /><circle cx="5" cy="13" r="1.4" /><circle cx="11" cy="13" r="1.4" />
							</svg>
						</div>
						<Input bind:value={section.title} maxlength={80} class="max-w-xs font-semibold" aria-label="Section title" />
						<span
							class="hidden items-center gap-1 rounded-md bg-gray-200 px-2 py-1 text-xs text-gray-600 sm:inline-flex dark:bg-gray-800 dark:text-gray-300"
							title={meta.hint}
						>
							<meta.icon class="h-3.5 w-3.5" />
							{meta.label}
						</span>
						<button
							type="button"
							class="ms-auto rounded-md p-1.5 text-gray-400 hover:bg-red-500/10 hover:text-red-500"
							title="Remove section"
							aria-label="Remove section {section.title}"
							onclick={() => removeSection(section.id)}
						>
							<TrashBinOutline class="h-4 w-4" />
						</button>
					</div>

					{#if section.type === 'LINKS'}
						<div
							class="grid min-h-16 gap-3 rounded-lg sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4"
							use:dndzone={{
								items: section.items,
								flipDurationMs: FLIP_MS,
								type: 'links',
								dropTargetStyle: { outline: '2px dashed var(--color-primary-500)', outlineOffset: '4px' },
							}}
							onconsider={(e) => (section.items = e.detail.items as DraftItem[])}
							onfinalize={(e) => (section.items = e.detail.items as DraftItem[])}
						>
							{#each section.items as item (item.id)}
								<div animate:flip={{ duration: FLIP_MS }} class="cursor-grab">
									<LinkTile bookmark={item.bookmark}>
										{#snippet overlay()}
											<button
												type="button"
												class="rounded-md p-1 text-gray-400 hover:bg-red-500/10 hover:text-red-500"
												aria-label="Remove {item.bookmark.title || item.bookmark.url}"
												onclick={(e) => {
													e.preventDefault();
													removeItem(section, item.id);
												}}
											>
												<CloseOutline class="h-4 w-4" />
											</button>
										{/snippet}
									</LinkTile>
								</div>
							{/each}
						</div>
						<button
							type="button"
							class="mt-3 flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 py-2.5 text-sm text-gray-500 transition hover:border-primary-500 hover:text-primary-600 dark:border-gray-700 dark:hover:text-primary-400"
							onclick={() => openPicker(section)}
						>
							<PlusOutline class="h-4 w-4" /> Add links
						</button>
					{:else}
						<p class="text-sm text-gray-500 dark:text-gray-400">{meta.hint}.</p>
					{/if}
				</section>
			{/each}
		</div>

		<div class="mt-6 flex flex-wrap items-center gap-2">
			<span class="text-sm text-gray-500 dark:text-gray-400">Add section:</span>
			<Button size="sm" color="alternative" onclick={() => addSection('LINKS')}>
				<GridPlusOutline class="me-1.5 h-4 w-4" /> Links
			</Button>
			<Button size="sm" color="alternative" onclick={() => addSection('FAVORITES')}>
				<StarSolid class="me-1.5 h-4 w-4" /> Favorites
			</Button>
			<Button size="sm" color="alternative" onclick={() => addSection('RECENT')}>
				<ClockSolid class="me-1.5 h-4 w-4" /> Recently added
			</Button>
			{#if dashboard?.customized}
				<button
					type="button"
					class="ms-auto text-sm text-gray-500 hover:text-red-500 hover:underline"
					onclick={() => (showReset = true)}
				>
					Reset to default
				</button>
			{/if}
		</div>
	{/if}
</div>

<LinkPickerModal
	bind:open={pickerOpen}
	{spaceId}
	excludeIds={new Set(pickerSection?.items.map((item) => item.bookmark.id) ?? [])}
	onpick={addLinks}
/>

<Modal title="Reset dashboard" bind:open={showReset} size="sm">
	<p class="text-gray-600 dark:text-gray-300">
		Replace your custom layout with the default (favorites and recently added)? Your links stay in their collections.
	</p>
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			<Button color="alternative" onclick={() => (showReset = false)}>Cancel</Button>
			<Button color="red" onclick={resetToDefault}>Reset</Button>
		</div>
	{/snippet}
</Modal>
