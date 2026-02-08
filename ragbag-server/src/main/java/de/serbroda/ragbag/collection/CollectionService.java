package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.space.SpaceMemberRepository;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class CollectionService {

    private static final String ROOT = "__ROOT__";

    private final CollectionRepository collectionRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public Optional<Collection> getCollection(String id) {
        return collectionRepository.findById(id);
    }

    public List<Collection> getCollectionsBySpace(String spaceId) {
        return collectionRepository.findBySpace_Id(spaceId);
    }

    public Collection createCollection(Collection collection) {
        return collectionRepository.save(collection);
    }

    public List<CollectionNode> getAllowedCollectionTree(String spaceId, String userId) {

        boolean isMember = spaceMemberRepository.existsBySpace_IdAndUser_Id(spaceId, userId);

        if (!isMember) {
            return List.of();
        }

        List<Collection> collections = collectionRepository.findBySpace_Id(spaceId);

        Map<String, List<Collection>> byParent = collections.stream()
                .collect(Collectors.groupingBy(
                        c -> c.getParent() != null ? c.getParent().getId() : ROOT));

        return buildTree(ROOT, byParent);
    }

    private List<CollectionNode> buildTree(String parentId, Map<String, List<Collection>> byParent) {

        return byParent.getOrDefault(parentId, List.of()).stream()
                .map(collection -> new CollectionNode(
                        collection.getId(),
                        collection.getName(),
                        collection.getDescription(),
                        ROOT.equals(parentId) ? null : parentId,
                        buildTree(collection.getId(), byParent)))
                .toList();
    }

    public record CollectionNode(
            String id, String name, String description, String parentId, List<CollectionNode> children) {}
}
