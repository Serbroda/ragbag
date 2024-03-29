<script lang="ts">
    import active from "svelte-spa-router/active";
    import { replace } from "svelte-spa-router";
    import logo from "../assets/logo.svg";
    import { groups, groupSubscriptions } from "../stores/groups";
    import { Icon, Home, User } from "svelte-hero-icons";
    import { authService } from "../services/Services";
    import { toggleTheme } from "../App.svelte";
    import InfoDropdown from "./components/InfoDropdown.svelte";

    export let onCreateGroupClick: () => void;
    export let onCreateGroupSubscriptionClick: () => void;
</script>

<div class="drawer-side h-full">
    <label for="main-menu" class="drawer-overlay" />
    <aside class="flex flex-col bg-base-200 text-base-content w-80 h-full">
        <div
            class="sticky inset-x-0 top-0 z-10 w-full py-1 transition duration-200 ease-in-out border-b border-base-200 bg-base-200">
            <div class="navbar">
                <div class="navbar-start">
                    <a href="/" class="px-2 flex-0 btn btn-ghost md:px-4" aria-label="Homepage">
                        <div class="inline-block text-3xl font-title text-primary flex">
                            <img src={logo} class="w-9 h-9 mr-3" alt="Logo" />
                            <span class="lowercase">rag</span><span class="uppercase text-base-content">bag</span>
                        </div>
                    </a>
                </div>

                <div class="navbar-end">
                    <div class="dropdown dropdown-end">
                        <!-- svelte-ignore a11y-label-has-associated-control -->
                        <label tabindex="0" class="btn btn-ghost btn-circle">
                            <svg
                                class="w-6 h-6"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                                xmlns="http://www.w3.org/2000/svg">
                                <path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                            </svg>
                        </label>
                        <ul tabindex="0" class="dropdown-content menu p-2 shadow bg-base-100 rounded-box w-52">
                            <li>
                                <a href="/#/profile"><Icon src={User} class="w-5 h-5" /> Profile</a>
                            </li>
                            <li>
                                <button on:click={toggleTheme}
                                    ><svg
                                        width="20"
                                        height="20"
                                        xmlns="http://www.w3.org/2000/svg"
                                        fill="none"
                                        viewBox="0 0 24 24"
                                        class="h-5 w-5 stroke-current"
                                        ><path
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                            stroke-width="2"
                                            d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" /></svg>
                                    Change theme</button>
                            </li>
                            <li>
                                <button
                                    on:click={async () => {
                                        authService.logout();
                                        await replace("/login");
                                    }}>
                                    <svg
                                        xmlns="http://www.w3.org/2000/svg"
                                        fill="none"
                                        viewBox="0 0 24 24"
                                        stroke-width="1.5"
                                        stroke="currentColor"
                                        class="w-5 h-5">
                                        <path
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                            d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
                                    </svg>

                                    Logout
                                </button>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>
        </div>
        <div class=" h-full">
            <ul class="menu flex flex-col p-4 pt-2 compact">
                <li>
                    <a
                        href="/#/"
                        class="capitalize active:bg-base-200 active:text-primary"
                        use:active={{ path: "/", className: "bg-base-300" }}>
                        <Icon src={Home} class="h-5 w-5" />
                        Home
                    </a>
                </li>
            </ul>

            <ul class="menu flex flex-col p-4 pt-0 compact">
                <li class="menu-title -ml-2">
                    <span>
                        My Groups

                        <div class="tooltip tooltip-left float-right" data-tip="Add Group">
                            <button on:click={onCreateGroupClick} class="hover:text-primary"
                                ><svg
                                    class="w-6 h-6"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                    xmlns="http://www.w3.org/2000/svg"
                                    ><path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="2"
                                        d="M12 6v6m0 0v6m0-6h6m-6 0H6" /></svg>
                            </button>
                        </div>
                    </span>
                </li>

                {#each $groups || [] as group}
                    <li>
                        <a
                            class="active:bg-base-200 active:text-primary"
                            href={`/#/groups/${group.id}`}
                            use:active={{ path: `/groups/${group.id}`, className: "bg-base-300" }}>
                            <span style="min-width: 20px">{group.icon}</span>
                            <span>{group.name}</span>
                        </a>
                    </li>
                {/each}
            </ul>

            <ul class="menu flex flex-col p-4 pt-0 compact">
                <li class="menu-title -ml-2">
                    <span>
                        External Groups

                        <div class="tooltip tooltip-left float-right" data-tip="Subscribe to Group">
                            <button on:click={onCreateGroupSubscriptionClick} class="hover:text-primary"
                                ><svg
                                    class="w-6 h-6"
                                    fill="none"
                                    stroke="currentColor"
                                    viewBox="0 0 24 24"
                                    xmlns="http://www.w3.org/2000/svg"
                                    ><path
                                        stroke-linecap="round"
                                        stroke-linejoin="round"
                                        stroke-width="2"
                                        d="M12 6v6m0 0v6m0-6h6m-6 0H6" /></svg>
                            </button>
                        </div>
                    </span>
                </li>

                {#each $groupSubscriptions || [] as groupSubscription}
                    <li>
                        <a
                            class="active:bg-base-200 active:text-primary"
                            href={`/#/external/groups/${groupSubscription.group.id}`}
                            use:active={{
                                path: `/external/groups/${groupSubscription.group.id}`,
                                className: "bg-base-300",
                            }}>
                            <span style="min-width: 20px">{groupSubscription.group.icon}</span>
                            <span>{groupSubscription.group.name}</span>
                        </a>
                    </li>
                {/each}
            </ul>

            <footer class="sticky inset-x-0 bottom-0 bg-base-200 border-t border-base-100 p-2 lg:hidden">
                <div class="dropdown dropdown-top">
                    <InfoDropdown />
                </div>
            </footer>
        </div>
    </aside>
</div>
