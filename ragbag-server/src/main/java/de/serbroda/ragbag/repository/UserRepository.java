package de.serbroda.ragbag.repository;

import de.serbroda.ragbag.model.User;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;

import java.util.Optional;

public interface UserRepository extends JpaRepository<User, String> {

    @Query("SELECT u FROM User u WHERE LOWER(u.username) = LOWER(?1) OR LOWER(u.email) = LOWER(?1)")
    Optional<User> findUserByUsernameOrEmail(String email);
}
