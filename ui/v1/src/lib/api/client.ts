import {
	AdminApi,
	AppApi,
	AuthenticationApi,
	BookmarkApi,
	Configuration,
	ResponseError,
	ShareApi,
	UserApi,
} from 'ragbag-frontend-sdk';
import type {
	BackupSummaryDto,
	BookmarkDto,
	CreateReportRequest,
	GetLegalPagePageEnum,
	HeadlinesDto,
	InvitePreviewDto,
	LegalPageDto,
	Middleware,
	SharedCollectionDto,
} from 'ragbag-frontend-sdk';

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

/** Shows the user's locked collections while set; managed by stores/lock.svelte.ts. */
let unlockToken: string | null = null;

export function setUnlockToken(token: string | null) {
	unlockToken = token;
}

/** Adds the unlock header at request time: API objects are created once and live longer than an unlock. */
function withUnlock(headers: HeadersInit | undefined): HeadersInit | undefined {
	if (!unlockToken) return headers;
	const result = new Headers(headers);
	result.set('X-Ragbag-Unlock', unlockToken);
	return result;
}

const authMiddleware: Middleware = {
	async pre(context) {
		return { url: context.url, init: { ...context.init, headers: withUnlock(context.init.headers) } };
	},
	async post(context) {
		// The setup admin replaces its initial password before anything else
		if (context.response.status === 403 && context.response.headers.get('X-Ragbag-Setup')) {
			if (!window.location.pathname.endsWith('/setup')) window.location.href = `${appBase()}/setup`;
			return;
		}
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
		const headers = new Headers(withUnlock(context.init.headers));
		headers.set('Authorization', `Bearer ${accessToken}`);
		const retryInit: RequestInit = { ...context.init, headers };
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

export async function register(
	username: string, email: string, password: string, acceptTerms = false, inviteToken?: string, displayName?: string,
): Promise<void> {
	const api = new AuthenticationApi(anonConfig());
	await api.register({ registerRequest: { username, email, password, acceptTerms, inviteToken, displayName } });
}

/** An invite opened while signed out: where it leads and whether it can create an account. */
export async function getRegistrationInvite(token: string): Promise<InvitePreviewDto> {
	return new AuthenticationApi(anonConfig()).getRegistrationInvite({ token });
}

/** Imprint, privacy policy or terms of use of this instance; works without being signed in. */
export async function getLegalPage(page: string): Promise<LegalPageDto> {
	return new AppApi(anonConfig()).getLegalPage({ page: page as GetLegalPagePageEnum });
}

/** The password of the last login, kept in memory only: the setup page needs the initial one once more. */
let loginPassword: string | null = null;

export async function login(username: string, password: string): Promise<void> {
	const api = new AuthenticationApi(anonConfig());
	const response = await api.login({ loginRequest: { username, password } });
	setAccessToken(response.accessToken);
	loginPassword = password;
}

export function takeLoginPassword(): string | null {
	const password = loginPassword;
	loginPassword = null;
	return password;
}

/** Replaces the setup admin's initial password and sets its email address; returns new tokens. */
export async function completeSetup(currentPassword: string, newPassword: string, email: string): Promise<void> {
	const response = await new AuthenticationApi(apiConfig()).completeSetup({
		completeSetupRequest: { currentPassword, newPassword, email },
	});
	setAccessToken(response.accessToken);
}

/** Downloads a backup of the whole installation (instance admins). */
export async function downloadBackup(stored?: string): Promise<void> {
	const api = new AdminApi(apiConfig());
	const response = stored ? await api.downloadStoredBackupRaw({ name: stored }) : await api.downloadBackupRaw();
	const blob = await response.value();
	const name =
		/filename="([^"]+)"/.exec(response.raw.headers.get('Content-Disposition') ?? '')?.[1] ?? 'ragbag-backup.json.gz';
	const link = document.createElement('a');
	link.href = URL.createObjectURL(blob);
	link.download = name;
	link.click();
	setTimeout(() => URL.revokeObjectURL(link.href), 10_000);
}

/** Checks (dryRun) or restores a backup; restoring replaces the signed in account too, which is signed out. */
export async function restoreBackup(file: File, dryRun: boolean): Promise<BackupSummaryDto> {
	const summary = await new AdminApi(apiConfig()).restoreBackup({ body: file, dryRun });
	if (!dryRun) setAccessToken(null);
	return summary;
}

/** Changes the password; the server signs out all other devices and hands this one new tokens. */
export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
	const response = await new AuthenticationApi(apiConfig()).changePassword({
		changePasswordRequest: { currentPassword, newPassword },
	});
	setAccessToken(response.accessToken);
}

/** A collection shared by a public link (or one of its sub-collections); works without being signed in. */
export async function getSharedCollection(token: string, collectionId?: string): Promise<SharedCollectionDto> {
	return new ShareApi(anonConfig()).getSharedCollection({ token, collectionId });
}

/** The headlines of a link in a public link; works without being signed in. */
export async function getSharedHeadlines(token: string, bookmarkId: string): Promise<HeadlinesDto> {
	return new ShareApi(anonConfig()).getSharedHeadlines({ token, bookmarkId });
}

/** Reports a public link to the administrators; works without being signed in. */
export async function reportShare(token: string, createReportRequest: CreateReportRequest): Promise<void> {
	await new ShareApi(anonConfig()).reportShare({ token, createReportRequest });
}

/** Deletes the own account for good and forgets the session. */
export async function deleteAccount(password: string): Promise<void> {
	await new UserApi(apiConfig()).deleteAccount({ deleteAccountRequest: { password } });
	setAccessToken(null);
}

/** The public link of a shared collection, e.g. https://ragbag.example/app/share/abc… */
export function shareLink(token: string): string {
	return `${window.location.origin}${appBase()}/share/${token}`;
}

/** Who a password reset link belongs to; works without being signed in. */
export async function getPasswordReset(token: string): Promise<{ valid: boolean; username?: string }> {
	return new AuthenticationApi(anonConfig()).getPasswordReset({ token });
}

export async function resetPassword(token: string, newPassword: string): Promise<void> {
	await new AuthenticationApi(anonConfig()).resetPassword({ token, resetPasswordRequest: { newPassword } });
}

/** Confirms a new account's email address with the link from the mail and signs it in. */
export async function verifyEmail(token: string): Promise<void> {
	const response = await new AuthenticationApi(anonConfig()).verifyEmail({ token });
	setAccessToken(response.accessToken);
}

/** Mails a reset link when the account exists (the server answers the same either way). */
export async function requestPasswordReset(login: string): Promise<void> {
	await new AuthenticationApi(anonConfig()).requestPasswordReset({ requestPasswordResetRequest: { login } });
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

/** URL of an avatar or space icon; images are public and cached forever since every upload gets a new id. */
export function imageUrl(imageId: string): string {
	return `${BASE_PATH}/v1/images/${imageId}`;
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

// --- Pending "add link" (shared from another app or the bookmarklet before signing in) ---

const PENDING_ADD_KEY = 'pending_add';

export function rememberPendingAdd(location: Location) {
	if (location.pathname.endsWith('/add')) sessionStorage.setItem(PENDING_ADD_KEY, location.search);
}

/** The query of the add page to return to after signing in, or null. */
export function takePendingAdd(): string | null {
	const search = sessionStorage.getItem(PENDING_ADD_KEY);
	sessionStorage.removeItem(PENDING_ADD_KEY);
	return search;
}

/** The invite a signed-out visitor opened, without using it up (the login page offers registering with it). */
export function pendingInvite(): string | null {
	return sessionStorage.getItem(PENDING_INVITE_KEY);
}

export function takePendingInvite(): string | null {
	const token = sessionStorage.getItem(PENDING_INVITE_KEY);
	sessionStorage.removeItem(PENDING_INVITE_KEY);
	return token;
}
