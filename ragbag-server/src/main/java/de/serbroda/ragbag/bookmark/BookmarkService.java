package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import org.springframework.data.domain.Limit;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Transactional
@RequiredArgsConstructor
@Service
public class BookmarkService {

    /** Matches the length of bookmarks.title in the database schema. */
    private static final int MAX_TITLE_LENGTH = 160;

    private final BookmarkRepository bookmarkRepository;
    private final CollectionRepository collectionRepository;
    private final UserRepository userRepository;
    private final BookmarkMetadataService bookmarkMetadataService;

    public List<Bookmark> getBookmarksByCollection(String collectionId) {
        return bookmarkRepository.findByCollection_IdOrderByCreatedAtDesc(collectionId);
    }

    public List<Bookmark> searchInSpace(String spaceId, String query) {
        if (query == null || query.isBlank()) {
            return List.of();
        }
        // Treat LIKE wildcards in user input literally
        String escaped = query.trim().replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_");
        return bookmarkRepository.searchInSpace(spaceId, escaped);
    }

    public List<Bookmark> getRecentInSpace(String spaceId, int limit) {
        return bookmarkRepository.findByCollection_Space_IdOrderByCreatedAtDesc(
                spaceId, Limit.of(Math.clamp(limit, 1, 100)));
    }

    public Bookmark getBookmark(String id) {
        return bookmarkRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
    }

    public Bookmark createBookmark(String userId, CreateBookmarkCommand cmd) {
        Collection collection = collectionRepository
                .findById(cmd.collectionId())
                .orElseThrow(() -> new ResourceNotFoundException("Collection not found: " + cmd.collectionId()));
        User user = userRepository
                .findById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found: " + userId));

        String url = requireHttpUrl(cmd.url());
        BookmarkMetadataService.BookmarkMetadata metadata = bookmarkMetadataService.fetch(url);

        Bookmark bookmark = new Bookmark();
        bookmark.setCollection(collection);
        bookmark.setCreatedBy(user);
        bookmark.setUrl(url);
        bookmark.setTitle(truncate(preferNonBlank(cmd.title(), metadata.title(), url), MAX_TITLE_LENGTH));
        bookmark.setDescription(preferNonBlank(cmd.description(), metadata.description()));
        bookmark.setOgImage(metadata.ogImage());
        bookmark.setFavicon(metadata.favicon());
        bookmark.setCanonical(metadata.canonical());
        bookmark.setTags(normalizeTags(cmd.tags()));

        return bookmarkRepository.save(bookmark);
    }

    public Bookmark updateBookmark(String id, UpdateBookmarkCommand cmd) {
        Bookmark bookmark = bookmarkRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        String url = requireHttpUrl(cmd.url());
        boolean urlChanged = !Objects.equals(url, bookmark.getUrl());
        bookmark.setUrl(url);
        bookmark.setTitle(truncate(preferNonBlank(cmd.title(), bookmark.getTitle()), MAX_TITLE_LENGTH));
        bookmark.setDescription(cmd.description());
        if (cmd.tags() != null) {
            bookmark.setTags(normalizeTags(cmd.tags()));
        }

        if (urlChanged) {
            BookmarkMetadataService.BookmarkMetadata metadata = bookmarkMetadataService.fetch(url);
            bookmark.setOgImage(metadata.ogImage());
            bookmark.setFavicon(metadata.favicon());
            bookmark.setCanonical(metadata.canonical());
            bookmark.setTitle(truncate(preferNonBlank(bookmark.getTitle(), metadata.title(), url), MAX_TITLE_LENGTH));
            bookmark.setDescription(preferNonBlank(bookmark.getDescription(), metadata.description()));
        }
        return bookmarkRepository.save(bookmark);
    }

    public void deleteBookmark(String id) {
        bookmarkRepository.findById(id).orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        bookmarkRepository.deleteById(id);
    }

    public record CreateBookmarkCommand(
            String collectionId, String url, String title, String description, List<String> tags) {}

    public record UpdateBookmarkCommand(String url, String title, String description, List<String> tags) {}

    private static String preferNonBlank(String primary, String fallback) {
        if (primary != null && !primary.isBlank()) {
            return primary;
        }
        return fallback;
    }

    private static String preferNonBlank(String primary, String fallback, String lastResort) {
        String value = preferNonBlank(primary, fallback);
        if (value != null && !value.isBlank()) {
            return value;
        }
        return lastResort;
    }

    /**
     * Only absolute http(s) URLs are stored: anything else (e.g. {@code javascript:}) would be rendered as a
     * clickable link for every member of the space.
     */
    static String requireHttpUrl(String value) {
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException("URL must not be empty");
        }
        String trimmed = value.trim();
        if (!trimmed.contains("://")) {
            trimmed = "https://" + trimmed;
        }
        try {
            URI uri = new URI(trimmed);
            String scheme = uri.getScheme();
            if (scheme == null
                    || !(scheme.equalsIgnoreCase("http") || scheme.equalsIgnoreCase("https"))
                    || uri.getHost() == null) {
                throw new IllegalArgumentException("Only http and https URLs are supported");
            }
            return trimmed;
        } catch (URISyntaxException e) {
            throw new IllegalArgumentException("Invalid URL: " + value);
        }
    }

    private static String truncate(String value, int maxLength) {
        if (value == null || value.length() <= maxLength) {
            return value;
        }
        return value.substring(0, maxLength - 1) + "…";
    }

    private static Set<String> normalizeTags(List<String> tags) {
        Set<String> normalized = new LinkedHashSet<>();
        if (tags == null) {
            return normalized;
        }
        for (String tag : tags) {
            if (tag == null) {
                continue;
            }
            String trimmed = tag.trim();
            if (!trimmed.isEmpty()) {
                normalized.add(trimmed);
            }
        }
        return normalized;
    }
}
