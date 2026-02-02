package de.serbroda.ragbag.security;

import de.serbroda.ragbag.exception.CollectionNotFoundException;
import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.shared.CollectionVisibility;
import de.serbroda.ragbag.model.shared.Permission;
import de.serbroda.ragbag.model.shared.SpaceMemberRole;
import de.serbroda.ragbag.repository.CollectionRepository;
import de.serbroda.ragbag.repository.SpaceMemberRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class PermissionService {

    private final CollectionRepository collectionRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public boolean hasPermission(
            String userId,
            String collectionId,
            Permission permission
    ) {
        Collection collection = collectionRepository
                .findById(collectionId)
                .orElseThrow(() -> new CollectionNotFoundException(collectionId));

        Space space = collection.getSpace();

        SpaceMemberRole role = spaceMemberRepository
                .findBySpaceAndUser_Id(space, userId)
                .map(SpaceMember::getRole)
                .orElse(null);

        if (role == null) {
            return false;
        }

        return switch (permission) {
            case READ -> canRead(role, collection);
            case WRITE -> canWrite(role, collection);
            case DELETE -> role == SpaceMemberRole.ADMIN;// || role == SpaceMemberRole.OWNER;
            //case CREATE_BOOKMARK -> role != SpaceMemberRole.VIEWER;
        };
    }

    private boolean canRead(SpaceMemberRole role, Collection c) {
        return switch (c.getVisibility()) {
            case PUBLIC -> true;
            case INTERNAL -> role != SpaceMemberRole.VIEWER;
            case PRIVATE -> role == SpaceMemberRole.ADMIN;// || role == SpaceMemberRole.OWNER;
        };
    }

    private boolean canWrite(SpaceMemberRole role, Collection c) {
        return role == SpaceMemberRole.ADMIN || (
                c.getVisibility() == CollectionVisibility.INTERNAL
                        && role == SpaceMemberRole.CONTRIBUTOR
        );// || role == SpaceRole.OWNER;
    }
}
