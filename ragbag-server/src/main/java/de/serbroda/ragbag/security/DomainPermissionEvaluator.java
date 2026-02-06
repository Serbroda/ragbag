package de.serbroda.ragbag.security;

import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.shared.Permission;
import java.io.Serializable;
import lombok.RequiredArgsConstructor;
import org.springframework.security.access.PermissionEvaluator;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationToken;
import org.springframework.stereotype.Component;

@Component
@RequiredArgsConstructor
public class DomainPermissionEvaluator implements PermissionEvaluator {

    private final PermissionService permissionService;

    @Override
    public boolean hasPermission(Authentication authentication, Object targetDomainObject, Object permission) {
        if (authentication == null || permission == null) {
            return false;
        }

        if (!(authentication instanceof JwtAuthenticationToken jwtAuth)) {
            return false;
        }

        Jwt jwt = jwtAuth.getToken();
        String userId = jwt.getClaimAsString("user_id");

        if (targetDomainObject instanceof Collection collection) {
            return permissionService.hasPermission(
                    userId, collection.getId(), Permission.valueOf(permission.toString()));
        }

        return false;
    }

    @Override
    public boolean hasPermission(
            Authentication authentication, Serializable targetId, String targetType, Object permission) {
        if (authentication == null) {
            return false;
        }

        if (!(authentication instanceof JwtAuthenticationToken jwtAuth)) {
            return false;
        }

        Jwt jwt = jwtAuth.getToken();
        String userId = jwt.getClaimAsString("user_id");

        if ("COLLECTION".equalsIgnoreCase(targetType)) {
            return permissionService.hasPermission(
                    userId, targetId.toString(), Permission.valueOf(permission.toString()));
        }

        return false;
    }
}
