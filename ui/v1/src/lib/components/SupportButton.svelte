<script lang="ts">
	import { CloseOutline } from 'flowbite-svelte-icons';
	import KoFiIcon from './KoFiIcon.svelte';

	/**
	 * The donation link (SUPPORT_URL) next to the search. Closing it is remembered on this device for that address,
	 * so it stays away; the account menu keeps the link.
	 */
	let { url }: { url: string } = $props();

	const storageKey = 'ragbag:support-dismissed';
	let dismissed = $state(read());

	function read(): string {
		try {
			return localStorage.getItem(storageKey) ?? '';
		} catch {
			return '';
		}
	}

	function dismiss() {
		dismissed = url;
		try {
			localStorage.setItem(storageKey, url);
		} catch {
			// Unavailable storage: hidden until the next load
		}
	}
</script>

{#if dismissed !== url}
	<div
		class="flex items-center rounded-md bg-orange-500/10 text-xs font-medium text-orange-700 ring-1 ring-inset ring-orange-500/30 dark:bg-orange-500/15 dark:text-orange-300"
	>
		<a
			href={url}
			target="_blank"
			rel="noopener noreferrer"
			class="flex items-center gap-1.5 rounded-l-md py-1.5 pl-2 pr-1.5 transition-colors hover:bg-orange-500/15 hover:text-orange-800 dark:hover:text-orange-200"
			title="Ragbag is a hobby project; a coffee helps with the server costs"
		>
			<KoFiIcon class="h-4 w-4 shrink-0" />
			<span class="hidden md:inline">Support Ragbag</span>
		</a>
		<button
			type="button"
			class="rounded-r-md py-1.5 pl-0.5 pr-1.5 text-orange-500/70 transition-colors hover:bg-orange-500/15 hover:text-orange-700 dark:hover:text-orange-200"
			aria-label="Hide the support link"
			title="Hide (it stays in the account menu)"
			onclick={dismiss}
		>
			<CloseOutline class="h-3 w-3" />
		</button>
	</div>
{/if}
