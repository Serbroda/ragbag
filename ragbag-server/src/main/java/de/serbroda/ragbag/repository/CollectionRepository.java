package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Collection;
import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface CollectionRepository extends JpaRepository<Collection, String> {

    List<Collection> findBySpace_Id(String spaceId);
}
