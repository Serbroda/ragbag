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

interface Preferences {
	linkView: LinkView;
}

const DEFAULTS: Preferences = { linkView: 'cards' };

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
}

export function setLinkView(view: LinkView) {
	preferences.linkView = view;
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
