<script lang="ts">
	import { SpaceApi, UserApi } from 'ragbag-frontend-sdk';
	import type { SpaceBarItemDto, SpaceDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal } from 'flowbite-svelte';
	import { CheckOutline, FolderOpenSolid, FolderPlusOutline, PenOutline, PlusOutline } from 'flowbite-svelte-icons';
	import { apiConfig } from '../api/client';
	import { isFolderOpen, setFolderOpen } from '../stores/preferences.svelte';
	import { loadSpaces, session, setStartSpace } from '../stores/spaces.svelte';
	import { initials } from '../utils/format';
	import { localId } from '../utils/id';
	import Avatar from './Avatar.svelte';
	import Logo from './Logo.svelte';

	/**
	 * The Discord-like bar of spaces: personal order and folders (saved on the server, so they follow the user
	 * across devices), a dot for spaces with new links, drag & drop and a context menu to arrange them.
	 */
	let {
		activeSpaceId,
		onselect,
		onhome,
		oncreate,
		onseen,
	}: {
		activeSpaceId: string | null;
		onselect: (spaceId: string) => void;
		onhome: () => void;
		oncreate: () => void;
		/** A space was marked as read here; the layout refreshes its sidebar. */
		onseen?: (spaceId: string) => void;
	} = $props();

	const userApi = new UserApi(apiConfig());

	// --- Layout ---------------------------------------------------------------

	type Folder = { folderId: string; name: string; spaceIds: string[] };
	type Item = { spaceId: string } | Folder;
	type RailItem = { kind: 'space'; space: SpaceDto } | { kind: 'folder'; folder: Folder; spaces: SpaceDto[] };

	let saved = $state<SpaceBarItemDto[]>([]);

	userApi
		.getSpaceBar()
		.then((bar) => (saved = bar.items))
		.catch(() => {});

	/** The saved layout applied to the current spaces; spaces missing in it come last, in the server's order. */
	const layout = $derived.by((): Item[] => {
		const known = new Set(session.spaces.map((space) => space.id));
		const placed = new Set<string>();
		const take = (id: string) => {
			if (!known.has(id) || placed.has(id)) return false;
			placed.add(id);
			return true;
		};
		const items: Item[] = [];
		for (const item of saved) {
			if (item.spaceId) {
				if (take(item.spaceId)) items.push({ spaceId: item.spaceId });
			} else if (item.folderId) {
				const spaceIds = (item.spaceIds ?? []).filter(take);
				if (spaceIds.length) items.push({ folderId: item.folderId, name: item.name ?? 'Folder', spaceIds });
			}
		}
		for (const space of session.spaces) {
			if (take(space.id)) items.push({ spaceId: space.id });
		}
		return items;
	});

	const railItems = $derived.by((): RailItem[] => {
		const byId = new Map(session.spaces.map((space) => [space.id, space]));
		return layout.map((item) =>
			'folderId' in item
				? { kind: 'folder', folder: item, spaces: item.spaceIds.map((id) => byId.get(id)!) }
				: { kind: 'space', space: byId.get(item.spaceId)! },
		);
	});

	const folders = $derived(layout.filter((item): item is Folder => 'folderId' in item));

	async function save(items: Item[]) {
		saved = items; // optimistic
		try {
			saved = (await userApi.updateSpaceBar({ spaceBarDto: { items } })).items;
		} catch {
			// Keeps the local order for this session
		}
	}

	/** Deep copy of the current layout to change and save. */
	function draft(): Item[] {
		return layout.map((item) => ('folderId' in item ? { ...item, spaceIds: [...item.spaceIds] } : { ...item }));
	}

	function folderOf(items: Item[], spaceId: string): Folder | undefined {
		return items.find((item): item is Folder => 'folderId' in item && item.spaceIds.includes(spaceId));
	}

	/** Index of the top-level item that is or contains the space. */
	function topIndexOf(items: Item[], spaceId: string): number {
		return items.findIndex((item) => ('folderId' in item ? item.spaceIds.includes(spaceId) : item.spaceId === spaceId));
	}

	/** Takes the space out of the layout and returns where it was (top-level index), dropping emptied folders. */
	function detach(items: Item[], spaceId: string): number {
		const index = topIndexOf(items, spaceId);
		const item = items[index];
		if ('folderId' in item) {
			item.spaceIds = item.spaceIds.filter((id) => id !== spaceId);
			if (item.spaceIds.length === 0) items.splice(index, 1);
		} else {
			items.splice(index, 1);
		}
		return index;
	}

	type Target = { before: string } | { beforeFolder: string } | { into: string } | { end: true };

	function moveSpace(spaceId: string, target: Target) {
		if ('before' in target && target.before === spaceId) return;
		const items = draft();
		detach(items, spaceId);
		if ('end' in target) {
			items.push({ spaceId });
		} else if ('into' in target) {
			items.find((item): item is Folder => 'folderId' in item && item.folderId === target.into)?.spaceIds.push(spaceId);
		} else if ('beforeFolder' in target) {
			items.splice(Math.max(0, items.findIndex((item) => 'folderId' in item && item.folderId === target.beforeFolder)), 0, {
				spaceId,
			});
		} else {
			const folder = folderOf(items, target.before);
			if (folder) folder.spaceIds.splice(folder.spaceIds.indexOf(target.before), 0, spaceId);
			else items.splice(topIndexOf(items, target.before), 0, { spaceId });
		}
		save(items);
	}

	function moveFolder(folderId: string, target: Target) {
		const items = draft();
		// Onto itself or one of its own spaces: nothing to do
		const own = items.find((item): item is Folder => 'folderId' in item && item.folderId === folderId);
		if (!own || ('before' in target && own.spaceIds.includes(target.before))) return;
		if (('beforeFolder' in target && target.beforeFolder === folderId) || ('into' in target && target.into === folderId)) {
			return;
		}
		const from = items.findIndex((item) => 'folderId' in item && item.folderId === folderId);
		const [folder] = items.splice(from, 1);
		let to = items.length;
		if ('before' in target) to = topIndexOf(items, target.before);
		else if ('beforeFolder' in target || 'into' in target) {
			const id = 'beforeFolder' in target ? target.beforeFolder : target.into;
			to = items.findIndex((item) => 'folderId' in item && item.folderId === id);
		}
		items.splice(to < 0 ? items.length : to, 0, folder);
		save(items);
	}

	function createFolder(spaceId: string, name: string) {
		const items = draft();
		const at = detach(items, spaceId);
		const folderId = localId();
		items.splice(Math.min(at, items.length), 0, { folderId, name, spaceIds: [spaceId] });
		setFolderOpen(folderId, true);
		save(items);
	}

	function removeFromFolder(spaceId: string) {
		const items = draft();
		const folder = folderOf(items, spaceId)!;
		const at = topIndexOf(items, spaceId);
		detach(items, spaceId);
		// Right below the folder, or in its place if it is gone now
		items.splice(items.includes(folder) ? at + 1 : at, 0, { spaceId });
		save(items);
	}

	function renameFolder(folderId: string, name: string) {
		const items = draft();
		const folder = items.find((item): item is Folder => 'folderId' in item && item.folderId === folderId);
		if (folder) folder.name = name;
		save(items);
	}

	function ungroup(folderId: string) {
		const items = draft();
		const index = items.findIndex((item) => 'folderId' in item && item.folderId === folderId);
		const folder = items[index] as Folder;
		items.splice(index, 1, ...folder.spaceIds.map((spaceId) => ({ spaceId })));
		save(items);
	}

	// --- Scrolling --------------------------------------------------------------

	let scroller = $state<HTMLElement | null>(null);
	let scrolled = $state(false);

	// Many spaces scroll below the logo; keep the open one in view (e.g. after picking it in the quick switcher)
	$effect(() => {
		// Also once the spaces have loaded, which happens after the first render
		if (!activeSpaceId || !scroller || railItems.length === 0) return;
		const id = activeSpaceId;
		requestAnimationFrame(() =>
			scroller?.querySelector(`[data-space-id="${id}"]`)?.scrollIntoView({ block: 'nearest' }),
		);
	});

	// --- Drag & drop ----------------------------------------------------------

	let dragging = $state<{ kind: 'space' | 'folder'; id: string } | null>(null);
	/** Key of the item the dragged one would be dropped on, for the drop marker. */
	let dropKey = $state<string | null>(null);

	function dragStart(kind: 'space' | 'folder', id: string, e: DragEvent) {
		dragging = { kind, id };
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', id);
		}
	}

	function dragEnd() {
		dragging = null;
		dropKey = null;
	}

	function allowDrop(key: string, e: DragEvent) {
		if (!dragging) return;
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		dropKey = key;
	}

	function drop(target: Target, e: DragEvent) {
		e.preventDefault();
		const dragged = dragging;
		dragEnd();
		if (!dragged) return;
		if (dragged.kind === 'space') moveSpace(dragged.id, target);
		else moveFolder(dragged.id, target);
	}

	/** On a folder: the upper part drops before it, the rest into it (folders always go before it). */
	function folderTarget(folderId: string, e: DragEvent): Target {
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const upper = e.clientY - rect.top < rect.height * 0.3;
		return dragging?.kind === 'folder' || upper ? { beforeFolder: folderId } : { into: folderId };
	}

	// --- Context menu ---------------------------------------------------------

	let menu = $state<{ x: number; y: number; spaceId?: string; folderId?: string } | null>(null);

	function openMenu(e: MouseEvent, target: { spaceId?: string; folderId?: string }) {
		e.preventDefault();
		e.stopPropagation();
		menu = { x: Math.min(e.clientX, window.innerWidth - 232), y: Math.min(e.clientY, window.innerHeight - 260), ...target };
	}

	const menuSpace = $derived(menu?.spaceId ? session.spaces.find((space) => space.id === menu!.spaceId) : undefined);
	const menuSpaceFolder = $derived(menu?.spaceId ? folderOf(layout, menu.spaceId) : undefined);
	const menuFolder = $derived(menu?.folderId ? folders.find((folder) => folder.folderId === menu!.folderId) : undefined);

	async function markSeen(spaceId: string) {
		menu = null;
		await new SpaceApi(apiConfig()).markSpaceSeen({ spaceId }).catch(() => {});
		await loadSpaces();
		onseen?.(spaceId);
	}

	// --- Folder name dialog ---------------------------------------------------

	let nameDialog = $state<{ mode: 'create'; spaceId: string } | { mode: 'rename'; folderId: string } | null>(null);
	let folderName = $state('');

	function askName(dialog: NonNullable<typeof nameDialog>, current = '') {
		menu = null;
		folderName = current;
		nameDialog = dialog;
	}

	function submitName() {
		const name = folderName.trim().slice(0, 40);
		if (!name || !nameDialog) return;
		if (nameDialog.mode === 'create') createFolder(nameDialog.spaceId, name);
		else renameFolder(nameDialog.folderId, name);
		nameDialog = null;
	}

	function newIn(spaces: SpaceDto[]): number {
		return spaces.reduce((sum, space) => sum + (space.newCount ?? 0), 0);
	}
</script>

<svelte:window onclick={() => (menu = null)} oncontextmenu={() => (menu = null)} />

{#snippet indicator(active: boolean, unread: boolean)}
	<!-- Discord's pill: tall for the open space, a dot for news, a short bar on hover -->
	<span
		class="absolute left-0 w-1 rounded-r-full bg-white transition-all duration-200 {active
			? 'h-10'
			: unread
				? 'h-2 group-hover:h-5'
				: 'h-0 group-hover:h-5'}"
	></span>
{/snippet}

{#snippet tooltip(text: string)}
	<span
		class="pointer-events-none fixed left-[80px] z-50 hidden whitespace-nowrap rounded-md bg-gray-950 px-3 py-1.5 text-sm font-semibold text-white shadow-xl ring-1 ring-white/10 md:group-hover:block"
	>
		{text}
	</span>
{/snippet}

{#snippet dropMarker(key: string)}
	{#if dropKey === key}
		<span class="pointer-events-none absolute -top-1.5 left-3 right-3 h-1 rounded-full bg-primary-500"></span>
	{/if}
{/snippet}

{#snippet spaceButton(space: SpaceDto)}
	{@const active = activeSpaceId === space.id}
	{@const unread = (space.newCount ?? 0) > 0}
	<div
		class="group relative flex w-full items-center justify-center"
		class:opacity-40={dragging?.kind === 'space' && dragging.id === space.id}
	>
		{@render indicator(active, unread)}
		{@render dropMarker(`space:${space.id}`)}
		<button
			type="button"
			draggable="true"
			aria-label={unread ? `${space.name}, ${space.newCount} new links` : space.name}
			aria-current={active ? 'page' : undefined}
			data-space-id={space.id}
			class="relative h-12 w-12 shrink-0 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/40 {active
				? 'rounded-2xl shadow-lg'
				: 'rounded-3xl hover:rounded-2xl'}"
			onclick={() => onselect(space.id)}
			oncontextmenu={(e) => openMenu(e, { spaceId: space.id })}
			ondragstart={(e) => dragStart('space', space.id, e)}
			ondragend={dragEnd}
			ondragover={(e) => allowDrop(`space:${space.id}`, e)}
			ondragleave={() => (dropKey = null)}
			ondrop={(e) => drop({ before: space.id }, e)}
		>
			<span
				class="block h-full w-full overflow-hidden transition-[filter,box-shadow] duration-200 group-hover:brightness-110 {active
					? 'rounded-2xl'
					: 'rounded-[inherit]'} {space.iconId
					? 'ring-1 ring-white/15 group-hover:ring-white/30'
					: ''}"
			>
				{#if space.iconId || active}
					<Avatar name={space.name} imageId={space.iconId} class="h-full w-full text-sm" />
				{:else}
					<span
						class="flex h-full w-full items-center justify-center bg-gray-800 text-sm font-semibold text-gray-300 transition-colors hover:bg-primary-500 hover:text-white"
					>
						{initials(space.name)}
					</span>
				{/if}
			</span>
			{#if unread && !active}
				<span
					class="absolute -bottom-1 -right-1 min-w-5 rounded-full bg-primary-500 px-1 text-center text-[11px] font-bold leading-5 text-white ring-4 ring-gray-950"
				>
					{(space.newCount ?? 0) > 99 ? '99+' : space.newCount}
				</span>
			{/if}
		</button>
		{@render tooltip(space.name)}
	</div>
{/snippet}

<nav class="flex w-[72px] shrink-0 flex-col bg-gray-950" aria-label="Spaces">
	<!-- The logo stays put, the spaces below it scroll -->
	<div
		class="relative z-10 flex shrink-0 flex-col items-center gap-2 pb-2 pt-3 transition-shadow {scrolled
			? 'shadow-[0_6px_8px_-4px_rgba(0,0,0,0.6)]'
			: ''}"
	>
		<button
			type="button"
			title="Ragbag"
			aria-label="Ragbag home"
			class="transition-transform duration-200 hover:scale-105"
			onclick={onhome}
		>
			<Logo size={48} />
		</button>
		<div class="h-0.5 w-8 rounded bg-gray-800"></div>
	</div>

	<div
		bind:this={scroller}
		class="flex min-h-0 flex-1 flex-col items-center gap-2 overflow-y-auto overflow-x-hidden pb-3 pt-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
		onscroll={() => (scrolled = (scroller?.scrollTop ?? 0) > 0)}
	>

	{#each railItems as item (item.kind === 'space' ? item.space.id : item.folder.folderId)}
		{#if item.kind === 'space'}
			{@render spaceButton(item.space)}
		{:else}
			{@const folder = item.folder}
			{@const open = isFolderOpen(folder.folderId)}
			{@const containsActive = item.spaces.some((space) => space.id === activeSpaceId)}
			{@const unread = newIn(item.spaces)}
			<div class="flex w-full flex-col items-center gap-2" class:opacity-40={dragging?.kind === 'folder' && dragging.id === folder.folderId}>
				<div class="group relative flex w-full items-center justify-center">
					{@render indicator(!open && containsActive, !open && unread > 0)}
					{@render dropMarker(`folder:${folder.folderId}`)}
					<button
						type="button"
						draggable="true"
						aria-label="Folder {folder.name}"
						aria-expanded={open}
						class="relative h-12 w-12 shrink-0 rounded-2xl transition-colors {open
							? 'text-primary-400 hover:bg-gray-800'
							: 'bg-gray-800/80 p-1.5 hover:bg-gray-800'} {dropKey === `into:${folder.folderId}`
							? 'ring-2 ring-primary-500'
							: ''}"
						onclick={() => setFolderOpen(folder.folderId, !open)}
						oncontextmenu={(e) => openMenu(e, { folderId: folder.folderId })}
						ondragstart={(e) => dragStart('folder', folder.folderId, e)}
						ondragend={dragEnd}
						ondragover={(e) => {
							const target = folderTarget(folder.folderId, e);
							allowDrop('into' in target ? `into:${folder.folderId}` : `folder:${folder.folderId}`, e);
						}}
						ondragleave={() => (dropKey = null)}
						ondrop={(e) => drop(folderTarget(folder.folderId, e), e)}
					>
						{#if open}
							<FolderOpenSolid class="mx-auto h-6 w-6" />
						{:else}
							<span class="grid h-full w-full grid-cols-2 grid-rows-2 gap-1">
								{#each item.spaces.slice(0, 4) as space (space.id)}
									<Avatar
										name={space.name}
										imageId={space.iconId}
										class="h-full w-full rounded-md text-[8px] {space.iconId ? 'ring-1 ring-white/15' : ''}"
									/>
								{/each}
							</span>
							{#if unread > 0}
								<span
									class="absolute -bottom-1 -right-1 min-w-5 rounded-full bg-primary-500 px-1 text-center text-[11px] font-bold leading-5 text-white ring-4 ring-gray-950"
								>
									{unread > 99 ? '99+' : unread}
								</span>
							{/if}
						{/if}
					</button>
					{@render tooltip(folder.name)}
				</div>
				{#if open}
					<div class="flex w-full flex-col items-center gap-2 rounded-2xl bg-gray-900/80 py-2">
						{#each item.spaces as space (space.id)}
							{@render spaceButton(space)}
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	{/each}

	<div class="group relative flex w-full items-center justify-center">
		{@render dropMarker('end')}
		<button
			type="button"
			aria-label="Create or join a space"
			class="flex h-12 w-12 items-center justify-center rounded-3xl bg-gray-800 text-green-400 transition-all duration-200 hover:rounded-2xl hover:bg-green-600 hover:text-white"
			onclick={oncreate}
			ondragover={(e) => allowDrop('end', e)}
			ondragleave={() => (dropKey = null)}
			ondrop={(e) => drop({ end: true }, e)}
		>
			<PlusOutline class="h-6 w-6" />
		</button>
		{@render tooltip('Create or join a space')}
	</div>
	</div>
</nav>

{#if menu}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div
		role="menu"
		tabindex="-1"
		class="fixed z-50 w-56 rounded-lg bg-gray-950 p-1.5 text-sm text-gray-200 shadow-2xl ring-1 ring-white/10"
		style:left="{menu.x}px"
		style:top="{menu.y}px"
		onclick={(e) => e.stopPropagation()}
	>
		{#if menuSpace}
			{@const space = menuSpace}
			{@const isStart = session.me?.startSpaceId === space.id}
			<p class="truncate px-2 pb-1 pt-0.5 text-xs font-semibold text-gray-400">{space.name}</p>
			<button
				role="menuitemcheckbox"
				aria-checked={isStart}
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				title="Ragbag opens with this space, on every device"
				onclick={() => {
					// Read before closing: the menu's space is gone once it is closed
					const spaceId = isStart ? null : space.id;
					menu = null;
					setStartSpace(spaceId);
				}}
			>
				Open on start
				{#if isStart}<CheckOutline class="h-4 w-4" />{/if}
			</button>
			<div class="my-1 h-px bg-gray-800"></div>
			{#if (space.newCount ?? 0) > 0}
				<button
					role="menuitem"
					class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
					onclick={() => markSeen(space.id)}
				>
					Mark as read <CheckOutline class="h-4 w-4" />
				</button>
				<div class="my-1 h-px bg-gray-800"></div>
			{/if}
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => askName({ mode: 'create', spaceId: space.id })}
			>
				New folder… <FolderPlusOutline class="h-4 w-4" />
			</button>
			{#each folders.filter((folder) => folder !== menuSpaceFolder) as folder (folder.folderId)}
				<button
					role="menuitem"
					class="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
					onclick={() => ((menu = null), moveSpace(space.id, { into: folder.folderId }))}
				>
					<span class="truncate">Move to {folder.name}</span>
				</button>
			{/each}
			{#if menuSpaceFolder}
				<button
					role="menuitem"
					class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
					onclick={() => ((menu = null), removeFromFolder(space.id))}
				>
					Remove from folder
				</button>
			{/if}
		{:else if menuFolder}
			{@const folder = menuFolder}
			<p class="truncate px-2 pb-1 pt-0.5 text-xs font-semibold text-gray-400">{folder.name}</p>
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => askName({ mode: 'rename', folderId: folder.folderId }, folder.name)}
			>
				Rename folder… <PenOutline class="h-4 w-4" />
			</button>
			<button
				role="menuitem"
				class="flex w-full items-center justify-between rounded-md px-2 py-1.5 hover:bg-primary-500 hover:text-white"
				onclick={() => ((menu = null), ungroup(folder.folderId))}
			>
				Ungroup spaces
			</button>
		{/if}
	</div>
{/if}

<Modal
	title={nameDialog?.mode === 'rename' ? 'Rename folder' : 'New folder'}
	open={nameDialog !== null}
	size="xs"
	autoclose={false}
	onclose={() => (nameDialog = null)}
>
	<form
		class="flex flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();
			submitName();
		}}
	>
		<div>
			<Label for="folder-name" class="mb-2">Name</Label>
			<Input id="folder-name" bind:value={folderName} maxlength={40} required placeholder="e.g. Family" />
		</div>
		<p class="text-xs text-gray-500 dark:text-gray-400">Folders and the order of your spaces are only for you.</p>
		<div class="flex justify-end gap-2">
			<Button color="light" onclick={() => (nameDialog = null)}>Cancel</Button>
			<Button type="submit" disabled={!folderName.trim()}>{nameDialog?.mode === 'rename' ? 'Save' : 'Create'}</Button>
		</div>
	</form>
</Modal>
