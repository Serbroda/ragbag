import { createRouter } from 'sv-router';
import { isAuthenticated, rememberPendingAdd, rememberPendingInvite } from './api/client';

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
	// Target of the share menu (installed app) and the bookmarklet: add a link from anywhere
	'/add': {
		'/': () => import('./pages/AddPage.svelte'),
	},
	'/space/:spaceId/trash': {
		'/': () => import('./pages/TrashPage.svelte'),
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
	// Instance admins: overview, accounts, reports, backups
	'/admin': {
		'/': () => import('./pages/AdminPage.svelte'),
	},
	'/admin/:tab': {
		'/': () => import('./pages/AdminPage.svelte'),
	},
	// Personal settings: profile, security, appearance, apps & API, account
	'/settings': {
		'/': () => import('./pages/SettingsPage.svelte'),
	},
	'/settings/:tab': {
		'/': () => import('./pages/SettingsPage.svelte'),
	},
	// Free or Pro (PLANS_ENABLED): usage and limits
	'/plan': {
		'/': () => import('./pages/PlanPage.svelte'),
	},
	// First login of the setup admin: own password, then start fresh or restore a backup
	'/setup': {
		'/': () => import('./pages/SetupPage.svelte'),
	},
	// The terms of use changed: accept them (or leave) before going on
	'/terms': {
		'/': () => import('./pages/TermsPage.svelte'),
	},
	'/invite/:token': {
		'/': () => import('./pages/InvitePage.svelte'),
	},
	'/login': {
		'/': () => import('./pages/LoginPage.svelte'),
		meta: { public: true },
	},
	// Public read-only links to a collection, viewable without an account
	'/share/:token': {
		'/': () => import('./pages/SharePage.svelte'),
		meta: { public: true },
	},
	'/share/:token/:collectionId': {
		'/': () => import('./pages/SharePage.svelte'),
		meta: { public: true },
	},
	'/legal/:page': {
		'/': () => import('./pages/LegalPage.svelte'),
		meta: { public: true },
	},
	'/reset-password/:token': {
		'/': () => import('./pages/ResetPasswordPage.svelte'),
		meta: { public: true },
	},
	'/verify-email/:token': {
		'/': () => import('./pages/VerifyEmailPage.svelte'),
		meta: { public: true },
	},
	// Everything else: an address that leads nowhere (a typo, an old link). Signed-out visitors land on the login
	// like for any page that is not public
	'*notfound': () => import('./pages/NotFoundPage.svelte'),
	hooks: {
		beforeLoad({ meta }) {
			if (!meta.public && !isAuthenticated()) {
				// Resume an opened invite link once the user has signed in
				rememberPendingInvite(window.location.pathname);
				rememberPendingAdd(window.location);
				throw navigate('/login');
			}
		},
	},
}, { base });
