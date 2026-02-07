package de.serbroda.ragbag.collection;

import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface CollectionRepository extends JpaRepository<Collection, String> {

    List<Collection> findBySpace_Id(String spaceId);
}
