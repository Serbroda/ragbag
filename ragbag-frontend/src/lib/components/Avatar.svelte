<script lang="ts">
	import { imageUrl } from '../api/client';
	import { colorFor, initials } from '../utils/format';

	let {
		name,
		imageId,
		class: className = 'h-8 w-8 rounded-full text-xs',
	}: {
		name: string;
		/** Uploaded avatar or icon; without one, colored initials are shown like Discord's default avatars. */
		imageId?: string | null;
		/** Size, shape and text size, e.g. "h-12 w-12 rounded-2xl text-sm". */
		class?: string;
	} = $props();

	let failedId = $state<string | null>(null);
	const showImage = $derived(!!imageId && failedId !== imageId);
</script>

{#if showImage}
	<img
		src={imageUrl(imageId!)}
		alt=""
		class="shrink-0 object-cover {className}"
		onerror={() => (failedId = imageId ?? null)}
	/>
{:else}
	<span
		class="flex shrink-0 items-center justify-center font-semibold text-white {colorFor(name)} {className}"
		aria-hidden="true"
	>
		{initials(name)}
	</span>
{/if}
