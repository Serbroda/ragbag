package de.serbroda.ragbag.dashboard;

import java.util.List;
import org.springframework.data.jpa.repository.JpaRepository;

public interface DashboardSectionRepository extends JpaRepository<DashboardSection, String> {

    List<DashboardSection> findByUserIdAndSpaceIdOrderByPosition(String userId, String spaceId);

    void deleteByUserIdAndSpaceId(String userId, String spaceId);
}
