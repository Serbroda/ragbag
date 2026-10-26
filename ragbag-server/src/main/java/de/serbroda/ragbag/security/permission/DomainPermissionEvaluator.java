package de.serbroda.ragbag.security.permission;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.space.SpacePermission;
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

    public static final String DOMAIN_PREFIX_COLLECTION = "COLLECTION";
    public static final String DOMAIN_PREFIX_SPACE = "SPACE";
    public static final String DOMAIN_PREFIX_BOOKMARK = "BOOKMARK";

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
        String userId = jwt.getSubject();

        if (targetDomainObject instanceof Collection collection) {
            return permissionService.hasPermissionForCollection(
                    userId, collection.getId(), SpacePermission.valueOf(permission.toString()));
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
        String userId = jwt.getSubject();

        if (DOMAIN_PREFIX_COLLECTION.equalsIgnoreCase(targetType)) {
            return permissionService.hasPermissionForCollection(
                    userId, targetId.toString(), SpacePermission.valueOf(permission.toString()));
        }

        if (DOMAIN_PREFIX_SPACE.equalsIgnoreCase(targetType)) {
            return permissionService.hasPermissionForSpace(
                    userId, targetId.toString(), SpacePermission.valueOf(permission.toString()));
        }

        if (DOMAIN_PREFIX_BOOKMARK.equalsIgnoreCase(targetType)) {
            return permissionService.hasPermissionForBookmark(
                    userId, targetId.toString(), SpacePermission.valueOf(permission.toString()));
        }

        return false;
    }
}
