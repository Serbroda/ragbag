package de.serbroda.ragbag.collection;

import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface CollectionRepository extends JpaRepository<Collection, String> {

    List<Collection> findBySpace_Id(String spaceId);

    boolean existsBySpace_IdAndRestrictedTrue(String spaceId);

    @Query("SELECT c FROM Collection c JOIN c.memberIds m WHERE c.space.id = :spaceId AND m = :userId")
    List<Collection> findRestrictedBySpaceAndMember(@Param("spaceId") String spaceId, @Param("userId") String userId);
}
