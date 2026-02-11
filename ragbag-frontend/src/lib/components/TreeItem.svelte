<script lang="ts">
	import { getContext } from 'svelte';
	import type { TreeNode } from './tree';
	import { isDescendantOrSelf } from './tree';
	import TreeItem from './TreeItem.svelte';
	import { DRAG_CTX_KEY, type DragContext } from './DraggableTreeList.svelte';

	let {
		node,
		depth = 0,
		draggable: isDraggable = false,
	}: { node: TreeNode; depth?: number; draggable?: boolean } = $props();

	let expanded = $state(false);
	let isDropTarget = $state(false);

	$effect(() => {
		if (node.expanded) expanded = true;
	});

	const hasChildren = $derived(node.children && node.children.length > 0);
	const paddingLeft = $derived(`${0.75 + depth * 1.25}rem`);

	// Drag context — getContext must be called at top level, but we only use it when isDraggable
	const dragCtx = getContext<DragContext | undefined>(DRAG_CTX_KEY);
	const dropHandler = getContext<((target: TreeNode) => void) | undefined>('tree-drop');

	function handleDragStart(e: DragEvent) {
		if (!dragCtx) return;
		dragCtx.start(node);
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
		}
	}

	function handleDragOver(e: DragEvent) {
		if (!dragCtx?.draggedNode) return;
		if (dragCtx.draggedNode === node) return;
		if (isDescendantOrSelf(dragCtx.draggedNode, node)) return;
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		isDropTarget = true;
	}

	function handleDragLeave() {
		isDropTarget = false;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		isDropTarget = false;
		dropHandler?.(node);
	}

	function handleDragEnd() {
		isDropTarget = false;
		dragCtx?.end();
	}

	const isDragged = $derived(dragCtx?.draggedNode === node);
</script>

<li>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="group flex items-center rounded-md text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
		class:opacity-40={isDragged}
		class:ring-2={isDropTarget}
		class:ring-primary-500={isDropTarget}
		class:bg-primary-50={isDropTarget}
		class:dark:bg-primary-900={isDropTarget}
		style:padding-left={paddingLeft}
		draggable={isDraggable ? 'true' : undefined}
		ondragstart={isDraggable ? handleDragStart : undefined}
		ondragover={isDraggable ? handleDragOver : undefined}
		ondragleave={isDraggable ? handleDragLeave : undefined}
		ondrop={isDraggable ? handleDrop : undefined}
		ondragend={isDraggable ? handleDragEnd : undefined}
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
				<TreeItem node={child} depth={depth + 1} draggable={isDraggable} />
			{/each}
		</ul>
	{/if}
</li>
