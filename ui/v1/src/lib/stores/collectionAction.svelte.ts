/**
 * A dialog of the collection page requested from elsewhere (the sidebar's context menu). The collection page
 * opens it once the collection is loaded and clears the request.
 */
export type CollectionAction = 'edit' | 'access' | 'share' | 'delete';

export const pendingCollectionAction = $state<{ current: { collectionId: string; action: CollectionAction } | null }>({
	current: null,
});
