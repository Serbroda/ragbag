package de.serbroda.ragbag.bookmark;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_BOOKMARK;
import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_COLLECTION;
import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.BookmarkApi;
import de.serbroda.ragbag.generated.model.BookmarkDto;
import de.serbroda.ragbag.generated.model.BookmarkMetadataDto;
import de.serbroda.ragbag.generated.model.CreateBookmarkDto;
import de.serbroda.ragbag.generated.model.FetchMetadataRequest;
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
    private final BookmarkMapper mapper;

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<List<BookmarkDto>> searchBookmarks(String spaceId, String q) {
        return ResponseEntity.ok(
                mapper.toDtos(bookmarkService.searchInSpace(spaceId, q), SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<List<BookmarkDto>> getRecentBookmarks(String spaceId, Integer limit) {
        return ResponseEntity.ok(mapper.toDtos(
                bookmarkService.getRecentInSpace(spaceId, limit != null ? limit : 20), SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'CREATE_BOOKMARKS')"
            + " or hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'EDIT_BOOKMARKS')")
    @Override
    public ResponseEntity<BookmarkMetadataDto> fetchBookmarkMetadata(
            String collectionId, FetchMetadataRequest fetchMetadataRequest) {
        BookmarkService.FetchedMetadata fetched = bookmarkService.fetchMetadata(fetchMetadataRequest.getUrl());
        BookmarkMetadataService.BookmarkMetadata metadata = fetched.metadata();
        return ResponseEntity.ok(new BookmarkMetadataDto.Builder()
                .url(fetched.url())
                .found(metadata.title() != null || metadata.description() != null)
                .title(metadata.title())
                .description(metadata.description())
                .ogImage(metadata.ogImage())
                .favicon(metadata.favicon())
                .canonical(metadata.canonical())
                .keywords(metadata.keywords())
                .build());
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'READ')")
    @Override
    public ResponseEntity<List<BookmarkDto>> getBookmarks(String collectionId) {
        return ResponseEntity.ok(
                mapper.toDtos(bookmarkService.getBookmarksByCollection(collectionId), SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#bookmarkId, '" + DOMAIN_PREFIX_BOOKMARK + "', 'READ')")
    @Override
    public ResponseEntity<BookmarkDto> getBookmark(String bookmarkId) {
        Bookmark bookmark = bookmarkService.getBookmark(bookmarkId);
        return ResponseEntity.ok(mapper.toDto(bookmark, SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#collectionId, '" + DOMAIN_PREFIX_COLLECTION + "', 'CREATE_BOOKMARKS')")
    @Override
    public ResponseEntity<BookmarkDto> createBookmark(String collectionId, CreateBookmarkDto createBookmarkDto) {
        Bookmark bookmark = bookmarkService.createBookmark(
                SecurityUtils.currentUserId(),
                new BookmarkService.CreateBookmarkCommand(
                        collectionId,
                        createBookmarkDto.getUrl(),
                        createBookmarkDto.getTitle(),
                        createBookmarkDto.getDescription(),
                        createBookmarkDto.getTags()));
        return ResponseEntity.ok(mapper.toDto(bookmark, SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#bookmarkId, '" + DOMAIN_PREFIX_BOOKMARK + "', 'EDIT_BOOKMARKS')")
    @Override
    public ResponseEntity<BookmarkDto> updateBookmark(String bookmarkId, UpdateBookmarkDto updateBookmarkDto) {
        Bookmark bookmark = bookmarkService.updateBookmark(
                bookmarkId,
                new BookmarkService.UpdateBookmarkCommand(
                        updateBookmarkDto.getUrl(),
                        updateBookmarkDto.getTitle(),
                        updateBookmarkDto.getDescription(),
                        updateBookmarkDto.getTags()));
        return ResponseEntity.ok(mapper.toDto(bookmark, SecurityUtils.currentUserId()));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<List<BookmarkDto>> getFavorites(String spaceId) {
        String userId = SecurityUtils.currentUserId();
        return ResponseEntity.ok(bookmarkService.getFavoritesInSpace(userId, spaceId).stream()
                .map(bookmark -> mapper.toDto(bookmark, true))
                .toList());
    }

    // Anyone who can see a bookmark may star it; favorites are personal
    @PreAuthorize("hasPermission(#bookmarkId, '" + DOMAIN_PREFIX_BOOKMARK + "', 'READ')")
    @Override
    public ResponseEntity<Void> addFavorite(String bookmarkId) {
        bookmarkService.addFavorite(SecurityUtils.currentUserId(), bookmarkId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasPermission(#bookmarkId, '" + DOMAIN_PREFIX_BOOKMARK + "', 'READ')")
    @Override
    public ResponseEntity<Void> removeFavorite(String bookmarkId) {
        bookmarkService.removeFavorite(SecurityUtils.currentUserId(), bookmarkId);
        return ResponseEntity.noContent().build();
    }

    @PreAuthorize("hasPermission(#bookmarkId, '" + DOMAIN_PREFIX_BOOKMARK + "', 'DELETE_BOOKMARKS')")
    @Override
    public ResponseEntity<Void> deleteBookmark(String bookmarkId) {
        bookmarkService.deleteBookmark(bookmarkId);
        return ResponseEntity.noContent().build();
    }
}
