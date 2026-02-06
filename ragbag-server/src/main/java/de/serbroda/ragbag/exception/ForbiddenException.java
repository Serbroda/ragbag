package de.serbroda.ragbag.exception;

public class ForbiddenException extends ResourceNotFoundException {

    public ForbiddenException(String msg) {
        super(msg);
    }
}
