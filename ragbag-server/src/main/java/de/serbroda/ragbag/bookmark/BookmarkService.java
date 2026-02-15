package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import jakarta.transaction.Transactional;
import java.util.List;
import java.util.Objects;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Transactional
@RequiredArgsConstructor
@Service
public class BookmarkService {

    private final BookmarkRepository bookmarkRepository;
    private final CollectionRepository collectionRepository;
    private final UserRepository userRepository;
    private final BookmarkMetadataService bookmarkMetadataService;

    public List<Bookmark> getBookmarksByCollection(String collectionId) {
        return bookmarkRepository.findByCollection_Id(collectionId);
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

        BookmarkMetadataService.BookmarkMetadata metadata = bookmarkMetadataService.fetch(cmd.url());

        Bookmark bookmark = new Bookmark();
        bookmark.setCollection(collection);
        bookmark.setCreatedBy(user);
        bookmark.setUrl(cmd.url());
        bookmark.setTitle(preferNonBlank(cmd.title(), metadata.title(), cmd.url()));
        bookmark.setDescription(preferNonBlank(cmd.description(), metadata.description()));
        bookmark.setOgImage(metadata.ogImage());
        bookmark.setFavicon(metadata.favicon());
        bookmark.setCanonical(metadata.canonical());

        return bookmarkRepository.save(bookmark);
    }

    public Bookmark updateBookmark(String id, UpdateBookmarkCommand cmd) {
        Bookmark bookmark = bookmarkRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        boolean urlChanged = !Objects.equals(cmd.url(), bookmark.getUrl());
        bookmark.setUrl(cmd.url());
        bookmark.setTitle(cmd.title());
        bookmark.setDescription(cmd.description());

        if (urlChanged) {
            BookmarkMetadataService.BookmarkMetadata metadata = bookmarkMetadataService.fetch(cmd.url());
            bookmark.setOgImage(metadata.ogImage());
            bookmark.setFavicon(metadata.favicon());
            bookmark.setCanonical(metadata.canonical());
            bookmark.setTitle(preferNonBlank(bookmark.getTitle(), metadata.title(), cmd.url()));
            bookmark.setDescription(preferNonBlank(bookmark.getDescription(), metadata.description()));
        }
        return bookmarkRepository.save(bookmark);
    }

    public void deleteBookmark(String id) {
        bookmarkRepository.findById(id).orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        bookmarkRepository.deleteById(id);
    }

    public record CreateBookmarkCommand(String collectionId, String url, String title, String description) {}

    public record UpdateBookmarkCommand(String url, String title, String description) {}

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
}
