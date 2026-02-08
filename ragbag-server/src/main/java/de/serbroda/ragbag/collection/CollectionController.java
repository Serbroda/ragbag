package de.serbroda.ragbag.collection;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_COLELCTION;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.CollectionApi;
import de.serbroda.ragbag.generated.model.CollectionDto;
import de.serbroda.ragbag.generated.model.CreateCollectionDto;
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

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'WRITE')")
    @Override
    public ResponseEntity<CollectionDto> createCollection(String spaceId, CreateCollectionDto createCollectionDto) {
        return CollectionApi.super.createCollection(spaceId, createCollectionDto);
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'DELETE')")
    @Override
    public ResponseEntity<Void> deleteCollection(String collectionId) {
        return CollectionApi.super.deleteCollection(collectionId);
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'READ')")
    @Override
    public ResponseEntity<CollectionDto> getCollection(String collectionId) {
        Collection collection = collectionService
                .getCollection(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
        return ResponseEntity.ok(new CollectionDto.Builder()
                .id(collection.getId())
                .name(collection.getName())
                .description(collection.getDescription())
                .build());
    }

    @Override
    public ResponseEntity<List<CollectionDto>> getCollections(String spaceId) {
        List<CollectionDto> collections =
                collectionService.getAllowedCollectionTree(spaceId, SecurityUtils.currentUserId()).stream()
                        .map(mapper::toDto)
                        .toList();
        return ResponseEntity.ok(collections);
    }

    @Override
    public ResponseEntity<CollectionDto> updateCollection(
            String collectionId, UpdateCollectionDto updateCollectionDto) {
        return CollectionApi.super.updateCollection(collectionId, updateCollectionDto);
    }
}
