import { DashboardApi } from 'ragbag-frontend-sdk';
import type { DashboardDto, UpdateDashboardSectionDto } from 'ragbag-frontend-sdk';
import { apiConfig } from '../api/client';

export function dashboardApi(): DashboardApi {
	return new DashboardApi(apiConfig());
}

/** Converts a loaded dashboard back into the layout the API stores. */
export function toLayout(dashboard: DashboardDto): UpdateDashboardSectionDto[] {
	return dashboard.sections.map((section) => ({
		title: section.title,
		type: section.type,
		bookmarkIds: section.type === 'LINKS' ? section.bookmarks.map((b) => b.id) : undefined,
	}));
}

/**
 * Pins a bookmark to a LINKS section of the user's dashboard. `target` is the index of an existing
 * section or the title of a new one that is appended.
 */
export async function pinToDashboard(spaceId: string, bookmarkId: string, target: number | string): Promise<void> {
	const api = dashboardApi();
	const layout = toLayout(await api.getDashboard({ spaceId }));
	if (typeof target === 'number') {
		const section = layout[target];
		if (!section || section.type !== 'LINKS') throw new Error('Not a links section');
		if (!section.bookmarkIds?.includes(bookmarkId)) {
			section.bookmarkIds = [...(section.bookmarkIds ?? []), bookmarkId];
		}
	} else {
		layout.push({ title: target, type: 'LINKS', bookmarkIds: [bookmarkId] });
	}
	await api.updateDashboard({ spaceId, updateDashboardDto: { sections: layout } });
}
