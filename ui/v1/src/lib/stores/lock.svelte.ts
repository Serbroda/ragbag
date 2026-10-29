import { UserApi } from 'ragbag-frontend-sdk';
import { apiConfig, setUnlockToken } from '../api/client';

/**
 * Unlocking shows the collections the user locked for themselves. The server's token lives a few minutes and is
 * renewed while the user is active, so an idle or closed tab locks again. It is kept in sessionStorage: a reload
 * keeps it, a new tab or a restart of the browser does not.
 */
export const lockState = $state<{ unlocked: boolean; version: number }>({ unlocked: false, version: 0 });

const STORAGE_KEY = 'unlock';
/** Renew when less than this is left; the server grants 10 minutes. */
const RENEW_BELOW_MS = 8 * 60 * 1000;

let token: string | null = null;
let expiresAt = 0;
let renewing = false;

function apply(next: { token: string; expiresAt: number } | null) {
	token = next?.token ?? null;
	expiresAt = next?.expiresAt ?? 0;
	setUnlockToken(token);
	try {
		if (next) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next));
		else sessionStorage.removeItem(STORAGE_KEY);
	} catch {
		// Without sessionStorage the unlock only lasts until the page is reloaded
	}
	if (lockState.unlocked !== !!next) {
		lockState.unlocked = !!next;
		// Pages reload their data, so locked links appear or disappear everywhere
		lockState.version++;
	}
}

// Resume an unlock from before a reload, unless it expired meanwhile
try {
	const stored = JSON.parse(sessionStorage.getItem(STORAGE_KEY) ?? 'null');
	if (stored?.token && stored.expiresAt > Date.now()) apply(stored);
	else sessionStorage.removeItem(STORAGE_KEY);
} catch {
	// Corrupt or unavailable storage: stay locked
}

export async function unlock(pin: string): Promise<void> {
	const result = await new UserApi(apiConfig()).unlock({ unlockRequest: { pin } });
	apply({ token: result.token, expiresAt: result.expiresAt.getTime() });
}

export function lock() {
	apply(null);
}

/** Called on user activity: keeps an unlock alive while the user is around. */
export async function keepUnlocked() {
	if (!token || renewing) return;
	if (Date.now() >= expiresAt) return lock();
	if (expiresAt - Date.now() > RENEW_BELOW_MS) return;
	renewing = true;
	try {
		const result = await new UserApi(apiConfig()).renewUnlock();
		if (token) apply({ token: result.token, expiresAt: result.expiresAt.getTime() });
	} catch {
		lock();
	} finally {
		renewing = false;
	}
}

/** Locks once the token ran out (the user was idle). */
export function checkExpiry() {
	if (token && Date.now() >= expiresAt) lock();
}

export function resetLock() {
	apply(null);
}
