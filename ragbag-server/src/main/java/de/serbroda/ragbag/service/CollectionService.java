package de.serbroda.ragbag.service;

import de.serbroda.ragbag.generated.model.CollectionDto;
import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.shared.CollectionVisibility;
import de.serbroda.ragbag.model.shared.SpaceMemberRole;
import de.serbroda.ragbag.repository.CollectionRepository;
import de.serbroda.ragbag.repository.SpaceMemberRepository;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class CollectionService {

    private static final String ROOT = "__ROOT__";

    private final CollectionRepository collectionRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public Collection getCollection(String id) {
        return collectionRepository.findById(id).orElse(null);
    }

    public List<Collection> getCollectionsBySpace(String spaceId) {
        return collectionRepository.findBySpace_Id(spaceId);
    }

    public Collection createCollection(Collection collection) {
        return collectionRepository.save(collection);
    }

    public List<CollectionDto> getAllowedCollectionTree(String spaceId, String userId) {

        // 1️⃣ Rolle des Users im Space bestimmen
        SpaceMemberRole role = spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .map(sm -> sm.getRole())
                .orElse(null);

        if (role == null) {
            return List.of(); // kein Zugriff auf den Space
        }

        // 2️⃣ Alle Collections des Spaces flach laden
        List<Collection> collections = collectionRepository.findBySpace_Id(spaceId);

        // 3️⃣ Nach parent_id gruppieren
        Map<String, List<Collection>> byParent = collections.stream()
                .collect(Collectors.groupingBy(
                        c -> c.getParent() != null ? c.getParent().getId() : ROOT));

        // 4️⃣ Tree bauen (Root = parentId null)
        return buildTree(ROOT, null, byParent, role);
    }

    /* =========================================================
    Interne Helfer
    ========================================================= */

    private List<CollectionDto> buildTree(
            String parentId,
            CollectionVisibility parentVisibility,
            Map<String, List<Collection>> byParent,
            SpaceMemberRole role) {

        return byParent.getOrDefault(parentId, List.of()).stream()
                .map(collection -> {
                    CollectionVisibility effectiveVisibility =
                            minVisibility(parentVisibility, collection.getVisibility());

                    if (!canRead(role, effectiveVisibility)) {
                        return null; // Subtree abschneiden
                    }

                    return new CollectionDto.Builder()
                            .id(collection.getId())
                            .name(collection.getName())
                            .parentId(ROOT.equals(parentId) ? null : parentId)
                            .children(buildTree(collection.getId(), effectiveVisibility, byParent, role))
                            .build();
                })
                .filter(Objects::nonNull)
                .toList();
    }

    private CollectionVisibility minVisibility(CollectionVisibility parent, CollectionVisibility own) {
        if (parent == null) {
            return own;
        }

        return parent.ordinal() < own.ordinal() ? parent : own;
    }

    private boolean canRead(SpaceMemberRole role, CollectionVisibility visibility) {
        return switch (visibility) {
            case PUBLIC -> true;
            case INTERNAL -> role != SpaceMemberRole.VIEWER;
            case PRIVATE -> role == SpaceMemberRole.ADMIN; // || role == SpaceMemberRole.OWNER;
        };
    }
}
