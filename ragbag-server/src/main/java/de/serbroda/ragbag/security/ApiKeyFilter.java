package de.serbroda.ragbag.security;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.List;
import java.util.Set;
import org.springframework.web.filter.OncePerRequestFilter;

public class ApiKeyFilter extends OncePerRequestFilter {

    public static final String ACTUATOR_API_KEY_HEADER = "X-Api-Key";

    private final List<byte[]> expectedApiKeys;

    public ApiKeyFilter(Set<String> expectedApiKeys) {
        this.expectedApiKeys = expectedApiKeys == null
                ? List.of()
                : expectedApiKeys.stream()
                        .filter(key -> key != null && !key.isBlank())
                        .map(key -> key.getBytes(StandardCharsets.UTF_8))
                        .toList();
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain filterChain)
            throws ServletException, IOException {

        String providedKey = request.getHeader(ACTUATOR_API_KEY_HEADER);
        if (providedKey != null) {
            byte[] provided = providedKey.getBytes(StandardCharsets.UTF_8);
            for (byte[] key : expectedApiKeys) {
                // Constant-time comparison to avoid leaking key prefixes through timing
                if (MessageDigest.isEqual(key, provided)) {
                    filterChain.doFilter(request, response);
                    return;
                }
            }
        }
        response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
    }
}
