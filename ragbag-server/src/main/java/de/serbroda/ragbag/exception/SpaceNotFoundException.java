package de.serbroda.ragbag.exception;

public class SpaceNotFoundException extends ResourceNotFoundException {

    public SpaceNotFoundException(String id) {
        super("Space with ID " + id + " not found.");
    }
}
