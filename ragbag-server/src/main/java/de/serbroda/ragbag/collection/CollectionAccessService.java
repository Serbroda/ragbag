package de.serbroda.ragbag.collection;

import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Decides who can see private (restricted) collections. A collection is visible if every restricted collection
 * on its way up to the root lists the user; space roles only decide what visible members may do.
 */
@Transactional(readOnly = true)
@RequiredArgsConstructor
@Service
public class CollectionAccessService {

    private final CollectionRepository collectionRepository;

    public boolean canSee(Collection collection, String userId) {
        for (Collection current = collection; current != null; current = current.getParent()) {
            if (current.isRestricted() && !current.getMemberIds().contains(userId)) {
                return false;
            }
        }
        return true;
    }

    /** Ids of all collections of the space that are hidden from the user, including those below private ones. */
    public Set<String> hiddenCollectionIds(String spaceId, String userId) {
        if (!collectionRepository.existsBySpace_IdAndRestrictedTrue(spaceId)) {
            return Set.of();
        }
        List<Collection> collections = collectionRepository.findBySpace_Id(spaceId);
        Map<String, Boolean> visible = new HashMap<>();
        Set<String> hidden = new HashSet<>();
        for (Collection collection : collections) {
            if (!isVisible(collection, userId, visible)) {
                hidden.add(collection.getId());
            }
        }
        return hidden;
    }

    private boolean isVisible(Collection collection, String userId, Map<String, Boolean> cache) {
        Boolean known = cache.get(collection.getId());
        if (known != null) {
            return known;
        }
        boolean visible =
                (!collection.isRestricted() || collection.getMemberIds().contains(userId))
                        && (collection.getParent() == null || isVisible(collection.getParent(), userId, cache));
        cache.put(collection.getId(), visible);
        return visible;
    }
}
