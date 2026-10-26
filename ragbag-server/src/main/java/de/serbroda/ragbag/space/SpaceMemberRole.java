package de.serbroda.ragbag.space;

import java.util.EnumSet;
import java.util.Set;

public enum SpaceMemberRole {
    OWNER(EnumSet.allOf(SpacePermission.class)),
    ADMIN(EnumSet.complementOf(EnumSet.of(SpacePermission.DELETE_SPACE))),
    CONTRIBUTOR(EnumSet.of(
            SpacePermission.READ,
            SpacePermission.CREATE_COLLECTIONS,
            SpacePermission.EDIT_COLLECTIONS,
            SpacePermission.DELETE_COLLECTIONS,
            SpacePermission.CREATE_BOOKMARKS,
            SpacePermission.EDIT_BOOKMARKS,
            SpacePermission.DELETE_BOOKMARKS)),
    VIEWER(EnumSet.of(SpacePermission.READ));

    private final Set<SpacePermission> permissions;

    SpaceMemberRole(Set<SpacePermission> permissions) {
        this.permissions = Set.copyOf(permissions);
    }

    public Set<SpacePermission> permissions() {
        return permissions;
    }

    public boolean has(SpacePermission permission) {
        return permissions.contains(permission);
    }

    /** Roles are declared from most to least privileged. */
    public boolean isAtLeast(SpaceMemberRole other) {
        return ordinal() <= other.ordinal();
    }
}
