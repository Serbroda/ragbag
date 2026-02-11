<script lang="ts">
	import type { TreeNode } from './tree';
	import TreeItem from './TreeItem.svelte';

	let { node, depth = 0 }: { node: TreeNode; depth?: number } = $props();

	let expanded = $state(false);

	$effect(() => {
		if (node.expanded) expanded = true;
	});

	const hasChildren = $derived(node.children && node.children.length > 0);
	const paddingLeft = $derived(`${0.75 + depth * 1.25}rem`);
</script>

<li>
	<div
		class="group flex items-center rounded-md text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
		style:padding-left={paddingLeft}
	>
		<!-- Icon -->
		{#if node.icon}
			{@const Icon = node.icon}
			<Icon class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400" />
		{/if}

		<!-- Clickable label -->
		{#if node.href}
			<a
				href={node.href}
				class="flex-1 truncate rounded-md px-2 py-1.5 hover:text-gray-900 dark:hover:text-white"
			>
				{node.label}
			</a>
		{:else}
			<button
				type="button"
				class="flex-1 truncate rounded-md px-2 py-1.5 text-left hover:text-gray-900 dark:hover:text-white"
				onclick={() => hasChildren && (expanded = !expanded)}
			>
				{node.label}
			</button>
		{/if}

		<!-- Expand/Collapse toggle (right-aligned) -->
		{#if hasChildren}
			<button
				type="button"
				class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 hover:text-gray-600 me-1 dark:text-gray-500 dark:hover:text-gray-300"
				aria-label={expanded ? 'Collapse' : 'Expand'}
				onclick={() => (expanded = !expanded)}
			>
				<svg
					class="h-3.5 w-3.5 transition-transform duration-150"
					class:rotate-90={expanded}
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		{/if}
	</div>

	<!-- Children -->
	{#if hasChildren && expanded}
		<ul>
			{#each node.children! as child (child.label)}
				<TreeItem node={child} depth={depth + 1} />
			{/each}
		</ul>
	{/if}
</li>
