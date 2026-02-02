package de.serbroda.ragbag.exception;

public class CollectionNotFoundException extends ResourceNotFoundException {

    public CollectionNotFoundException(String id) {
        super("Collection with ID " + id + " not found.");
    }
}
