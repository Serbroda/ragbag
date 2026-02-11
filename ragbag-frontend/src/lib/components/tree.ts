export interface TreeNode {
	label: string;
	href?: string;
	expanded?: boolean;
	children?: TreeNode[];
}
