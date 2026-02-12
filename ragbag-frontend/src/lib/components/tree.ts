import type { Component } from 'svelte';
import type { CollectionDto } from 'ragbag-frontend-sdk';

export interface TreeNode {
	id?: string;
	label: string;
	href?: string;
	expanded?: boolean;
	icon?: Component<{ class?: string }>;
	children?: TreeNode[];
}

/** Map a CollectionDto tree to a TreeNode tree. */
export function collectionToTreeNode(collection: CollectionDto, spaceId: string): TreeNode {
	return {
		id: collection.id,
		label: collection.name,
		children: collection.children?.length
			? collection.children.map((c) => collectionToTreeNode(c, spaceId))
			: undefined,
		href: `/space/${spaceId}/collections/${collection.id}`,
	};
}

/** Map an array of CollectionDto to TreeNode[]. */
export function collectionsToTree(collections: CollectionDto[], spaceId: string): TreeNode[] {
	return collections.map((c) => collectionToTreeNode(c, spaceId));
}

/** Remove a node (by reference) from anywhere in the tree. Returns true if found. */
export function removeNode(nodes: TreeNode[], target: TreeNode): boolean {
	const idx = nodes.indexOf(target);
	if (idx !== -1) {
		nodes.splice(idx, 1);
		return true;
	}
	for (const node of nodes) {
		if (node.children && removeNode(node.children, target)) return true;
	}
	return false;
}

/** Flatten a tree into a list of { id, label, depth } for use in select dropdowns. */
export interface FlatTreeOption {
	id: string;
	label: string;
	depth: number;
}

export function flattenTree(nodes: TreeNode[], depth = 0): FlatTreeOption[] {
	const result: FlatTreeOption[] = [];
	for (const node of nodes) {
		if (node.id) {
			result.push({ id: node.id, label: node.label, depth });
		}
		if (node.children) {
			result.push(...flattenTree(node.children, depth + 1));
		}
	}
	return result;
}

/** Find a node by ID in the tree. */
export function findNode(nodes: TreeNode[], id: string): TreeNode | null {
	for (const node of nodes) {
		if (node.id === id) return node;
		if (node.children) {
			const found = findNode(node.children, id);
			if (found) return found;
		}
	}
	return null;
}

/** Collect all IDs in a node's subtree (including itself). */
export function collectSubtreeIds(node: TreeNode): Set<string> {
	const ids = new Set<string>();
	if (node.id) ids.add(node.id);
	if (node.children) {
		for (const child of node.children) {
			for (const id of collectSubtreeIds(child)) {
				ids.add(id);
			}
		}
	}
	return ids;
}

/** Expand all ancestors of the node with the given ID. Returns true if found. */
export function expandToNode(nodes: TreeNode[], id: string): boolean {
	for (const node of nodes) {
		if (node.id === id) return true;
		if (node.children && expandToNode(node.children, id)) {
			node.expanded = true;
			return true;
		}
	}
	return false;
}

/** Check if `target` is the same as or a descendant of `ancestor`. */
export function isDescendantOrSelf(ancestor: TreeNode, target: TreeNode): boolean {
	if (ancestor === target) return true;
	if (!ancestor.children) return false;
	return ancestor.children.some((child) => isDescendantOrSelf(child, target));
}
