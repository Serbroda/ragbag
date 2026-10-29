<script lang="ts" module>
	import type { TreeNode } from './tree';

	export interface DragContext {
		readonly draggedNode: TreeNode | null;
		start(node: TreeNode): void;
		end(): void;
	}

	export const DRAG_CTX_KEY = Symbol('tree-drag');

	/** Dropping on a collection sorts the dragged one in front of it, behind it or moves it inside. */
	export type DropZone = 'before' | 'after' | 'into';
</script>

<script lang="ts">
	import { setContext } from 'svelte';
	import { isDescendantOrSelf } from './tree';
	import TreeItem from './TreeItem.svelte';
	import { isExpanded } from '../stores/preferences.svelte';

	let {
		nodes = $bindable(),
		draggable = true,
		onmove,
	}: {
		nodes: TreeNode[];
		draggable?: boolean;
		/** Moves the collection below newParent (null: top level), in front of before (null: at the end). */
		onmove?: (movedNode: TreeNode, newParent: TreeNode | null, before: TreeNode | null) => Promise<void>;
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

	/** The parent of the node (null at the top level). */
	function parentOf(list: TreeNode[], target: TreeNode, parent: TreeNode | null = null): TreeNode | null | undefined {
		for (const node of list) {
			if (node === target) return parent;
			const found = node.children && parentOf(node.children, target, node);
			if (found !== undefined) return found;
		}
		return undefined;
	}

	async function handleDrop(target: TreeNode, zone: DropZone) {
		const moved = draggedNode;
		draggedNode = null;
		if (!moved || moved === target || isDescendantOrSelf(moved, target)) return;

		let parent: TreeNode | null = target;
		let before: TreeNode | null = null;
		const openWithChildren = !!target.id && isExpanded(target.id) && !!target.children?.length;
		if (zone === 'after' && openWithChildren) {
			// Below an open collection the marker sits above its first child: that is where it goes
			before = target.children!.find((node) => node !== moved) ?? null;
		} else if (zone !== 'into') {
			parent = parentOf(nodes, target) ?? null;
			const siblings = (parent ? parent.children : nodes) ?? [];
			if (zone === 'before') {
				before = target;
			} else {
				// Behind the target means in front of its next sibling, or at the end
				const next = siblings.slice(siblings.indexOf(target) + 1).find((node) => node !== moved);
				before = next ?? null;
			}
		}
		try {
			await onmove?.(moved, parent, before);
		} catch (err) {
			console.error('Move failed:', err);
		}
	}

	setContext('tree-drop', handleDrop);
</script>

<ul class="space-y-0.5 py-2">
	{#each nodes as node (node.id ?? node.label)}
		<TreeItem {node} {draggable} />
	{/each}
</ul>
