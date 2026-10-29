<script lang="ts">
	import { getContext } from 'svelte';
	import { DotsHorizontalOutline } from 'flowbite-svelte-icons';
	import type { TreeNode } from './tree';
	import { isDescendantOrSelf, totalNewCount } from './tree';
	import TreeItem from './TreeItem.svelte';
	import { route } from '../router';
	import { DRAG_CTX_KEY, type DragContext } from './DraggableTreeList.svelte';
	import { isExpanded, setExpanded } from '../stores/preferences.svelte';
	import { bookmarkDrag } from '../stores/bookmarkDrag.svelte';

	let {
		node,
		depth = 0,
		draggable: isDraggable = false,
	}: { node: TreeNode; depth?: number; draggable?: boolean } = $props();

	let isDropTarget = $state(false);

	const expanded = $derived(!!node.id && isExpanded(node.id));

	function toggle() {
		if (node.id) setExpanded(node.id, !expanded);
	}

	const hasChildren = $derived(node.children && node.children.length > 0);
	/** A collapsed collection also counts what is new below it, so nothing gets lost in a closed branch. */
	const newCount = $derived(expanded ? (node.newCount ?? 0) : totalNewCount(node));
	const paddingLeft = $derived(`${depth * 1}rem`);

	// Drag context — getContext must be called at top level, but we only use it when isDraggable
	const dragCtx = getContext<DragContext | undefined>(DRAG_CTX_KEY);
	const dropHandler = getContext<((target: TreeNode) => void) | undefined>('tree-drop');
	const openMenu = getContext<((node: TreeNode, x: number, y: number) => void) | undefined>('collection-menu');

	/** A link dragged from a collection page can be dropped on any other collection. */
	const acceptsBookmark = $derived(
		!!node.id && !!bookmarkDrag.current && bookmarkDrag.current.collectionId !== node.id,
	);
	let expandTimer: ReturnType<typeof setTimeout> | undefined;

	function handleDragStart(e: DragEvent) {
		if (!dragCtx) return;
		dragCtx.start(node);
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
		}
	}

	function handleDragOver(e: DragEvent) {
		if (acceptsBookmark) {
			e.preventDefault();
			if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
			isDropTarget = true;
			// Hovering a collapsed parent opens it, so links can be dropped into its children
			if (hasChildren && !expanded && !expandTimer) expandTimer = setTimeout(toggle, 600);
			return;
		}
		if (!dragCtx?.draggedNode) return;
		if (dragCtx.draggedNode === node) return;
		if (isDescendantOrSelf(dragCtx.draggedNode, node)) return;
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
		isDropTarget = true;
	}

	function handleDragLeave() {
		isDropTarget = false;
		clearTimeout(expandTimer);
		expandTimer = undefined;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		handleDragLeave();
		if (acceptsBookmark) {
			bookmarkDrag.current!.drop(node.id!);
			return;
		}
		dropHandler?.(node);
	}

	function handleContextMenu(e: MouseEvent) {
		if (!node.id || !openMenu) return;
		e.preventDefault();
		e.stopPropagation();
		openMenu(node, e.clientX, e.clientY);
	}

	function handleDragEnd() {
		isDropTarget = false;
		dragCtx?.end();
	}

	const isDragged = $derived(dragCtx?.draggedNode === node);
	const isActive = $derived(!!node.path && route.pathname === node.path);
</script>

<li>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="group mx-2 flex items-center rounded-md text-sm transition-colors {isActive
			? 'bg-gray-200 font-medium text-gray-900 dark:bg-gray-700/70 dark:text-white'
			: 'text-gray-600 hover:bg-gray-200/60 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200'}"
		class:opacity-40={isDragged}
		class:ring-2={isDropTarget}
		class:ring-primary-500={isDropTarget}
		class:bg-primary-50={isDropTarget}
		class:dark:bg-primary-900={isDropTarget}
		style:padding-left={paddingLeft}
		draggable={isDraggable ? 'true' : undefined}
		ondragstart={isDraggable ? handleDragStart : undefined}
		ondragover={isDraggable || node.id ? handleDragOver : undefined}
		ondragleave={isDraggable || node.id ? handleDragLeave : undefined}
		ondrop={isDraggable || node.id ? handleDrop : undefined}
		ondragend={isDraggable ? handleDragEnd : undefined}
		oncontextmenu={node.id ? handleContextMenu : undefined}
	>
		<!-- Icon -->
		{#if node.icon}
			{@const Icon = node.icon}
			<Icon class="ms-2 h-4 w-4 shrink-0 {isActive ? 'text-primary-500' : 'text-gray-400 dark:text-gray-500'}" />
		{/if}

		<!-- Clickable label -->
		{#if node.href}
			<a
				href={node.href}
				class="flex-1 truncate rounded-md px-2 py-1.5 {newCount > 0 && !isActive
					? 'font-semibold text-gray-900 dark:text-white'
					: ''}"
				aria-current={isActive ? 'page' : undefined}
			>
				{node.label}
			</a>
			{#if newCount > 0}
				<span
					class="me-1 shrink-0 rounded-full bg-primary-500 px-1.5 text-[11px] font-bold leading-4 text-white"
					title="{newCount} new link{newCount === 1 ? '' : 's'}">{newCount}</span
				>
			{/if}
		{:else}
			<button
				type="button"
				class="flex-1 truncate rounded-md px-2 py-1.5 text-left"
				onclick={() => hasChildren && toggle()}
			>
				{node.label}
			</button>
		{/if}

		{#if node.id && openMenu}
			<button
				type="button"
				class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 opacity-0 hover:text-gray-600 focus:opacity-100 group-hover:opacity-100 dark:text-gray-500 dark:hover:text-gray-300 {hasChildren
					? ''
					: 'me-1'}"
				aria-label="Collection actions"
				aria-haspopup="menu"
				onclick={(e) => {
					e.stopPropagation();
					const rect = e.currentTarget.getBoundingClientRect();
					openMenu(node, rect.left, rect.bottom + 4);
				}}
			>
				<DotsHorizontalOutline class="h-4 w-4" />
			</button>
		{/if}

		<!-- Expand/Collapse toggle (right-aligned) -->
		{#if hasChildren}
			<button
				type="button"
				class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 hover:text-gray-600 me-1 dark:text-gray-500 dark:hover:text-gray-300"
				aria-label={expanded ? 'Collapse' : 'Expand'}
				onclick={toggle}
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
			{#each node.children! as child (child.id ?? child.label)}
				<TreeItem node={child} depth={depth + 1} draggable={isDraggable} />
			{/each}
		</ul>
	{/if}
</li>
