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

const LAST_SPACE_KEY = 'last_space';

export function rememberSpace(spaceId: string) {
	localStorage.setItem(LAST_SPACE_KEY, spaceId);
}

export function lastSpaceId(): string | null {
	return localStorage.getItem(LAST_SPACE_KEY);
}
