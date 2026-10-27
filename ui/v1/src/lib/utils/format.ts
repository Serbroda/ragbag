/** Hostname of a URL, or the input itself if it cannot be parsed. */
export function getDomain(url: string): string {
	try {
		return new URL(url).hostname;
	} catch {
		return url;
	}
}

/**
 * Returns the URL only if it uses http(s). Guards against `javascript:` links in data that
 * predates server-side validation.
 */
export function safeUrl(url: string | undefined | null): string | undefined {
	if (!url) return undefined;
	try {
		const parsed = new URL(url);
		return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? url : undefined;
	} catch {
		return undefined;
	}
}

/** Up to two initials, e.g. "Team Research" → "TR". */
export function initials(name: string): string {
	return name
		.split(/\s+/)
		.filter(Boolean)
		.map((w) => w[0])
		.join('')
		.toUpperCase()
		.slice(0, 2);
}

const AVATAR_COLORS = [
	'bg-indigo-500',
	'bg-emerald-500',
	'bg-amber-500',
	'bg-rose-500',
	'bg-sky-500',
	'bg-violet-500',
	'bg-teal-500',
	'bg-orange-500',
];

/** Stable color class per name, like Discord's default avatars. */
export function colorFor(name: string): string {
	let hash = 0;
	for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) | 0;
	return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

// English to match the rest of the UI
const relativeTime = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
	['year', 365 * 24 * 3600],
	['month', 30 * 24 * 3600],
	['week', 7 * 24 * 3600],
	['day', 24 * 3600],
	['hour', 3600],
	['minute', 60],
];

/** "3 hours ago", "yesterday", "just now". */
export function timeAgo(date: Date | undefined | null): string {
	if (!date) return '';
	const seconds = (date.getTime() - Date.now()) / 1000;
	for (const [unit, size] of UNITS) {
		if (Math.abs(seconds) >= size) return relativeTime.format(Math.round(seconds / size), unit);
	}
	return 'just now';
}
