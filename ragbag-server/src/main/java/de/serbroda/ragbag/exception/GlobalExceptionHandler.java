package de.serbroda.ragbag.exception;

import jakarta.servlet.http.HttpServletRequest;
import java.net.URI;
import java.util.Map;
import java.util.stream.Collectors;
import org.springframework.http.HttpStatus;
import org.springframework.http.ProblemDetail;
import org.springframework.security.authorization.AuthorizationDeniedException;
import org.springframework.validation.FieldError;
import org.springframework.web.bind.MethodArgumentNotValidException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(ResourceNotFoundException.class)
    public ProblemDetail handleNotFound(ResourceNotFoundException ex, HttpServletRequest request) {
        ProblemDetail problem = ProblemDetail.forStatus(HttpStatus.NOT_FOUND);
        problem.setTitle("Resource not found");
        problem.setDetail(ex.getMessage());
        problem.setInstance(URI.create(request.getRequestURI()));
        return problem;
    }

    @ExceptionHandler(IllegalArgumentException.class)
    public ProblemDetail handleBadRequest(IllegalArgumentException ex, HttpServletRequest request) {
        ProblemDetail problem = ProblemDetail.forStatus(HttpStatus.BAD_REQUEST);
        problem.setTitle("Invalid request");
        problem.setDetail(ex.getMessage());
        problem.setInstance(URI.create(request.getRequestURI()));
        return problem;
    }

    @ExceptionHandler(AuthorizationDeniedException.class)
    public ProblemDetail handleAuthorizationDeniedException(
            AuthorizationDeniedException ex, HttpServletRequest request) {
        ProblemDetail problem = ProblemDetail.forStatus(HttpStatus.FORBIDDEN);
        problem.setTitle("Access denied");
        problem.setDetail("Access denied for resource");
        problem.setInstance(URI.create(request.getRequestURI()));
        return problem;
    }

    @ExceptionHandler(Exception.class)
    public ProblemDetail handleGeneric(Exception ex, HttpServletRequest request) {
        ProblemDetail problem = ProblemDetail.forStatus(HttpStatus.INTERNAL_SERVER_ERROR);
        problem.setTitle("Internal server error");
        problem.setDetail("An unexpected error occurred");
        problem.setInstance(URI.create(request.getRequestURI()));
        return problem;
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ProblemDetail handleValidation(MethodArgumentNotValidException ex, HttpServletRequest request) {
        ProblemDetail problem = ProblemDetail.forStatus(HttpStatus.BAD_REQUEST);

        problem.setTitle("Validation failed");
        problem.setDetail("One or more fields are invalid");
        problem.setInstance(URI.create(request.getRequestURI()));

        Map<String, String> errors = ex.getBindingResult().getFieldErrors().stream()
                .collect(Collectors.toMap(FieldError::getField, FieldError::getDefaultMessage, (a, b) -> a));

        problem.setProperty("errors", errors);
        return problem;
    }
}
