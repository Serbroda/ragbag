/**
 * The link being dragged from a collection page, so that the collections in the sidebar can accept it.
 * Kept outside the DOM drag data because dragover handlers may not read that data.
 */
export interface BookmarkDrag {
	/** Collection the link is in; dropping it there again does nothing. */
	collectionId: string;
	drop(targetCollectionId: string): void;
}

export const bookmarkDrag = $state<{ current: BookmarkDrag | null }>({ current: null });
