package de.serbroda.ragbag.security.permission;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.SpaceMember;
import de.serbroda.ragbag.space.SpaceMemberRepository;
import de.serbroda.ragbag.space.SpaceMemberRole;
import de.serbroda.ragbag.space.SpaceRepository;
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
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));

        return hasPermissionForSpace(userId, collection.getSpace().getId(), permission);
    }

    public boolean hasPermissionForSpace(String userId, String spaceId, Permission permission) {

        spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));

        SpaceMemberRole role = spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .map(SpaceMember::getRole)
                .orElse(null);

        if (role == null) {
            return false;
        }

        return switch (permission) {
            case READ -> true; // Mitglied = lesen erlaubt
            case WRITE -> role == SpaceMemberRole.ADMIN || role == SpaceMemberRole.CONTRIBUTOR || role == SpaceMemberRole.OWNER;
            case DELETE -> role == SpaceMemberRole.OWNER;
        };
    }
}
