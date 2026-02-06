package de.serbroda.ragbag.controller;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;
import static de.serbroda.ragbag.security.DomainPermissionEvaluator.DOMAIN_PREFIX_COLELCTION;

import de.serbroda.ragbag.generated.api.CollectionsApi;
import de.serbroda.ragbag.generated.model.CollectionDto;
import de.serbroda.ragbag.generated.model.CreateCollectionDto;
import de.serbroda.ragbag.generated.model.UpdateCollectionDto;
import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.service.CollectionService;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class CollectionController implements CollectionsApi {

    private final CollectionService collectionService;

    @Override
    public ResponseEntity<CollectionDto> createCollection(String spaceId, CreateCollectionDto createCollectionDto) {
        return CollectionsApi.super.createCollection(spaceId, createCollectionDto);
    }

    @Override
    public ResponseEntity<Void> deleteCollection(String collectionId) {
        return CollectionsApi.super.deleteCollection(collectionId);
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'READ')")
    @Override
    public ResponseEntity<CollectionDto> getCollection(String collectionId) {
        Collection collection = collectionService.getCollection(collectionId);
        return ResponseEntity.ok(new CollectionDto.Builder()
                .id(collection.getId())
                .name(collection.getName())
                .description(collection.getDescription())
                .build());
    }

    @Override
    public ResponseEntity<List<CollectionDto>> getCollections(String spaceId) {
        List<CollectionDto> collections =
                collectionService.getAllowedCollectionTree(spaceId, SecurityUtils.currentUserId());
        return ResponseEntity.ok(collections);
    }

    @Override
    public ResponseEntity<CollectionDto> updateCollection(
            String collectionId, UpdateCollectionDto updateCollectionDto) {
        return CollectionsApi.super.updateCollection(collectionId, updateCollectionDto);
    }
}
