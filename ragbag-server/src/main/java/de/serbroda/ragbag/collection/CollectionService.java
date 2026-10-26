package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import de.serbroda.ragbag.shared.exception.ResourceConflictException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.SpaceMember;
import de.serbroda.ragbag.space.SpaceMemberRepository;
import de.serbroda.ragbag.space.SpaceMemberRole;
import de.serbroda.ragbag.space.SpaceMembershipEndedEvent;
import de.serbroda.ragbag.space.SpacePermission;
import de.serbroda.ragbag.space.SpaceRepository;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import de.serbroda.ragbag.user.UserService;
import java.util.Comparator;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Transactional
@RequiredArgsConstructor
@Service
public class CollectionService {

    private static final String ROOT = "__ROOT__";

    private final CollectionRepository collectionRepository;
    private final CollectionAccessService accessService;
    private final SpaceMemberRepository spaceMemberRepository;
    private final SpaceRepository spaceRepository;
    private final UserService userService;
    private final UserRepository userRepository;

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
        collection.setName(requireName(cmd.name()));
        collection.setSpace(space);
        collection.setCreatedBy(user);

        if (cmd.parentId() != null) {
            Collection parent = requireVisible(cmd.parentId(), userId);
            requireSameSpace(parent, space.getId());
            collection.setParent(parent);
        }
        if (cmd.restricted()) {
            collection.setRestricted(true);
            collection.setMemberIds(requireSpaceMembers(space.getId(), cmd.memberIds(), userId));
        }

        return collectionRepository.save(collection);
    }

    public Collection updateCollection(String id, UpdateCollectionCommand cmd) {
        Collection collection = collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));
        collection.setName(requireName(cmd.name()));
        return collectionRepository.save(collection);
    }

    public Collection moveCollection(String userId, String id, MoveCollectionCommand cmd) {
        Collection collection = collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));

        if (cmd.parentId() != null) {
            if (cmd.parentId().equals(id)) {
                throw new IllegalArgumentException("A collection cannot be its own parent");
            }
            Collection newParent = requireVisible(cmd.parentId(), userId);
            requireSameSpace(newParent, collection.getSpace().getId());
            for (Collection c = newParent; c != null; c = c.getParent()) {
                if (c.getId().equals(id)) {
                    throw new IllegalArgumentException("A collection cannot be moved into its own descendant");
                }
            }
            collection.setParent(newParent);
        } else {
            collection.setParent(null);
        }

        return collectionRepository.save(collection);
    }

    public void deleteCollection(String userId, String id) {
        Collection collection = collectionRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + id));
        // Deleting would silently take other members' private collections with it
        Set<String> hidden =
                accessService.hiddenCollectionIds(collection.getSpace().getId(), userId);
        if (!hidden.isEmpty() && containsAny(collection, hidden)) {
            throw new ResourceConflictException(
                    "This collection contains private collections of other members and cannot be deleted");
        }
        collectionRepository.delete(collection);
    }

    @Transactional(readOnly = true)
    public CollectionAccess getAccess(String userId, String collectionId) {
        Collection collection = collectionRepository
                .findById(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
        List<User> members = userRepository.findAllById(collection.getMemberIds()).stream()
                .sorted(Comparator.comparing(User::getUsername))
                .toList();
        return new CollectionAccess(collection.isRestricted(), members, canManageAccess(userId, collection));
    }

    public CollectionAccess updateAccess(
            String userId, String collectionId, boolean restricted, List<String> memberIds) {
        Collection collection = collectionRepository
                .findById(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
        if (!canManageAccess(userId, collection)) {
            throw new AccessDeniedException(
                    "Only the creator of the collection or space admins can change who sees it");
        }
        collection.setRestricted(restricted);
        collection.getMemberIds().clear();
        if (restricted) {
            collection
                    .getMemberIds()
                    .addAll(requireSpaceMembers(collection.getSpace().getId(), memberIds, userId));
        }
        collectionRepository.save(collection);
        return getAccess(userId, collectionId);
    }

    /**
     * Private collections are managed by the people who can see them. Making a shared collection private hides it
     * from everybody else, so only its creator or a space admin may do that.
     */
    private boolean canManageAccess(String userId, Collection collection) {
        SpaceMemberRole role = spaceMemberRepository
                .findBySpace_IdAndUser_Id(collection.getSpace().getId(), userId)
                .map(SpaceMember::getRole)
                .orElse(null);
        if (role == null || !role.has(SpacePermission.EDIT_COLLECTIONS)) {
            return false;
        }
        return collection.isRestricted()
                || collection.getCreatedBy().getId().equals(userId)
                || role.has(SpacePermission.EDIT_SPACE);
    }

    /** Former members lose access; private collections nobody can see anymore are deleted with them. */
    @EventListener
    public void onMembershipEnded(SpaceMembershipEndedEvent event) {
        for (Collection collection :
                collectionRepository.findRestrictedBySpaceAndMember(event.spaceId(), event.userId())) {
            collection.getMemberIds().remove(event.userId());
            if (collection.getMemberIds().isEmpty()) {
                if (collection.getParent() != null) {
                    collection.getParent().getChildren().remove(collection);
                }
                collectionRepository.delete(collection);
            }
        }
    }

    private Collection requireVisible(String collectionId, String userId) {
        return collectionRepository
                .findById(collectionId)
                .filter(collection -> accessService.canSee(collection, userId))
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
    }

    /** The given users plus the acting user, who must never lock themselves out. */
    private Set<String> requireSpaceMembers(String spaceId, List<String> userIds, String actingUserId) {
        Set<String> ids = new HashSet<>(userIds != null ? userIds : List.of());
        ids.add(actingUserId);
        for (String id : ids) {
            if (!spaceMemberRepository.existsBySpace_IdAndUser_Id(spaceId, id)) {
                throw new IllegalArgumentException("Only members of the space can be given access: " + id);
            }
        }
        return ids;
    }

    private static boolean containsAny(Collection collection, Set<String> ids) {
        if (ids.contains(collection.getId())) {
            return true;
        }
        return collection.getChildren().stream().anyMatch(child -> containsAny(child, ids));
    }

    private static String requireName(String name) {
        if (name == null || name.isBlank()) {
            throw new IllegalArgumentException("Collection name must not be empty");
        }
        String trimmed = name.trim();
        if (trimmed.length() > 80) {
            throw new IllegalArgumentException("Collection name must be at most 80 characters");
        }
        return trimmed;
    }

    private static void requireSameSpace(Collection parent, String spaceId) {
        if (!parent.getSpace().getId().equals(spaceId)) {
            throw new IllegalArgumentException("Parent collection belongs to a different space");
        }
    }

    public List<CollectionNode> getAllowedCollectionTree(String spaceId, String userId) {

        boolean isMember = spaceMemberRepository.existsBySpace_IdAndUser_Id(spaceId, userId);

        if (!isMember) {
            return List.of();
        }

        Set<String> hidden = accessService.hiddenCollectionIds(spaceId, userId);
        List<Collection> collections = collectionRepository.findBySpace_Id(spaceId).stream()
                .filter(collection -> !hidden.contains(collection.getId()))
                .toList();

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
                        collection.isRestricted(),
                        buildTree(collection.getId(), byParent)))
                .toList();
    }

    public record CollectionNode(
            String id,
            String name,
            String description,
            String parentId,
            boolean restricted,
            List<CollectionNode> children) {}

    public record CreateCollectionCommand(
            String spaceId, String name, String parentId, boolean restricted, List<String> memberIds) {
        public CreateCollectionCommand(String spaceId, String name) {
            this(spaceId, name, null, false, List.of());
        }
    }

    public record UpdateCollectionCommand(String name) {}

    public record MoveCollectionCommand(String parentId) {}

    public record CollectionAccess(boolean restricted, List<User> members, boolean canManage) {}
}
