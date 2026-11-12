<script lang="ts">
	import { CloseOutline } from 'flowbite-svelte-icons';
	import { legalLinks, type LegalLink } from '../utils/legal';

	/**
	 * An announced change of the terms of use (UserDto.termsChangeAt): from then on they have to be accepted again.
	 * Closing it is remembered on this device for that date.
	 */
	let { at }: { at: Date } = $props();

	const storageKey = 'ragbag:terms-notice-dismissed';
	let dismissed = $state(read());
	let link = $state<LegalLink | null>(null);

	legalLinks().then((links) => (link = links.find((l) => l.page === 'terms') ?? null));

	function read(): string {
		try {
			return localStorage.getItem(storageKey) ?? '';
		} catch {
			return '';
		}
	}

	function dismiss() {
		dismissed = at.toISOString();
		try {
			localStorage.setItem(storageKey, dismissed);
		} catch {
			// Unavailable storage: hidden until the next load
		}
	}
</script>

{#if dismissed !== at.toISOString()}
	<div
		class="flex items-center gap-3 border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm text-amber-800 dark:text-amber-200"
	>
		<p class="min-w-0 flex-1">
			New terms of use apply from {at.toLocaleDateString(undefined, { dateStyle: 'long' })}; you will be asked to
			accept them then.
			{#if link}
				<a href={link.href} target={link.external ? '_blank' : undefined} rel="noopener noreferrer" class="font-medium underline">
					Read them
				</a>
			{/if}
		</p>
		<button
			type="button"
			class="rounded-md p-1 text-amber-600 hover:bg-amber-500/15 dark:text-amber-300"
			aria-label="Hide the notice"
			onclick={dismiss}
		>
			<CloseOutline class="h-4 w-4" />
		</button>
	</div>
{/if}
