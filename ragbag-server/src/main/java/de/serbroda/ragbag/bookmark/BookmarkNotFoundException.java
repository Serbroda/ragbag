package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;

public class BookmarkNotFoundException extends ResourceNotFoundException {

    public BookmarkNotFoundException(String id) {
        super("Bookmark with ID " + id + " not found.");
    }
}
