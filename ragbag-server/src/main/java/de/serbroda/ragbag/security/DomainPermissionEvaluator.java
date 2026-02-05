package de.serbroda.ragbag.security;

import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.shared.Permission;
import java.io.Serializable;
import lombok.RequiredArgsConstructor;
import org.springframework.security.access.PermissionEvaluator;
import org.springframework.security.core.Authentication;
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

        UserPrincipal principal = (UserPrincipal) authentication.getPrincipal();

        if (targetDomainObject instanceof Collection collection) {
            return permissionService.hasPermission(
                    principal.getUserId(), collection.getId(), Permission.valueOf(permission.toString()));
        }

        return false;
    }

    @Override
    public boolean hasPermission(
            Authentication authentication, Serializable targetId, String targetType, Object permission) {
        if (authentication == null) {
            return false;
        }

        UserPrincipal principal = (UserPrincipal) authentication.getPrincipal();

        if ("COLLECTION".equalsIgnoreCase(targetType)) {
            return permissionService.hasPermission(
                    principal.getUserId(), targetId.toString(), Permission.valueOf(permission.toString()));
        }

        return false;
    }
}
