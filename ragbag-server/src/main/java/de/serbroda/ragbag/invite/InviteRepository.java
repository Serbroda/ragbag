package de.serbroda.ragbag.invite;

import java.util.Optional;
import org.springframework.data.jpa.repository.JpaRepository;

public interface InviteRepository extends JpaRepository<Invite, String> {

    Optional<Invite> findByToken(String token);
}
