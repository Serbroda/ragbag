import { AppApi, AuthenticationApi, BookmarkApi, Configuration, ResponseError } from 'ragbag-frontend-sdk';
import type { BookmarkDto, Middleware } from 'ragbag-frontend-sdk';

const STORAGE_KEY = 'access_token';
// Production builds are served by the backend itself, so the API lives on the same origin
const BASE_PATH = import.meta.env.VITE_API_URL || (import.meta.env.DEV ? 'http://localhost:8080/api' : '/api');

let accessToken: string | null = localStorage.getItem(STORAGE_KEY);

function setAccessToken(token: string | null) {
	accessToken = token;
	if (token) {
		localStorage.setItem(STORAGE_KEY, token);
	} else {
		localStorage.removeItem(STORAGE_KEY);
	}
}

// --- 401 Interceptor ---

let refreshPromise: Promise<boolean> | null = null;

const authMiddleware: Middleware = {
	async post(context) {
		if (context.response.status !== 401) return;

		// Don't retry auth endpoints (login, register, refresh)
		if (context.url.includes('/auth/')) return;

		// Deduplicate concurrent refresh attempts
		if (!refreshPromise) {
			refreshPromise = refreshAccessToken().finally(() => {
				refreshPromise = null;
			});
		}

		const success = await refreshPromise;
		if (!success) {
			setAccessToken(null);
			window.location.href = `${appBase()}/login`;
			return;
		}

		// Retry original request with new token (raw fetch to avoid middleware loop)
		const retryInit: RequestInit = {
			...context.init,
			headers: {
				...context.init.headers,
				Authorization: `Bearer ${accessToken}`,
			},
		};
		return fetch(context.url, retryInit);
	},
};

// --- Config & Auth ---

export function apiConfig(): Configuration {
	return new Configuration({
		basePath: BASE_PATH,
		accessToken: accessToken ?? undefined,
		credentials: 'include',
		middleware: [authMiddleware],
	});
}

/** Unauthenticated config for login/register/refresh (no middleware). */
function anonConfig(): Configuration {
	return new Configuration({
		basePath: BASE_PATH,
		credentials: 'include',
	});
}

export async function register(username: string, email: string, password: string): Promise<void> {
	const api = new AuthenticationApi(anonConfig());
	await api.register({ registerRequest: { username, email, password } });
}

export async function login(username: string, password: string): Promise<void> {
	const api = new AuthenticationApi(anonConfig());
	const response = await api.login({ loginRequest: { username, password } });
	setAccessToken(response.accessToken);
}

/** Signs out on this device only. */
export async function logout(): Promise<void> {
	try {
		await new AuthenticationApi(anonConfig()).logout();
	} finally {
		setAccessToken(null);
	}
}

/** Invalidates all sessions of the user on every device. */
export async function logoutEverywhere(): Promise<void> {
	try {
		await new AuthenticationApi(apiConfig()).logoutAll();
	} finally {
		setAccessToken(null);
	}
}

export async function refreshAccessToken(): Promise<boolean> {
	try {
		const api = new AuthenticationApi(anonConfig());
		const response = await api.refresh({ refreshToken: '_cookie_' });
		setAccessToken(response.accessToken);
		return true;
	} catch {
		setAccessToken(null);
		return false;
	}
}

export function isAuthenticated(): boolean {
	return accessToken !== null;
}

export async function fetchFeatureFlags(): Promise<{ [key: string]: string }> {
	const api = new AppApi(anonConfig());
	return api.featureFlags();
}

export async function searchBookmarks(spaceId: string, query: string): Promise<BookmarkDto[]> {
	const api = new BookmarkApi(apiConfig());
	return api.searchBookmarks({ spaceId, q: query });
}

/** Path prefix of the SPA: `/app` when served by the Spring Boot backend, empty in the Vite dev server. */
export function appBase(): string {
	return window.location.pathname.startsWith('/app') ? '/app' : '';
}

/** Extracts the `detail` of a ProblemDetail error response, falling back to the given message. */
export async function errorMessage(err: unknown, fallback: string): Promise<string> {
	if (err instanceof ResponseError) {
		try {
			const body = await err.response.json();
			if (typeof body?.detail === 'string' && body.detail) return body.detail;
		} catch {
			// Body was empty or not JSON
		}
	}
	return fallback;
}

// --- Pending invite (survives the login redirect) ---

const PENDING_INVITE_KEY = 'pending_invite';

export function rememberPendingInvite(pathname: string) {
	const match = pathname.match(/\/invite\/([^/]+)/);
	if (match) sessionStorage.setItem(PENDING_INVITE_KEY, match[1]);
}

export function takePendingInvite(): string | null {
	const token = sessionStorage.getItem(PENDING_INVITE_KEY);
	sessionStorage.removeItem(PENDING_INVITE_KEY);
	return token;
}
