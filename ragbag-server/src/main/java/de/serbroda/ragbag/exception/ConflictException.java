package de.serbroda.ragbag.exception;

public class ConflictException extends ResourceNotFoundException {

    public ConflictException(String msg) {
        super(msg);
    }
}
