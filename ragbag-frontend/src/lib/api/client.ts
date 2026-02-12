import { AuthenticationApi, Configuration } from 'ragbag-frontend-sdk';
import type { Middleware } from 'ragbag-frontend-sdk';

const STORAGE_KEY = 'access_token';
const BASE_PATH = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

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
			window.location.href = '/login';
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

export async function logout(): Promise<void> {
	const api = new AuthenticationApi(apiConfig());
	await api.logoutAll();
	setAccessToken(null);
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
