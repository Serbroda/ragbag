package de.serbroda.ragbag.space;

import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SpaceRepository extends JpaRepository<Space, String> {

    List<Space> findByCreatedBy_Id(String userId);
}
