package de.serbroda.ragbag.bookmark;

import java.util.Collection;
import java.util.List;
import java.util.Set;
import org.springframework.data.domain.Limit;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface BookmarkFavoriteRepository extends JpaRepository<BookmarkFavorite, BookmarkFavorite.Id> {

    @Query("SELECT f.id.bookmarkId FROM BookmarkFavorite f"
            + " WHERE f.id.userId = :userId AND f.id.bookmarkId IN :bookmarkIds")
    Set<String> findFavoriteIds(@Param("userId") String userId, @Param("bookmarkIds") Collection<String> bookmarkIds);

    @Query("""
			SELECT b FROM Bookmark b, BookmarkFavorite f
			WHERE f.id.bookmarkId = b.id AND f.id.userId = :userId AND b.collection.space.id = :spaceId
			AND b.collection.id NOT IN :hidden
			ORDER BY f.createdAt DESC
			""")
    List<Bookmark> findFavoritesInSpace(
            @Param("userId") String userId,
            @Param("spaceId") String spaceId,
            @Param("hidden") Collection<String> hiddenCollectionIds,
            Limit limit);

    @Modifying
    @Query("""
			DELETE FROM BookmarkFavorite f
			WHERE f.id.userId = :userId
			AND f.id.bookmarkId IN (SELECT b.id FROM Bookmark b WHERE b.collection.space.id = :spaceId)
			""")
    void deleteAllInSpace(@Param("userId") String userId, @Param("spaceId") String spaceId);
}
