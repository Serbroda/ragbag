<script lang="ts" module>
	import type { TreeNode } from './tree';

	export interface DragContext {
		readonly draggedNode: TreeNode | null;
		start(node: TreeNode): void;
		end(): void;
	}

	export const DRAG_CTX_KEY = Symbol('tree-drag');
</script>

<script lang="ts">
	import { setContext } from 'svelte';
	import { removeNode, isDescendantOrSelf } from './tree';
	import TreeItem from './TreeItem.svelte';

	let {
		nodes = $bindable(),
		draggable = true,
		onmove,
	}: {
		nodes: TreeNode[];
		draggable?: boolean;
		onmove?: (movedNode: TreeNode, newParent: TreeNode | null) => void;
	} = $props();

	let draggedNode = $state<TreeNode | null>(null);
	let rootDropTarget = $state(false);

	const ctx: DragContext = {
		get draggedNode() {
			return draggedNode;
		},
		start(node: TreeNode) {
			draggedNode = node;
		},
		end() {
			draggedNode = null;
			rootDropTarget = false;
		},
	};

	setContext(DRAG_CTX_KEY, ctx);

	export function handleDrop(targetNode: TreeNode) {
		if (!draggedNode || draggedNode === targetNode) return;
		if (isDescendantOrSelf(draggedNode, targetNode)) return;

		const moved = draggedNode;
		removeNode(nodes, moved);
		if (!targetNode.children) targetNode.children = [];
		targetNode.children.push(moved);
		targetNode.expanded = true;

		nodes = [...nodes];
		onmove?.(moved, targetNode);
		draggedNode = null;
	}

	function handleRootDrop(e: DragEvent) {
		e.preventDefault();
		rootDropTarget = false;
		if (!draggedNode) return;

		// Already at root level — nothing to do
		if (nodes.includes(draggedNode)) {
			draggedNode = null;
			return;
		}

		const moved = draggedNode;
		removeNode(nodes, moved);
		nodes = [...nodes, moved];
		onmove?.(moved, null);
		draggedNode = null;
	}

	function handleRootDragOver(e: DragEvent) {
		if (!draggedNode) return;
		if (nodes.includes(draggedNode)) return;
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		rootDropTarget = true;
	}

	function handleRootDragLeave() {
		rootDropTarget = false;
	}

	setContext('tree-drop', handleDrop);
</script>

<ul class="space-y-0.5 py-2">
	{#each nodes as node (node.label)}
		<TreeItem {node} {draggable} />
	{/each}

	<!-- Drop zone: move to root level -->
	{#if draggable && draggedNode && !nodes.includes(draggedNode)}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<li
			class="mx-2 mt-1 flex items-center justify-center rounded-md border-2 border-dashed py-2 text-xs transition-colors"
			class:border-primary-400={rootDropTarget}
			class:bg-primary-50={rootDropTarget}
			class:text-primary-600={rootDropTarget}
			class:dark:bg-primary-900={rootDropTarget}
			class:dark:text-primary-400={rootDropTarget}
			class:border-gray-300={!rootDropTarget}
			class:text-gray-400={!rootDropTarget}
			class:dark:border-gray-600={!rootDropTarget}
			ondragover={handleRootDragOver}
			ondragleave={handleRootDragLeave}
			ondrop={handleRootDrop}
		>
			Move to top level
		</li>
	{/if}
</ul>
