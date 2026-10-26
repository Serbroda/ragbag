package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.bookmark.Bookmark;
import de.serbroda.ragbag.bookmark.BookmarkRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.shared.http.SafeHttpClient;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.ObjectProvider;
import org.springframework.boot.info.BuildProperties;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.interceptor.TransactionAspectSupport;
import tools.jackson.databind.json.JsonMapper;

/** Exports collections as versioned JSON files and imports them, see {@link CollectionExchange}. */
@Transactional
@RequiredArgsConstructor
@Service
public class CollectionExchangeService {

    static final int MAX_COLLECTIONS = 1_000;
    static final int MAX_LINKS = 20_000;
    static final int MAX_DEPTH = 32;

    private static final int MAX_NAME_LENGTH = 80;
    private static final int MAX_TITLE_LENGTH = 160;

    private final CollectionRepository collectionRepository;
    private final CollectionAccessService accessService;
    private final BookmarkRepository bookmarkRepository;
    private final UserRepository userRepository;
    private final JsonMapper jsonMapper;
    private final ObjectProvider<BuildProperties> buildProperties;

    /** The collection as the user sees it: private sub-collections of others are left out. */
    @Transactional(readOnly = true)
    public CollectionExchange export(String userId, String collectionId) {
        Collection collection = requireCollection(collectionId);
        Set<String> hidden =
                accessService.hiddenCollectionIds(collection.getSpace().getId(), userId);
        return new CollectionExchange(
                CollectionExchange.FORMAT,
                CollectionExchange.FORMAT_VERSION,
                appVersion(),
                Instant.now(),
                toNode(collection, hidden));
    }

    /**
     * Imports a file into the collection: its links are added there, its sub-collections are merged with
     * existing ones of the same name. As preview, everything runs and is then rolled back, so the result tells
     * exactly what the import would do.
     */
    public ImportResult importInto(String userId, String collectionId, Object body, boolean preview) {
        CollectionExchange file = parse(body);
        Collection target = requireCollection(collectionId);
        User user = userRepository
                .findById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found: " + userId));
        Set<String> hidden = accessService.hiddenCollectionIds(target.getSpace().getId(), userId);

        Counter counter = new Counter();
        if (file.collection() != null) {
            checkLimits(file.collection(), 0, counter);
            importNode(target, file.collection(), user, hidden, counter);
        }
        if (preview) {
            TransactionAspectSupport.currentTransactionStatus().setRollbackOnly();
        }
        return new ImportResult(
                preview,
                file.formatVersion(),
                file.appVersion(),
                counter.collectionsCreated,
                counter.collectionsMerged,
                counter.linksImported,
                counter.linksSkipped,
                counter.linksInvalid);
    }

    /** Reads a file of any known format version and converts it to the current one, like migrations. */
    CollectionExchange parse(Object body) {
        if (!(body instanceof Map<?, ?> header)) {
            throw new IllegalArgumentException("Not a Ragbag collection export");
        }
        if (!CollectionExchange.FORMAT.equals(header.get("format"))) {
            throw new IllegalArgumentException("Not a Ragbag collection export");
        }
        if (!(header.get("format_version") instanceof Number version) || version.intValue() < 1) {
            throw new IllegalArgumentException("The export file has no valid format_version");
        }
        if (version.intValue() > CollectionExchange.FORMAT_VERSION) {
            throw new IllegalArgumentException("The file was exported by a newer version of Ragbag (format "
                    + version.intValue() + "), please update first");
        }
        // Version 1 is the current one; converters for later versions go here (v1 → v2, …)
        try {
            return jsonMapper.convertValue(body, CollectionExchange.class);
        } catch (RuntimeException e) {
            throw new IllegalArgumentException("The export file is invalid: " + e.getMessage(), e);
        }
    }

    private CollectionExchange.Node toNode(Collection collection, Set<String> hidden) {
        // Oldest first: the import creates links in file order, so the newest one stays the newest
        List<CollectionExchange.Link> links =
                bookmarkRepository.findByCollection_IdOrderByCreatedAtDesc(collection.getId()).reversed().stream()
                        .map(bookmark -> new CollectionExchange.Link(
                                bookmark.getUrl(),
                                bookmark.getTitle(),
                                bookmark.getDescription(),
                                bookmark.getTags() != null
                                        ? bookmark.getTags().stream()
                                                .sorted(String.CASE_INSENSITIVE_ORDER)
                                                .toList()
                                        : List.of(),
                                bookmark.getOgImage(),
                                bookmark.getFavicon(),
                                bookmark.getCanonical(),
                                bookmark.getCreatedAt()))
                        .toList();
        List<CollectionExchange.Node> children = collection.getChildren().stream()
                .filter(child -> !hidden.contains(child.getId()))
                .sorted(Comparator.comparing(Collection::getName, String.CASE_INSENSITIVE_ORDER))
                .map(child -> toNode(child, hidden))
                .toList();
        return new CollectionExchange.Node(
                collection.getName(), collection.getDescription(), collection.isRestricted(), links, children);
    }

    private void checkLimits(CollectionExchange.Node node, int depth, Counter counter) {
        if (depth > MAX_DEPTH) {
            throw new IllegalArgumentException("Collections are nested deeper than " + MAX_DEPTH + " levels");
        }
        counter.totalCollections++;
        counter.totalLinks += node.links() != null ? node.links().size() : 0;
        if (counter.totalCollections > MAX_COLLECTIONS || counter.totalLinks > MAX_LINKS) {
            throw new IllegalArgumentException(
                    "An import can hold at most " + MAX_COLLECTIONS + " collections and " + MAX_LINKS + " links");
        }
        for (CollectionExchange.Node child : nonNull(node.collections())) {
            checkLimits(child, depth + 1, counter);
        }
    }

    private void importNode(
            Collection target, CollectionExchange.Node node, User user, Set<String> hidden, Counter counter) {
        Set<String> existingUrls = bookmarkRepository.findByCollection_IdOrderByCreatedAtDesc(target.getId()).stream()
                .map(Bookmark::getUrl)
                .collect(Collectors.toCollection(HashSet::new));
        for (CollectionExchange.Link link : nonNull(node.links())) {
            String url;
            try {
                url = SafeHttpClient.requireHttpUrl(link.url());
            } catch (IllegalArgumentException e) {
                counter.linksInvalid++;
                continue;
            }
            if (!existingUrls.add(url)) {
                counter.linksSkipped++;
                continue;
            }
            Bookmark bookmark = new Bookmark();
            bookmark.setCollection(target);
            bookmark.setCreatedBy(user);
            bookmark.setUrl(url);
            bookmark.setTitle(
                    truncate(blankToNull(link.title()) != null ? link.title().trim() : url, MAX_TITLE_LENGTH));
            bookmark.setDescription(blankToNull(link.description()));
            bookmark.setTags(normalizeTags(link.tags()));
            // Rendered as images and links for every member, so only http(s) URLs are kept
            bookmark.setOgImage(httpUrlOrNull(link.ogImage()));
            bookmark.setFavicon(httpUrlOrNull(link.favicon()));
            bookmark.setCanonical(httpUrlOrNull(link.canonical()));
            // Original date if the file has one, otherwise strictly increasing in file order (oldest first)
            bookmark.setCreatedAt(
                    link.createdAt() != null
                            ? link.createdAt()
                            : counter.importStart.plusMillis(counter.linksImported));
            bookmarkRepository.save(bookmark);
            counter.linksImported++;
        }

        for (CollectionExchange.Node childNode : nonNull(node.collections())) {
            String name = collectionName(childNode.name());
            // Only collections the user can see are merged into; hidden ones must stay untouched
            Collection child = target.getChildren().stream()
                    .filter(existing -> !hidden.contains(existing.getId()))
                    .filter(existing -> existing.getName().equalsIgnoreCase(name))
                    .findFirst()
                    .orElse(null);
            if (child != null) {
                counter.collectionsMerged++;
            } else {
                child = new Collection();
                child.setName(name);
                child.setDescription(blankToNull(childNode.description()));
                child.setSpace(target.getSpace());
                child.setParent(target);
                child.setCreatedBy(user);
                if (Boolean.TRUE.equals(childNode.restricted())) {
                    child.setRestricted(true);
                    child.getMemberIds().add(user.getId());
                }
                child = collectionRepository.save(child);
                target.getChildren().add(child);
                counter.collectionsCreated++;
            }
            importNode(child, childNode, user, hidden, counter);
        }
    }

    private Collection requireCollection(String collectionId) {
        return collectionRepository
                .findById(collectionId)
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + collectionId));
    }

    private String appVersion() {
        BuildProperties properties = buildProperties.getIfAvailable();
        return properties != null ? properties.getVersion() : "dev";
    }

    private static String collectionName(String name) {
        String trimmed = blankToNull(name) != null ? name.trim() : "Imported";
        return trimmed.length() > MAX_NAME_LENGTH ? trimmed.substring(0, MAX_NAME_LENGTH) : trimmed;
    }

    private static String httpUrlOrNull(String value) {
        if (blankToNull(value) == null || !value.contains("://")) {
            return null;
        }
        try {
            return SafeHttpClient.requireHttpUrl(value);
        } catch (IllegalArgumentException e) {
            return null;
        }
    }

    private static Set<String> normalizeTags(List<String> tags) {
        Set<String> normalized = new LinkedHashSet<>();
        for (String tag : nonNull(tags)) {
            if (tag != null && !tag.isBlank()) {
                normalized.add(tag.trim());
            }
        }
        return normalized;
    }

    private static String blankToNull(String value) {
        return value == null || value.isBlank() ? null : value;
    }

    private static String truncate(String value, int maxLength) {
        return value.length() <= maxLength ? value : value.substring(0, maxLength - 1) + "…";
    }

    private static <T> List<T> nonNull(List<T> list) {
        return list != null ? list.stream().filter(Objects::nonNull).toList() : new ArrayList<>();
    }

    private static final class Counter {
        final Instant importStart = Instant.now();
        int totalCollections;
        int totalLinks;
        int collectionsCreated;
        int collectionsMerged;
        int linksImported;
        int linksSkipped;
        int linksInvalid;
    }

    public record ImportResult(
            boolean preview,
            int formatVersion,
            String appVersion,
            int collectionsCreated,
            int collectionsMerged,
            int linksImported,
            int linksSkipped,
            int linksInvalid) {}
}
