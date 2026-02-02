package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Collection;
import org.springframework.data.jpa.repository.JpaRepository;

public interface CollectionRepository extends JpaRepository<Collection, String> {
}
