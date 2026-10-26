<script lang="ts" module>
	export interface SwitcherItem {
		id: string;
		label: string;
		/** Secondary text, e.g. the parent path or "Space". */
		hint: string;
		kind: 'space' | 'collection';
		go: () => void;
	}
</script>

<script lang="ts">
	import { tick } from 'svelte';

	let { open = $bindable(false), items }: { open?: boolean; items: SwitcherItem[] } = $props();

	let query = $state('');
	let selected = $state(0);
	let input = $state<HTMLInputElement | null>(null);

	const results = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const matches = q ? items.filter((item) => item.label.toLowerCase().includes(q)) : items;
		// Prefix matches first, like Discord's switcher
		return [...matches]
			.sort((a, b) => Number(b.label.toLowerCase().startsWith(q)) - Number(a.label.toLowerCase().startsWith(q)))
			.slice(0, 12);
	});

	$effect(() => {
		if (open) {
			query = '';
			selected = 0;
			tick().then(() => input?.focus());
		}
	});

	$effect(() => {
		// Keep the selection in range while typing
		if (selected >= results.length) selected = Math.max(0, results.length - 1);
	});

	function choose(item: SwitcherItem | undefined) {
		if (!item) return;
		open = false;
		item.go();
	}

	function onkeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selected = (selected + 1) % Math.max(results.length, 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selected = (selected - 1 + results.length) % Math.max(results.length, 1);
		} else if (e.key === 'Enter') {
			e.preventDefault();
			choose(results[selected]);
		} else if (e.key === 'Escape') {
			open = false;
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
	<div class="fixed inset-0 z-[60] flex items-start justify-center bg-gray-900/60 px-4 pt-[15vh]" onclick={() => (open = false)}>
		<div
			class="w-full max-w-lg overflow-hidden rounded-lg bg-white shadow-2xl dark:bg-gray-800"
			role="dialog"
			tabindex="-1"
			aria-modal="true"
			aria-label="Quick switcher"
			onclick={(e) => e.stopPropagation()}
		>
			<div class="border-b border-gray-200 p-3 dark:border-gray-700">
				<input
					bind:this={input}
					bind:value={query}
					{onkeydown}
					type="text"
					placeholder="Where would you like to go?"
					class="w-full rounded-md border-0 bg-gray-100 px-3 py-2.5 text-gray-900 focus:ring-2 focus:ring-primary-500 dark:bg-gray-900 dark:text-white"
				/>
			</div>
			<ul class="max-h-80 overflow-y-auto p-2" role="listbox">
				{#each results as item, i (item.kind + item.id)}
					<li role="option" aria-selected={i === selected}>
						<button
							type="button"
							class="flex w-full items-center justify-between gap-3 rounded-md px-3 py-2 text-left text-sm {i === selected
								? 'bg-primary-500 text-white'
								: 'text-gray-700 dark:text-gray-200'}"
							onmouseenter={() => (selected = i)}
							onclick={() => choose(item)}
						>
							<span class="truncate">
								<span class="me-2 opacity-60">{item.kind === 'space' ? '◆' : '#'}</span>{item.label}
							</span>
							<span class="shrink-0 truncate text-xs opacity-70">{item.hint}</span>
						</button>
					</li>
				{:else}
					<li class="px-3 py-6 text-center text-sm text-gray-500 dark:text-gray-400">No matches</li>
				{/each}
			</ul>
			<p class="border-t border-gray-200 px-4 py-2 text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">
				<kbd class="font-semibold">↑↓</kbd> to navigate · <kbd class="font-semibold">Enter</kbd> to open ·
				<kbd class="font-semibold">Esc</kbd> to close
			</p>
		</div>
	</div>
{/if}
