package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.repository.UserRepository;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.apache.commons.lang3.StringUtils;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.util.Optional;

@Transactional
@RequiredArgsConstructor
@Service
public class UserService {

    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;
    private final SpaceService spaceService;

    public Optional<User> findUserByUsernameOrEmail(String usernameOrEmail) {
        return userRepository.findUserByUsernameOrEmail(usernameOrEmail);
    }

    public User createUser(String username, String email, final String passwordPlain) {
        if (userRepository.findUserByUsernameOrEmail(username).isPresent()) {
            throw new IllegalArgumentException("Username already exists: " + username);
        }

        User user = new User();
        user.setUsername(username.toLowerCase());
        user.setEmail(email.toLowerCase());
        user.setPassword(passwordEncoder.encode(passwordPlain));

        user = userRepository.save(user);

        spaceService.createSpace(user, StringUtils.capitalize(user.getUsername()) + "'s Space");

        return user;
    }

}
