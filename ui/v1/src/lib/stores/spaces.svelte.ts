import { SpaceApi, UserApi } from 'ragbag-frontend-sdk';
import type { SpaceDto, UserDto } from 'ragbag-frontend-sdk';
import { apiConfig } from '../api/client';

/** Spaces of the current user, shared by the layout, the start page and the settings page. */
export const session = $state<{ spaces: SpaceDto[]; loaded: boolean; me: UserDto | null }>({
	spaces: [],
	loaded: false,
	me: null,
});

let loading: Promise<void> | null = null;

export function loadSpaces(): Promise<void> {
	loading ??= new SpaceApi(apiConfig())
		.getSpaces()
		.then((spaces) => {
			session.spaces = spaces;
			session.loaded = true;
		})
		.finally(() => {
			loading = null;
		});
	return loading;
}

export async function loadMe(): Promise<UserDto> {
	session.me ??= await new UserApi(apiConfig()).me();
	return session.me;
}

export function resetSession() {
	session.spaces = [];
	session.loaded = false;
	session.me = null;
}

// --- Last visited space (Discord reopens the server you were in) ---

/** Per user: another account in the same browser has spaces of its own. */
function lastSpaceKey(): string | null {
	return session.me ? `last_space:${session.me.id}` : null;
}

export function rememberSpace(spaceId: string) {
	const key = lastSpaceKey();
	if (!key) return;
	try {
		localStorage.setItem(key, spaceId);
	} catch {
		// Private mode etc.: the start falls back to the first space
	}
}

export function lastSpaceId(): string | null {
	const key = lastSpaceKey();
	try {
		// "last_space" is where versions before 0.12 kept it for everybody
		return (key && localStorage.getItem(key)) || localStorage.getItem('last_space');
	} catch {
		return null;
	}
}

/** Chooses the space opened on start (null: the last visited one). */
export async function setStartSpace(spaceId: string | null): Promise<void> {
	// Without spaceId the server opens the last visited space again
	session.me = await new UserApi(apiConfig()).setStartSpace({ startSpaceDto: { spaceId: spaceId ?? undefined } });
}

/**
 * The space to open on start: the chosen start space, else the last visited one, else the first in the user's
 * own space bar (spaceBarOrder), else the first one at all. Spaces the user left or that were deleted are skipped.
 */
export function startSpaceId(spaceBarOrder: string[] = []): string | undefined {
	const member = (id: string | null | undefined) => !!id && session.spaces.some((space) => space.id === id);
	const chosen = session.me?.startSpaceId;
	if (member(chosen)) return chosen!;
	const last = lastSpaceId();
	if (member(last)) return last!;
	return spaceBarOrder.find(member) ?? session.spaces[0]?.id;
}
