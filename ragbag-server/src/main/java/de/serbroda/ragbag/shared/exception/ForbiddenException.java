package de.serbroda.ragbag.shared.exception;

public class ForbiddenException extends ResourceNotFoundException {

    public ForbiddenException(String msg) {
        super(msg);
    }
}
