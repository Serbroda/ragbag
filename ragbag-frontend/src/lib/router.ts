import { createRouter } from 'sv-router';
import { isAuthenticated, rememberPendingInvite } from './api/client';

declare module 'sv-router' {
	interface RouteMeta {
		public?: boolean;
	}
}

// sv-router strips the base from the path before matching, so a base of '/' would turn '/login' into
// 'login' and match nothing. Only pass a real prefix: '/app' when served by the Spring Boot backend.
const configuredBase = import.meta.env.VITE_APP_BASE?.replace(/\/+$/, '');
const base = configuredBase || (globalThis.location?.pathname?.startsWith('/app') ? '/app' : undefined);

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
	'/space/:spaceId/favorites': {
		'/': () => import('./pages/FavoritesPage.svelte'),
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
}, { base });
