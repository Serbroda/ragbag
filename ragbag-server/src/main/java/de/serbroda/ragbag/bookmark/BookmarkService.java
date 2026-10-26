package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionAccessService;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.shared.http.SafeHttpClient;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
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
    private final BookmarkFavoriteRepository favoriteRepository;
    private final CollectionAccessService collectionAccessService;

    public List<Bookmark> getBookmarksByCollection(String collectionId) {
        return bookmarkRepository.findByCollection_IdOrderByCreatedAtDesc(collectionId);
    }

    public List<Bookmark> searchInSpace(String spaceId, String userId, String query) {
        if (query == null || query.isBlank()) {
            return List.of();
        }
        // Treat LIKE wildcards in user input literally
        String escaped = query.trim().replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_");
        return bookmarkRepository.searchInSpace(spaceId, escaped, hiddenCollections(spaceId, userId));
    }

    /** Favorites are capped so a dashboard section stays fast even for heavy users. */
    public static final int MAX_FAVORITES_LISTED = 200;

    public List<Bookmark> getFavoritesInSpace(String userId, String spaceId) {
        return favoriteRepository.findFavoritesInSpace(
                userId, spaceId, hiddenCollections(spaceId, userId), Limit.of(MAX_FAVORITES_LISTED));
    }

    public void addFavorite(String userId, String bookmarkId) {
        BookmarkFavorite.Id id = new BookmarkFavorite.Id(userId, bookmarkId);
        if (!favoriteRepository.existsById(id)) {
            favoriteRepository.save(new BookmarkFavorite(userId, bookmarkId));
        }
    }

    public void removeFavorite(String userId, String bookmarkId) {
        favoriteRepository.deleteById(new BookmarkFavorite.Id(userId, bookmarkId));
    }

    public List<Bookmark> getRecentInSpace(String spaceId, String userId, int limit) {
        return bookmarkRepository.findByCollection_Space_IdAndCollection_IdNotInOrderByCreatedAtDesc(
                spaceId, hiddenCollections(spaceId, userId), Limit.of(Math.clamp(limit, 1, 100)));
    }

    /** Private collections the user cannot see; never empty, as an empty NOT IN list is not portable SQL. */
    private java.util.Collection<String> hiddenCollections(String spaceId, String userId) {
        Set<String> hidden = collectionAccessService.hiddenCollectionIds(spaceId, userId);
        return hidden.isEmpty() ? List.of("") : hidden;
    }

    /** Normalizes the URL and loads its metadata without persisting anything. */
    public FetchedMetadata fetchMetadata(String url) {
        String normalized = SafeHttpClient.requireHttpUrl(url);
        return new FetchedMetadata(normalized, bookmarkMetadataService.fetch(normalized));
    }

    public record FetchedMetadata(String url, BookmarkMetadataService.BookmarkMetadata metadata) {}

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

        String url = SafeHttpClient.requireHttpUrl(cmd.url());
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
        String url = SafeHttpClient.requireHttpUrl(cmd.url());
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
