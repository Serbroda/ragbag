<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import { StarOutline, StarSolid } from 'flowbite-svelte-icons';
	import { apiConfig } from '../api/client';

	let {
		bookmarkId,
		favorite = $bindable(false),
		onchange,
		class: className = '',
	}: {
		bookmarkId: string;
		favorite?: boolean;
		onchange?: (favorite: boolean) => void;
		class?: string;
	} = $props();

	let busy = $state(false);

	async function toggle(e: MouseEvent) {
		// Sits on top of cards whose title link covers the whole card
		e.preventDefault();
		e.stopPropagation();
		if (busy) return;
		busy = true;
		const next = !favorite;
		favorite = next; // optimistic
		try {
			const api = new BookmarkApi(apiConfig());
			if (next) await api.addFavorite({ bookmarkId });
			else await api.removeFavorite({ bookmarkId });
			onchange?.(next);
		} catch {
			favorite = !next;
		} finally {
			busy = false;
		}
	}
</script>

<button
	type="button"
	class="rounded-md p-1 transition {favorite
		? 'text-primary-500 hover:text-primary-400'
		: 'text-gray-400 hover:text-primary-500'} {className}"
	title={favorite ? 'Remove from favorites' : 'Add to favorites'}
	aria-label={favorite ? 'Remove from favorites' : 'Add to favorites'}
	aria-pressed={favorite}
	onclick={toggle}
>
	{#if favorite}
		<StarSolid class="h-4 w-4" />
	{:else}
		<StarOutline class="h-4 w-4" />
	{/if}
</button>
