package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Collection;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface CollectionRepository extends JpaRepository<Collection, String> {

    List<Collection> findBySpace_Id(String spaceId);
}
