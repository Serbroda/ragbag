import {Configuration} from "./gen";

let authHeader: string | undefined;

export function setBasicAuth(username: string, password: string) {
    authHeader = 'Basic ' + btoa(`${username}:${password}`);
}

export function apiConfig() {
    return new Configuration({
        //basePath: import.meta.env.VITE_API_URL,
        basePath: "http://localhost:8080",
        headers: authHeader
            ? { Authorization: authHeader }
            : {}
    });
}
