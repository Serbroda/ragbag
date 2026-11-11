import type { Component } from 'svelte';
import type { CollectionDto } from 'ragbag-frontend-sdk';
import { EyeSlashOutline, FilterOutline, FolderOutline, GlobeOutline, LockOutline } from 'flowbite-svelte-icons';
import { p } from '../router';

export interface TreeNode {
	id?: string;
	label: string;
	/** Link target including the router base (e.g. /app); build it with p() */
	href?: string;
	/** Route path without the base, used to detect the active item */
	path?: string;
	/** Private collection (only its members see it and everything below). */
	restricted?: boolean;
	/** Links directly in the collection. */
	bookmarkCount?: number;
	/** Locked by the current user: only listed while unlocked. */
	locked?: boolean;
	/** Links directly in the collection that others added since the user last looked. */
	newCount?: number;
	/** Search of a smart collection, which shows its results and holds neither links nor sub-collections. */
	query?: string;
	icon?: Component<{ class?: string }>;
	children?: TreeNode[];
}

/** Map a CollectionDto tree to a TreeNode tree. */
export function collectionToTreeNode(collection: CollectionDto, spaceId: string): TreeNode {
	return {
		id: collection.id,
		label: collection.name,
		// Locked ones only show up unlocked, private ones only for selected members, shared ones for anybody with the link
		icon: collection.locked
			? EyeSlashOutline
			: collection.restricted
				? LockOutline
				: collection.query
					? FilterOutline
					: collection.shared
						? GlobeOutline
						: FolderOutline,
		restricted: collection.restricted,
		query: collection.query,
		bookmarkCount: collection.bookmarkCount,
		locked: collection.locked,
		newCount: collection.newCount,
		children: collection.children?.length
			? collection.children.map((c) => collectionToTreeNode(c, spaceId))
			: undefined,
		path: `/space/${spaceId}/collections/${collection.id}`,
		href: p('/space/:spaceId/collections/:collectionId', { params: { spaceId, collectionId: collection.id } }),
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

/** New links in the node and everything below it. */
export function totalNewCount(node: TreeNode): number {
	return (node.newCount ?? 0) + (node.children ?? []).reduce((sum, child) => sum + totalNewCount(child), 0);
}

/** Flatten a tree into a list of { id, label, depth } for use in select dropdowns. */
export interface FlatTreeOption {
	id: string;
	label: string;
	depth: number;
	/** Set for smart collections. */
	query?: string;
}

export function flattenTree(nodes: TreeNode[], depth = 0): FlatTreeOption[] {
	const result: FlatTreeOption[] = [];
	for (const node of nodes) {
		if (node.id) {
			result.push({ id: node.id, label: node.label, depth, query: node.query });
		}
		if (node.children) {
			result.push(...flattenTree(node.children, depth + 1));
		}
	}
	return result;
}

/** The collections that can take links or sub-collections: all but the smart ones. */
export function flattenTargets(nodes: TreeNode[]): FlatTreeOption[] {
	return flattenTree(nodes).filter((option) => !option.query);
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

/** Check if `target` is the same as or a descendant of `ancestor`. */
export function isDescendantOrSelf(ancestor: TreeNode, target: TreeNode): boolean {
	if (ancestor === target) return true;
	if (!ancestor.children) return false;
	return ancestor.children.some((child) => isDescendantOrSelf(child, target));
}

/** Nodes from the root down to the node with the given id (inclusive), or [] if it is not in the tree. */
export function pathToNode(nodes: TreeNode[], id: string): TreeNode[] {
	for (const node of nodes) {
		if (node.id === id) return [node];
		if (node.children) {
			const path = pathToNode(node.children, id);
			if (path.length > 0) return [node, ...path];
		}
	}
	return [];
}
