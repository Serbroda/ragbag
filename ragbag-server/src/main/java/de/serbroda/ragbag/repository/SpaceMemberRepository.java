package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.model.keys.SpaceMemberId;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.Optional;

public interface SpaceMemberRepository extends JpaRepository<SpaceMember, SpaceMemberId> {

    List<SpaceMember> findByUser_Id(String userId);

    Optional<SpaceMember> findBySpaceAndUser_Id(Space space, String user);
}