import { untrack } from 'svelte';

/**
 * Personal display settings. They live in localStorage per user and device on purpose: a compact list may suit
 * the phone while the desktop shows cards with thumbnails.
 */
export type LinkView = 'cards' | 'compact' | 'list';

export const LINK_VIEWS: { value: LinkView; label: string; description: string }[] = [
	{ value: 'cards', label: 'Cards', description: 'Cards with preview images' },
	{ value: 'compact', label: 'Compact', description: 'Cards without preview images' },
	{ value: 'list', label: 'List', description: 'One line per link' },
];

export type LinkSort = 'newest' | 'oldest' | 'name' | 'domain';

export const LINK_SORTS: { value: LinkSort; label: string }[] = [
	{ value: 'newest', label: 'Newest first' },
	{ value: 'oldest', label: 'Oldest first' },
	{ value: 'name', label: 'Name A–Z' },
	{ value: 'domain', label: 'Domain' },
];

interface Preferences {
	linkView: LinkView;
	linkSort: LinkSort;
	sidebarWidth: number;
	/** Collections that are expanded in the sidebar tree. */
	expanded: string[];
	/** Folders that are open in the space bar. */
	openFolders: string[];
}

export const SIDEBAR_WIDTH = { min: 200, max: 480, default: 256 };

const DEFAULTS: Preferences = {
	linkView: 'cards',
	linkSort: 'newest',
	sidebarWidth: SIDEBAR_WIDTH.default,
	expanded: [],
	openFolders: [],
};

export const preferences = $state<Preferences>({ ...DEFAULTS });

let storageKey: string | null = null;

/** Loads the preferences of the signed-in user; until then the defaults apply. */
export function loadPreferences(userId: string) {
	storageKey = `preferences:${userId}`;
	let stored: Partial<Preferences> = {};
	try {
		stored = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
	} catch {
		// Corrupt or unavailable storage: fall back to the defaults
	}
	const linkView = LINK_VIEWS.some((view) => view.value === stored.linkView) ? stored.linkView! : DEFAULTS.linkView;
	preferences.linkView = linkView;
	preferences.linkSort = LINK_SORTS.some((sort) => sort.value === stored.linkSort) ? stored.linkSort! : DEFAULTS.linkSort;
	preferences.sidebarWidth =
		typeof stored.sidebarWidth === 'number' ? clampSidebarWidth(stored.sidebarWidth) : DEFAULTS.sidebarWidth;
	const expanded = Array.isArray(stored.expanded) ? stored.expanded.filter((id) => typeof id === 'string') : [];
	// Keeps what was expanded while the user was still loading (the open collection and its parents). Untracked:
	// this runs inside an effect, which must not depend on what it writes.
	preferences.expanded = [...new Set([...expanded, ...untrack(() => preferences.expanded)])];
	preferences.openFolders = Array.isArray(stored.openFolders)
		? stored.openFolders.filter((id) => typeof id === 'string')
		: [];
}

export function setLinkView(view: LinkView) {
	preferences.linkView = view;
	save();
}

export function setLinkSort(sort: LinkSort) {
	preferences.linkSort = sort;
	save();
}

export function clampSidebarWidth(width: number): number {
	return Math.min(SIDEBAR_WIDTH.max, Math.max(SIDEBAR_WIDTH.min, Math.round(width)));
}

/** Applies the width right away; call saveSidebarWidth() once dragging ends. */
export function setSidebarWidth(width: number) {
	preferences.sidebarWidth = clampSidebarWidth(width);
}

export function saveSidebarWidth() {
	save();
}

export function isExpanded(collectionId: string): boolean {
	return preferences.expanded.includes(collectionId);
}

export function setExpanded(collectionId: string, expanded: boolean) {
	if (expanded === isExpanded(collectionId)) return;
	preferences.expanded = expanded
		? [...preferences.expanded, collectionId]
		: preferences.expanded.filter((id) => id !== collectionId);
	save();
}

export function isFolderOpen(folderId: string): boolean {
	return preferences.openFolders.includes(folderId);
}

export function setFolderOpen(folderId: string, open: boolean) {
	if (open === isFolderOpen(folderId)) return;
	preferences.openFolders = open
		? [...preferences.openFolders, folderId]
		: preferences.openFolders.filter((id) => id !== folderId);
	save();
}

function save() {
	if (!storageKey) return;
	try {
		localStorage.setItem(storageKey, JSON.stringify({ ...preferences }));
	} catch {
		// Private mode etc.: the choice still applies until the page is reloaded
	}
}

/** Container classes for a list of BookmarkCards in the current view. */
export function linkListClass(view: LinkView): string {
	switch (view) {
		case 'cards':
			return 'grid gap-4 sm:grid-cols-2 xl:grid-cols-3';
		case 'compact':
			return 'grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4';
		case 'list':
			return 'flex flex-col divide-y divide-gray-200 rounded-xl border border-gray-200 bg-white dark:divide-gray-800 dark:border-gray-800 dark:bg-gray-900';
	}
}
