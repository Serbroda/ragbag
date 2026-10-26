package de.serbroda.ragbag.bookmark;

import java.util.Collection;
import java.util.List;
import java.util.Optional;
import org.springframework.data.domain.Limit;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface BookmarkRepository extends JpaRepository<Bookmark, String> {

    List<Bookmark> findByCollection_IdOrderByCreatedAtDesc(String collectionId);

    List<Bookmark> findByCollection_Space_IdAndCollection_IdNotInOrderByCreatedAtDesc(
            String spaceId, Collection<String> hiddenCollectionIds, Limit limit);

    @Query("SELECT b.collection.id FROM Bookmark b WHERE b.id = :id")
    Optional<String> findCollectionIdById(@Param("id") String id);

    @Query("""
			SELECT DISTINCT b FROM Bookmark b
			LEFT JOIN b.tags t
			WHERE b.collection.space.id = :spaceId
			AND b.collection.id NOT IN :hidden
			AND (LOWER(b.title) LIKE LOWER(CONCAT('%', :query, '%')) ESCAPE '\\'
			OR LOWER(b.url) LIKE LOWER(CONCAT('%', :query, '%')) ESCAPE '\\'
			OR LOWER(b.description) LIKE LOWER(CONCAT('%', :query, '%')) ESCAPE '\\'
			OR LOWER(t) LIKE LOWER(CONCAT('%', :query, '%')) ESCAPE '\\')
			ORDER BY b.createdAt DESC
			""")
    List<Bookmark> searchInSpace(
            @Param("spaceId") String spaceId,
            @Param("query") String query,
            @Param("hidden") Collection<String> hiddenCollectionIds);
}
