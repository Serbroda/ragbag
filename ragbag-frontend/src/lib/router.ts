import { createRouter } from 'sv-router';
import { isAuthenticated } from './api/client';

declare module 'sv-router' {
	interface RouteMeta {
		public?: boolean;
	}
}

export const { p, navigate, isActive, route } = createRouter({
	'/': {
		'/': () => import('./pages/HomePage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/collections/:collectionId': {
		'/': () => import('./pages/CollectionPage.svelte'),
		layout: () => import('./layouts/AuthLayout.svelte'),
	},
	'/login': {
		'/': () => import('./pages/LoginPage.svelte'),
		meta: { public: true },
	},
	hooks: {
		beforeLoad({ meta }) {
			if (!meta.public && !isAuthenticated()) {
				throw navigate('/login');
			}
		},
	},
});
