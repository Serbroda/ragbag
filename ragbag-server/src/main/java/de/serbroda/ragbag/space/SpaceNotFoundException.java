package de.serbroda.ragbag.space;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;

public class SpaceNotFoundException extends ResourceNotFoundException {

    public SpaceNotFoundException(String id) {
        super("Space with ID " + id + " not found.");
    }
}
