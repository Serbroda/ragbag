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

/** Check if `target` is the same as or a descendant of `ancestor`. */
export function isDescendantOrSelf(ancestor: TreeNode, target: TreeNode): boolean {
	if (ancestor === target) return true;
	if (!ancestor.children) return false;
	return ancestor.children.some((child) => isDescendantOrSelf(child, target));
}
