package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Space;
import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SpaceRepository extends JpaRepository<Space, String> {

    List<Space> findByCreatedBy_id(String userId);
}
