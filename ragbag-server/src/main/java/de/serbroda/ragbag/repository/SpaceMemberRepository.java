package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.keys.SpaceMemberId;
import java.util.List;
import java.util.Optional;
import org.springframework.data.jpa.repository.JpaRepository;

public interface SpaceMemberRepository extends JpaRepository<SpaceMember, SpaceMemberId> {

    List<SpaceMember> findByUser_Id(String userId);

    Optional<SpaceMember> findBySpaceAndUser_Id(Space space, String user);

    Optional<SpaceMember> findBySpace_IdAndUser_Id(String spaceId, String user);
}
