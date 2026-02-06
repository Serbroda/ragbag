package de.serbroda.ragbag.security;

import de.serbroda.ragbag.exception.CollectionNotFoundException;
import de.serbroda.ragbag.exception.SpaceNotFoundException;
import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.shared.CollectionVisibility;
import de.serbroda.ragbag.model.shared.Permission;
import de.serbroda.ragbag.model.shared.SpaceMemberRole;
import de.serbroda.ragbag.repository.CollectionRepository;
import de.serbroda.ragbag.repository.SpaceMemberRepository;
import de.serbroda.ragbag.repository.SpaceRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class PermissionService {

    private final CollectionRepository collectionRepository;
    private final SpaceRepository spaceRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public boolean hasPermission(String userId, String collectionId, Permission permission) {
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
            case DELETE -> role == SpaceMemberRole.ADMIN;
        };
    }

    public boolean hasPermissionForSpace(String userId, String spaceId, Permission permission) {
        Space space = spaceRepository.findById(spaceId).orElseThrow(() -> new SpaceNotFoundException(spaceId));

        SpaceMemberRole role = spaceMemberRepository
                .findBySpaceAndUser_Id(space, userId)
                .map(SpaceMember::getRole)
                .orElse(null);

        return role != null
                && switch (permission) {
                    case READ -> true;
                    case WRITE -> role == SpaceMemberRole.ADMIN || role == SpaceMemberRole.CONTRIBUTOR;
                    case DELETE -> role == SpaceMemberRole.ADMIN;
                };
    }

    private boolean canRead(SpaceMemberRole role, Collection c) {
        return switch (c.getVisibility()) {
            case PUBLIC -> true;
            case INTERNAL -> role != SpaceMemberRole.VIEWER;
            case PRIVATE -> role == SpaceMemberRole.ADMIN; // || role == SpaceMemberRole.OWNER;
        };
    }

    private boolean canWrite(SpaceMemberRole role, Collection c) {
        return role == SpaceMemberRole.ADMIN
                || (c.getVisibility() == CollectionVisibility.INTERNAL
                        && role == SpaceMemberRole.CONTRIBUTOR); // || role == SpaceRole.OWNER;
    }
}
