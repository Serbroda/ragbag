package de.serbroda.ragbag.bookmark;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_COLELCTION;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.BookmarkApi;
import de.serbroda.ragbag.generated.model.BookmarkDto;
import de.serbroda.ragbag.generated.model.CreateBookmarkDto;
import de.serbroda.ragbag.generated.model.UpdateBookmarkDto;
import de.serbroda.ragbag.security.SecurityUtils;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class BookmarkController implements BookmarkApi {

    private final BookmarkService bookmarkService;

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'READ')")
    @Override
    public ResponseEntity<List<BookmarkDto>> getBookmarks(String collectionId) {
        List<BookmarkDto> bookmarks = bookmarkService.getBookmarksByCollection(collectionId).stream()
                .map(this::toDto)
                .toList();
        return ResponseEntity.ok(bookmarks);
    }

    @Override
    public ResponseEntity<BookmarkDto> getBookmark(String bookmarkId) {
        Bookmark bookmark = bookmarkService.getBookmark(bookmarkId);
        return ResponseEntity.ok(toDto(bookmark));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLELCTION + "', 'WRITE')")
    @Override
    public ResponseEntity<BookmarkDto> createBookmark(String collectionId, CreateBookmarkDto createBookmarkDto) {
        Bookmark bookmark = bookmarkService.createBookmark(
                SecurityUtils.currentUserId(),
                new BookmarkService.CreateBookmarkCommand(
                        collectionId,
                        createBookmarkDto.getUrl(),
                        createBookmarkDto.getTitle(),
                        createBookmarkDto.getDescription()));
        return ResponseEntity.ok(toDto(bookmark));
    }

    @Override
    public ResponseEntity<BookmarkDto> updateBookmark(String bookmarkId, UpdateBookmarkDto updateBookmarkDto) {
        Bookmark bookmark = bookmarkService.updateBookmark(
                bookmarkId,
                new BookmarkService.UpdateBookmarkCommand(
                        updateBookmarkDto.getUrl(), updateBookmarkDto.getTitle(), updateBookmarkDto.getDescription()));
        return ResponseEntity.ok(toDto(bookmark));
    }

    @Override
    public ResponseEntity<Void> deleteBookmark(String bookmarkId) {
        bookmarkService.deleteBookmark(bookmarkId);
        return ResponseEntity.noContent().build();
    }

    private BookmarkDto toDto(Bookmark bookmark) {
        return new BookmarkDto.Builder()
                .id(bookmark.getId())
                .collectionId(
                        bookmark.getCollection() != null
                                ? bookmark.getCollection().getId()
                                : null)
                .url(bookmark.getUrl())
                .title(bookmark.getTitle())
                .description(bookmark.getDescription())
                .ogImage(bookmark.getOgImage())
                .favicon(bookmark.getFavicon())
                .canonical(bookmark.getCanonical())
                .build();
    }
}
