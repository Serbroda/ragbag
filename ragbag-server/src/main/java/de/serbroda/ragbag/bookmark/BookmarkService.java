package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionRepository;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import jakarta.transaction.Transactional;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Transactional
@RequiredArgsConstructor
@Service
public class BookmarkService {

    private final BookmarkRepository bookmarkRepository;
    private final CollectionRepository collectionRepository;
    private final UserRepository userRepository;

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

        Bookmark bookmark = new Bookmark();
        bookmark.setCollection(collection);
        bookmark.setCreatedBy(user);
        bookmark.setUrl(cmd.url());
        bookmark.setTitle(cmd.title());
        bookmark.setDescription(cmd.description());

        return bookmarkRepository.save(bookmark);
    }

    public Bookmark updateBookmark(String id, UpdateBookmarkCommand cmd) {
        Bookmark bookmark = bookmarkRepository
                .findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        bookmark.setUrl(cmd.url());
        bookmark.setTitle(cmd.title());
        bookmark.setDescription(cmd.description());
        return bookmarkRepository.save(bookmark);
    }

    public void deleteBookmark(String id) {
        bookmarkRepository.findById(id).orElseThrow(() -> new ResourceNotFoundException("Bookmark not found: " + id));
        bookmarkRepository.deleteById(id);
    }

    public record CreateBookmarkCommand(String collectionId, String url, String title, String description) {}

    public record UpdateBookmarkCommand(String url, String title, String description) {}
}
