package de.serbroda.ragbag.exception;

public class EntityAlreadyExistsException extends ResourceNotFoundException {

    public EntityAlreadyExistsException() {
        super("Entity already exists");
    }

    public EntityAlreadyExistsException(String msg) {
        super(msg);
    }
}
