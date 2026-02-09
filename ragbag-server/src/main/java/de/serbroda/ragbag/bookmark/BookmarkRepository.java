package de.serbroda.ragbag.bookmark;

import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface BookmarkRepository extends JpaRepository<Bookmark, String> {

    List<Bookmark> findByCollection_Id(String collectionId);
}
