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
		onmove?: (movedNode: TreeNode, newParent: TreeNode) => void;
	} = $props();

	let draggedNode = $state<TreeNode | null>(null);

	const ctx: DragContext = {
		get draggedNode() {
			return draggedNode;
		},
		start(node: TreeNode) {
			draggedNode = node;
		},
		end() {
			draggedNode = null;
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

		// Trigger reactivity
		nodes = [...nodes];

		onmove?.(moved, targetNode);
		draggedNode = null;
	}

	setContext('tree-drop', handleDrop);
</script>

<ul class="space-y-0.5 py-2">
	{#each nodes as node (node.label)}
		<TreeItem {node} {draggable} />
	{/each}
</ul>
