import type { Component } from 'svelte';

export interface TreeNode {
	label: string;
	href?: string;
	expanded?: boolean;
	icon?: Component<{ class?: string }>;
	children?: TreeNode[];
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

/** Check if `target` is the same as or a descendant of `ancestor`. */
export function isDescendantOrSelf(ancestor: TreeNode, target: TreeNode): boolean {
	if (ancestor === target) return true;
	if (!ancestor.children) return false;
	return ancestor.children.some((child) => isDescendantOrSelf(child, target));
}
