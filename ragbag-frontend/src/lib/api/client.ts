import {AuthenticationApi, Configuration} from "ragbag-frontend-sdk";

const STORAGE_KEY = 'access_token';

let accessToken: string | null = localStorage.getItem(STORAGE_KEY);

function setAccessToken(token: string | null) {
    accessToken = token;
    if (token) {
        localStorage.setItem(STORAGE_KEY, token);
    } else {
        localStorage.removeItem(STORAGE_KEY);
    }
}

export function apiConfig(): Configuration {
    return new Configuration({
        basePath: import.meta.env.VITE_API_URL || "http://localhost:8080/api",
        accessToken: accessToken ?? undefined,
        credentials: 'include',
    });
}

export async function login(username: string, password: string): Promise<void> {
    const api = new AuthenticationApi(new Configuration({
        basePath: import.meta.env.VITE_API_URL || "http://localhost:8080/api",
        credentials: 'include',
    }));
    const response = await api.login({loginRequest: {username, password}});
    setAccessToken(response.accessToken);
}

export async function logout(): Promise<void> {
    const api = new AuthenticationApi(apiConfig());
    await api.logoutAll();
    setAccessToken(null);
}

export async function refreshAccessToken(): Promise<boolean> {
    try {
        const api = new AuthenticationApi(new Configuration({
            basePath: import.meta.env.VITE_API_URL || "http://localhost:8080/api",
            credentials: 'include',
        }));
        const response = await api.refresh({refreshToken: '_cookie_'});
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
