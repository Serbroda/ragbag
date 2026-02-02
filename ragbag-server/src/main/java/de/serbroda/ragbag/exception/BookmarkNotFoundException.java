package de.serbroda.ragbag.exception;

public class BookmarkNotFoundException extends ResourceNotFoundException {

    public BookmarkNotFoundException(String id) {
        super("Bookmark with ID " + id + " not found.");
    }
}
