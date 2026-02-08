package de.serbroda.ragbag.space;

import java.util.List;
import java.util.Optional;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SpaceMemberRepository extends JpaRepository<SpaceMember, SpaceMemberId> {

    List<SpaceMember> findByUser_Id(String userId);

    Optional<SpaceMember> findBySpaceAndUser_Id(Space space, String user);

    Optional<SpaceMember> findBySpace_IdAndUser_Id(String spaceId, String user);

    boolean existsBySpace_IdAndUser_Id(String spaceId, String userId);
}
