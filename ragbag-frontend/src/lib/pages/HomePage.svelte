<script lang="ts">
    import {UserApi} from "ragbag-frontend-sdk";
    import {apiConfig} from "../api/client";
    import type {UserDto} from "ragbag-frontend-sdk/src/models";

    let user = $state<UserDto | null>(null);

    const api = new UserApi(apiConfig());
    async function fetchUser() {
        try {
            user = await api.me();
        } catch (error) {
            console.error("Failed to fetch user:", error);
        }
    }

    fetchUser();
</script>

<main class="mx-auto max-w-4xl p-8">
    <h1 class="mb-4 text-2xl font-bold text-gray-900 dark:text-white">Dashboard</h1>
    {#if user}
        <p class="text-gray-500 dark:text-gray-400">Welcome back, {user.username}! 👋</p>
    {:else}
        <p class="text-gray-500 dark:text-gray-400">Welcome to Ragbag.</p>
    {/if}
</main>
