package de.serbroda.ragbag.security;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.Set;
import org.springframework.web.filter.OncePerRequestFilter;

public class ApiKeyFilter extends OncePerRequestFilter {

    public static final String ACTUATOR_API_KEY_HEADER = "X-Api-Key";

    private final Set<String> expectedApiKey;

    public ApiKeyFilter(Set<String> expectedApiKey) {
        this.expectedApiKey = expectedApiKey != null ? expectedApiKey : Set.of();
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain)
            throws ServletException, IOException {

        String providedKey = request.getHeader(ACTUATOR_API_KEY_HEADER);
        for (String key : expectedApiKey) {
            if (key.equals(providedKey)) {
                filterChain.doFilter(request, response);
                return;
            }
        }
        response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
    }
}
