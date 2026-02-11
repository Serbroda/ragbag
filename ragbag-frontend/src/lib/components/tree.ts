import type { Component } from 'svelte';

export interface TreeNode {
	label: string;
	href?: string;
	expanded?: boolean;
	icon?: Component<{ class?: string }>;
	children?: TreeNode[];
}
