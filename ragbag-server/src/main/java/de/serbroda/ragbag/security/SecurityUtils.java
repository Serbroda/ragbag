package de.serbroda.ragbag.security;

import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;

public final class SecurityUtils {

    private SecurityUtils() {}

    public static Jwt currentJwt() {
        Authentication auth = SecurityContextHolder.getContext().getAuthentication();

        if (auth instanceof JwtAuthenticationToken jwtAuth) {
            return jwtAuth.getToken();
        }

        throw new IllegalStateException("Current authentication is not JWT-based");
    }

    public static String currentUserId() {
        return currentJwt().getSubject();
    }

    public static String currentUsername() {
        return currentJwt().getClaimAsString("username");
    }
}
