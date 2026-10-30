import type { LinkSort } from '../stores/preferences.svelte';
import { imageUrl } from '../api/client';
import { getDomain, safeUrl } from './format';

/** The fields of a link that sorting and filtering look at (bookmarks and links of a public share). */
interface SortableLink {
	url: string;
	title?: string;
	description?: string;
	tags?: string[];
	createdAt?: Date;
}

const collator = new Intl.Collator(undefined, { sensitivity: 'base', numeric: true });

function name(link: SortableLink): string {
	return link.title || link.url;
}

function time(link: SortableLink): number {
	return link.createdAt?.getTime() ?? 0;
}

function domain(link: SortableLink): string {
	return getDomain(link.url).replace(/^www\./, '');
}

/** A sorted copy; ties keep the newest first. */
export function sortLinks<T extends SortableLink>(links: T[], sort: LinkSort): T[] {
	const compare: Record<LinkSort, (a: T, b: T) => number> = {
		newest: (a, b) => time(b) - time(a),
		oldest: (a, b) => time(a) - time(b),
		name: (a, b) => collator.compare(name(a), name(b)) || time(b) - time(a),
		domain: (a, b) => collator.compare(domain(a), domain(b)) || collator.compare(name(a), name(b)),
	};
	return [...links].sort(compare[sort]);
}

/** Links whose title, URL, description or tags contain the query (case-insensitive). */
export function filterLinks<T extends SortableLink>(links: T[], query: string): T[] {
	const q = query.trim().toLowerCase();
	if (!q) return links;
	return links.filter((link) =>
		[link.title, link.url, link.description, ...(link.tags ?? [])].some((value) => value?.toLowerCase().includes(q)),
	);
}

/** Links that have every one of the tags (case-insensitive). */
export function filterByTags<T extends SortableLink>(links: T[], tags: string[]): T[] {
	if (tags.length === 0) return links;
	const wanted = tags.map((tag) => tag.toLowerCase());
	return links.filter((link) => {
		const own = new Set((link.tags ?? []).map((tag) => tag.toLowerCase()));
		return wanted.every((tag) => own.has(tag));
	});
}

/** What decides a link's preview image (a bookmark or a link of a public share). */
interface ImageSource {
	ogImage?: string;
	customImageId?: string;
	customImageUrl?: string;
	imageHidden?: boolean;
}

/** The preview image to show: an upload, else an own address, else the page's image; none if hidden. */
export function previewImage(link: ImageSource): string | undefined {
	if (link.imageHidden) return undefined;
	if (link.customImageId) return imageUrl(link.customImageId);
	return safeUrl(link.customImageUrl) ?? safeUrl(link.ogImage);
}
