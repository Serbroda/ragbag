package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Space;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SpaceRepository extends JpaRepository<Space, String> {
}
