package de.serbroda.ragbag.user;

import java.time.Instant;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;

public interface PasswordResetRepository extends JpaRepository<PasswordReset, String> {

    @Modifying
    @Query("DELETE FROM PasswordReset r WHERE r.userId = :userId OR r.expiresAt < :now")
    void deleteForUserOrExpired(@Param("userId") String userId, @Param("now") Instant now);
}
