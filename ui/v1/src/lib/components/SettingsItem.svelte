<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Button } from 'flowbite-svelte';

	/**
	 * One setting as a row: what it is, how it is set right now and a button. Its form (children) only shows
	 * after the button was pressed, so a page of settings never looks like one long form.
	 */
	let {
		title,
		description,
		status,
		tone = 'neutral',
		action,
		danger = false,
		open = $bindable(false),
		children,
	}: {
		title: string;
		description?: string;
		/** The current state, e.g. "On" or "Not set". */
		status?: string;
		tone?: 'neutral' | 'good';
		/** Label of the button that opens the form; none: the content is always shown. */
		action?: string;
		danger?: boolean;
		open?: boolean;
		children?: Snippet;
	} = $props();
</script>

<div class="py-4 first:pt-0 last:pb-0">
	<!-- The button below the text on phones, next to it on wider screens -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-start">
		<div class="min-w-0 flex-1">
			<p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm font-medium text-gray-900 dark:text-white">
				{title}
				{#if status}
					<span
						class="rounded-md px-1.5 py-0.5 text-xs font-medium {tone === 'good'
							? 'bg-green-500/15 text-green-700 dark:text-green-300'
							: 'bg-gray-500/15 text-gray-600 dark:text-gray-300'}">{status}</span
					>
				{/if}
			</p>
			{#if description}
				<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{description}</p>
			{/if}
		</div>
		{#if action && !open}
			<Button size="sm" color={danger ? 'red' : 'light'} outline={danger} class="shrink-0 self-start" onclick={() => (open = true)}>
				{action}
			</Button>
		{/if}
	</div>
	{#if children && (open || !action)}
		<div class="mt-3">{@render children()}</div>
	{/if}
</div>
