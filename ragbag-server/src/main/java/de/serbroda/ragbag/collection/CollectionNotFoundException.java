package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;

public class CollectionNotFoundException extends ResourceNotFoundException {

    public CollectionNotFoundException(String id) {
        super("Collection with ID " + id + " not found.");
    }
}
