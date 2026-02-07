package de.serbroda.ragbag.shared.exception;

public class ConflictException extends ResourceNotFoundException {

    public ConflictException(String msg) {
        super(msg);
    }
}
