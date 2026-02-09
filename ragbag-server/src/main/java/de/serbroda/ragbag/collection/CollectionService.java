package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.SpaceMemberRepository;
import de.serbroda.ragbag.space.SpaceRepository;
import de.serbroda.ragbag.user.UserService;
import jakarta.transaction.Transactional;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Transactional
@RequiredArgsConstructor
@Service
public class CollectionService {

    private static final String ROOT = "__ROOT__";

    private final CollectionRepository collectionRepository;
    private final SpaceMemberRepository spaceMemberRepository;
    private final SpaceRepository spaceRepository;
    private final UserService userService;

    public Optional<Collection> getCollection(String id) {
        return collectionRepository.findById(id);
    }

    public List<Collection> getCollectionsBySpace(String spaceId) {
        return collectionRepository.findBySpace_Id(spaceId);
    }

    public Collection createCollection(String userId, CreateCollectionCommand cmd) {
        var user = userService
                .findUserById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found: " + userId));
        var space = spaceRepository
                .findById(cmd.spaceId())
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + cmd.spaceId()));

        Collection collection = new Collection();
        collection.setName(cmd.name());
        collection.setSpace(space);
        collection.setCreatedBy(user);

        if (cmd.parentId() != null) {
            Collection parent = collectionRepository
                    .findById(cmd.parentId())
                    .orElseThrow(() -> new ResourceNotFoundException("Parent collection not found: " + cmd.parentId()));
            collection.setParent(parent);
        }

        return collectionRepository.save(collection);
    }

    public Collection updateCollection(String id, UpdateCollectionCommand cmd) {
        Collection collection = collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));
        collection.setName(cmd.name());
        return collectionRepository.save(collection);
    }

    public Collection moveCollection(String id, MoveCollectionCommand cmd) {
        Collection collection = collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));

        if (cmd.parentId() != null) {
            if (cmd.parentId().equals(id)) {
                throw new IllegalArgumentException("A collection cannot be its own parent");
            }
            Collection newParent = collectionRepository
                    .findById(cmd.parentId())
                    .orElseThrow(() -> new ResourceNotFoundException("Parent collection not found: " + cmd.parentId()));
            collection.setParent(newParent);
        } else {
            collection.setParent(null);
        }

        return collectionRepository.save(collection);
    }

    public void deleteCollection(String id) {
        collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));
        collectionRepository.deleteById(id);
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

    public record CreateCollectionCommand(String spaceId, String name, String parentId) {
        public CreateCollectionCommand(String spaceId, String name) {
            this(spaceId, name, null);
        }
    }

    public record UpdateCollectionCommand(String name) {}

    public record MoveCollectionCommand(String parentId) {}
}
