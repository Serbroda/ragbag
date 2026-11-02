import { fetchFeatureFlags } from '../api/client';
import { p } from '../router';

export type LegalPage = 'imprint' | 'privacy' | 'terms';

export interface LegalLink {
	page: LegalPage;
	label: string;
	href: string;
	/** An address elsewhere (IMPRINT_URL & co.), opened in a new tab */
	external: boolean;
}

const LABELS: Record<LegalPage, string> = { imprint: 'Imprint', privacy: 'Privacy', terms: 'Terms' };

let cached: Promise<LegalLink[]> | null = null;

/** The legal pages this instance has (feature flags legal_*), loaded once. */
export function legalLinks(): Promise<LegalLink[]> {
	cached ??= fetchFeatureFlags()
		.then((flags) =>
			(Object.keys(LABELS) as LegalPage[]).flatMap((page) => {
				const link = flags[`legal_${page}`];
				if (!link) return [];
				const external = !link.startsWith('/app/');
				return [{ page, label: LABELS[page], href: external ? link : p('/legal/:page', { params: { page } }), external }];
			}),
		)
		.catch(() => {
			cached = null;
			return [];
		});
	return cached;
}
