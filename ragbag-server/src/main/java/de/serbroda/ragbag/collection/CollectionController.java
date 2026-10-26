package de.serbroda.ragbag.collection;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_COLLECTION;
import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.CollectionApi;
import de.serbroda.ragbag.generated.model.CollectionAccessDto;
import de.serbroda.ragbag.generated.model.CollectionAccessMemberDto;
import de.serbroda.ragbag.generated.model.CollectionDto;
import de.serbroda.ragbag.generated.model.CreateCollectionDto;
import de.serbroda.ragbag.generated.model.MoveCollectionDto;
import de.serbroda.ragbag.generated.model.UpdateCollectionAccessDto;
import de.serbroda.ragbag.generated.model.UpdateCollectionDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class CollectionController implements CollectionApi {

    private final CollectionService collectionService;
    private final CollectionMapper mapper;

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'CREATE_COLLECTIONS')")
    @Override
    public ResponseEntity<CollectionDto> createCollection(String spaceId, CreateCollectionDto createCollectionDto) {
        Collection collection = collectionService.createCollection(
                SecurityUtils.currentUserId(),
                new CollectionService.CreateCollectionCommand(
                        spaceId,
                        createCollectionDto.getName(),
                        createCollectionDto.getParentId(),
                        Boolean.TRUE.equals(createCollectionDto.getRestricted()),
                        createCollectionDto.getMemberIds()));
        return ResponseEntity.ok(toDto(collection));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'DELETE_COLLECTIONS')")
    @Override
    public ResponseEntity<Void> deleteCollection(String collectionId) {
        collectionService.deleteCollection(SecurityUtils.currentUserId(), collectionId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'READ')")
    @Override
    public ResponseEntity<CollectionDto> getCollection(String collectionId) {
        Collection collection = collectionService
                .getCollection(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
        return ResponseEntity.ok(toDto(collection));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<List<CollectionDto>> getCollections(String spaceId) {
        List<CollectionDto> collections =
                collectionService.getAllowedCollectionTree(spaceId, SecurityUtils.currentUserId()).stream()
                        .map(mapper::toDto)
                        .toList();
        return ResponseEntity.ok(collections);
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'EDIT_COLLECTIONS')")
    @Override
    public ResponseEntity<CollectionDto> updateCollection(
            String collectionId, UpdateCollectionDto updateCollectionDto) {
        Collection collection = collectionService.updateCollection(
                collectionId, new CollectionService.UpdateCollectionCommand(updateCollectionDto.getName()));
        return ResponseEntity.ok(toDto(collection));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'EDIT_COLLECTIONS')")
    @Override
    public ResponseEntity<CollectionDto> moveCollection(String collectionId, MoveCollectionDto moveCollectionDto) {
        Collection collection = collectionService.moveCollection(
                SecurityUtils.currentUserId(),
                collectionId,
                new CollectionService.MoveCollectionCommand(moveCollectionDto.getParentId()));
        return ResponseEntity.ok(toDto(collection));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'READ')")
    @Override
    public ResponseEntity<CollectionAccessDto> getCollectionAccess(String collectionId) {
        return ResponseEntity.ok(toDto(collectionService.getAccess(SecurityUtils.currentUserId(), collectionId)));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'EDIT_COLLECTIONS')")
    @Override
    public ResponseEntity<CollectionAccessDto> updateCollectionAccess(
            String collectionId, UpdateCollectionAccessDto updateCollectionAccessDto) {
        return ResponseEntity.ok(toDto(collectionService.updateAccess(
                SecurityUtils.currentUserId(),
                collectionId,
                Boolean.TRUE.equals(updateCollectionAccessDto.getRestricted()),
                updateCollectionAccessDto.getMemberIds())));
    }

    private static CollectionAccessDto toDto(CollectionService.CollectionAccess access) {
        return new CollectionAccessDto.Builder()
                .restricted(access.restricted())
                .canManage(access.canManage())
                .members(access.members().stream()
                        .map(user -> new CollectionAccessMemberDto.Builder()
                                .userId(user.getId())
                                .username(user.getUsername())
                                .avatarId(user.getAvatarImageId())
                                .build())
                        .toList())
                .build();
    }

    private static CollectionDto toDto(Collection collection) {
        return new CollectionDto.Builder()
                .id(collection.getId())
                .name(collection.getName())
                .description(collection.getDescription())
                .parentId(
                        collection.getParent() != null ? collection.getParent().getId() : null)
                .restricted(collection.isRestricted())
                .build();
    }
}
