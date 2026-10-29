<script lang="ts" module>
	/** The fixed set the server accepts (reactionEmojis in internal/server/activity.go), in this order. */
	export const REACTION_EMOJIS = ['👍', '❤️', '😂', '🔥', '😮', '🙏'];
</script>

<script lang="ts">
	import { BookmarkApi } from 'ragbag-frontend-sdk';
	import type { ReactionDto } from 'ragbag-frontend-sdk';
	import { FaceGrinOutline } from 'flowbite-svelte-icons';
	import { apiConfig } from '../api/client';

	let { bookmarkId, reactions: initial = [] }: { bookmarkId: string; reactions?: ReactionDto[] } = $props();

	let reactions = $state<ReactionDto[]>([]);
	$effect(() => {
		reactions = initial;
	});

	let pickerOpen = $state(false);
	let busy = $state(false);

	async function toggle(emoji: string, e: MouseEvent) {
		// Sits on top of cards whose title link covers the whole card
		e.preventDefault();
		e.stopPropagation();
		pickerOpen = false;
		if (busy) return;
		busy = true;
		const reacted = !reactions.some((r) => r.emoji === emoji && r.reacted);
		try {
			reactions = await new BookmarkApi(apiConfig()).setReaction({
				bookmarkId,
				setReactionDto: { emoji, reacted },
			});
		} catch {
			// Unchanged; the next load shows the real state
		} finally {
			busy = false;
		}
	}

	function openPicker(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		pickerOpen = !pickerOpen;
	}
</script>

<svelte:window onclick={() => (pickerOpen = false)} />

<div class="relative z-10 flex flex-wrap items-center gap-1">
	{#each reactions as reaction (reaction.emoji)}
		<button
			type="button"
			class="flex items-center gap-1 rounded-full px-1.5 py-0.5 text-xs ring-1 ring-inset transition-colors {reaction.reacted
				? 'bg-primary-500/15 text-primary-700 ring-primary-500/40 dark:text-primary-300'
				: 'bg-gray-100 text-gray-600 ring-transparent hover:ring-gray-300 dark:bg-gray-800 dark:text-gray-300 dark:hover:ring-gray-600'}"
			title={reaction.users.join(', ')}
			aria-label="{reaction.emoji} {reaction.count}, {reaction.reacted ? 'remove your reaction' : 'react'}"
			aria-pressed={reaction.reacted}
			onclick={(e) => toggle(reaction.emoji, e)}
		>
			<span class="text-sm leading-none">{reaction.emoji}</span>
			<span class="font-medium tabular-nums">{reaction.count}</span>
		</button>
	{/each}
	<div class="relative">
		<button
			type="button"
			class="flex h-6 items-center rounded-full px-1.5 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-gray-800 dark:hover:text-gray-200 {reactions.length
				? ''
				: 'opacity-0 focus:opacity-100 group-hover:opacity-100'} {pickerOpen ? 'opacity-100' : ''}"
			title="Add reaction"
			aria-label="Add reaction"
			aria-expanded={pickerOpen}
			onclick={openPicker}
		>
			<FaceGrinOutline class="h-4 w-4" />
		</button>
		{#if pickerOpen}
			<div
				class="absolute bottom-8 right-0 z-30 flex gap-0.5 rounded-full bg-white p-1 shadow-lg ring-1 ring-gray-200 dark:bg-gray-800 dark:ring-gray-700"
				role="menu"
			>
				{#each REACTION_EMOJIS as emoji (emoji)}
					<button
						type="button"
						role="menuitem"
						class="flex h-8 w-8 items-center justify-center rounded-full text-lg transition hover:scale-110 hover:bg-gray-100 dark:hover:bg-gray-700"
						aria-label="React with {emoji}"
						onclick={(e) => toggle(emoji, e)}
					>
						{emoji}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>
