import { createRouter } from 'sv-router';
import { isAuthenticated, rememberPendingInvite } from './api/client';

declare module 'sv-router' {
	interface RouteMeta {
		public?: boolean;
	}
}

export const { p, navigate, isActive, route } = createRouter({
	'/': {
		'/': () => import('./pages/SpaceRedirect.svelte'),
	},
	'/space/:spaceId': {
		'/': () => import('./pages/HomePage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/space/:spaceId/collections/:collectionId': {
		'/': () => import('./pages/CollectionPage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/space/:spaceId/search': {
		'/': () => import('./pages/SearchPage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/space/:spaceId/settings': {
		'/': () => import('./pages/SpaceSettingsPage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/invite/:token': {
		'/': () => import('./pages/InvitePage.svelte'),
	},
	'/login': {
		'/': () => import('./pages/LoginPage.svelte'),
		meta: { public: true },
	},
	hooks: {
		beforeLoad({ meta }) {
			if (!meta.public && !isAuthenticated()) {
				// Resume an opened invite link once the user has signed in
				rememberPendingInvite(window.location.pathname);
				throw navigate('/login');
			}
		},
	},
});
