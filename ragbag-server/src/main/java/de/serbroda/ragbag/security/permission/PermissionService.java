package de.serbroda.ragbag.security.permission;

import de.serbroda.ragbag.bookmark.BookmarkRepository;
import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.SpaceMember;
import de.serbroda.ragbag.space.SpaceMemberRepository;
import de.serbroda.ragbag.space.SpacePermission;
import de.serbroda.ragbag.space.SpaceRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class PermissionService {

    private final CollectionRepository collectionRepository;
    private final SpaceRepository spaceRepository;
    private final SpaceMemberRepository spaceMemberRepository;
    private final BookmarkRepository bookmarkRepository;

    public boolean hasPermissionForBookmark(String userId, String bookmarkId, SpacePermission permission) {
        String collectionId = bookmarkRepository
                .findCollectionIdById(bookmarkId)
                .orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + bookmarkId));
        return hasPermissionForCollection(userId, collectionId, permission);
    }

    public boolean hasPermissionForCollection(String userId, String collectionId, SpacePermission permission) {
        Collection collection = collectionRepository
                .findById(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));

        return hasPermissionForSpace(userId, collection.getSpace().getId(), permission);
    }

    public boolean hasPermissionForSpace(String userId, String spaceId, SpacePermission permission) {
        if (!spaceRepository.existsById(spaceId)) {
            throw new ResourceNotFoundException("Space not found: " + spaceId);
        }

        return spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .map(SpaceMember::getRole)
                .map(role -> role.has(permission))
                .orElse(false);
    }
}
