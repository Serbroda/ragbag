package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.generated.model.BookmarkDto;
import java.time.ZoneOffset;
import java.util.List;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Component;

@RequiredArgsConstructor
@Component
public class BookmarkMapper {

    private final BookmarkFavoriteRepository favoriteRepository;

    /** Maps a list for the given user, resolving all favorite flags with a single query. */
    public List<BookmarkDto> toDtos(List<Bookmark> bookmarks, String userId) {
        if (bookmarks.isEmpty()) {
            return List.of();
        }
        Set<String> favorites = favoriteRepository.findFavoriteIds(
                userId, bookmarks.stream().map(Bookmark::getId).toList());
        return bookmarks.stream()
                .map(bookmark -> toDto(bookmark, favorites.contains(bookmark.getId())))
                .toList();
    }

    public BookmarkDto toDto(Bookmark bookmark, String userId) {
        return toDto(bookmark, favoriteRepository.existsById(new BookmarkFavorite.Id(userId, bookmark.getId())));
    }

    public BookmarkDto toDto(Bookmark bookmark, boolean favorite) {
        return new BookmarkDto.Builder()
                .id(bookmark.getId())
                .collectionId(
                        bookmark.getCollection() != null
                                ? bookmark.getCollection().getId()
                                : null)
                .url(bookmark.getUrl())
                .title(bookmark.getTitle())
                .description(bookmark.getDescription())
                .tags(
                        bookmark.getTags() != null
                                // Stored in an unordered set, so sort for a stable display order
                                ? bookmark.getTags().stream()
                                        .sorted(String.CASE_INSENSITIVE_ORDER)
                                        .toList()
                                : null)
                .ogImage(bookmark.getOgImage())
                .favicon(bookmark.getFavicon())
                .canonical(bookmark.getCanonical())
                .createdAt(
                        bookmark.getCreatedAt() != null
                                ? bookmark.getCreatedAt().atOffset(ZoneOffset.UTC)
                                : null)
                .createdBy(
                        bookmark.getCreatedBy() != null
                                ? bookmark.getCreatedBy().getUsername()
                                : null)
                .favorite(favorite)
                .build();
    }
}
