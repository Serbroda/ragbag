package de.serbroda.ragbag.service;

import de.serbroda.ragbag.exception.EntityAlreadyExistsException;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.repository.UserRepository;
import jakarta.transaction.Transactional;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

@Transactional
@RequiredArgsConstructor
@Service
public class UserService {

    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;

    public Optional<User> findUserById(String id) {
        return userRepository.findById(id);
    }

    public User createUser(String username, String email, final String passwordPlain) {
        if (anyExists(username.toLowerCase(), email.toLowerCase())) {
            throw new EntityAlreadyExistsException("User with same username or email already exists");
        }

        User user = new User();
        user.setUsername(username.toLowerCase());
        user.setEmail(email.toLowerCase());
        user.setPassword(passwordEncoder.encode(passwordPlain));
        user.setTokenVersion(0);

        return userRepository.save(user);
    }

    public void incrementTokenVersion(String userId) {
        userRepository.findById(userId).ifPresent(user -> {
            user.setTokenVersion(user.getTokenVersion() + 1);
            userRepository.save(user);
        });
    }

    private boolean anyExists(String... usernamesOrEmails) {
        for (String uoe : usernamesOrEmails) {
            if (userRepository.findUserByUsernameOrEmail(uoe).isPresent()) {
                return true;
            }
        }
        return false;
    }
}
