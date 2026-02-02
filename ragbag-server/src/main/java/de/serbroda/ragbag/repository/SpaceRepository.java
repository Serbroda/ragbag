package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.User;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface SpaceRepository extends JpaRepository<Space, String> {

    List<Space> findByCreatedBy(User user);
}
