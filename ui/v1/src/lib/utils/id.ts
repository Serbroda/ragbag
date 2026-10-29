/**
 * Random id for client-side bookkeeping (dashboard drafts, space bar folders). crypto.randomUUID() only exists in
 * secure contexts (HTTPS or localhost), so self-hosted instances on plain http://192.168.x.x would fail with it;
 * crypto.getRandomValues() works everywhere.
 */
export function localId(): string {
	const bytes = crypto.getRandomValues(new Uint8Array(12));
	return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}
