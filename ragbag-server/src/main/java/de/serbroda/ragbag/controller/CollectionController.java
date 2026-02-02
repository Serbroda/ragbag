package de.serbroda.ragbag.controller;

import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.dto.CollectionDto;
import de.serbroda.ragbag.service.CollectionService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/collections")
public class CollectionController {

    private final CollectionService collectionService;

    @PreAuthorize("hasPermission(#collectionId, 'COLLECTION', 'READ')")
    @GetMapping("/{collectionId}")
    public ResponseEntity<CollectionDto> getCollection(@PathVariable String collectionId) {
        Collection collection = collectionService.getCollection(collectionId);
        return ResponseEntity.ok(new CollectionDto(
                collection.getId(),
                collection.getName(),
                collection.getDescription()
        ));
    }
}
