package de.serbroda.ragbag.user;

import de.serbroda.ragbag.image.ImageService;
import de.serbroda.ragbag.shared.exception.ResourceAlreadyExistsException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.nio.charset.StandardCharsets;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.multipart.MultipartFile;

@Transactional
@RequiredArgsConstructor
@Service
public class UserService {

    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;
    private final ImageService imageService;

    public static final int MIN_PASSWORD_LENGTH = 8;

    public Optional<User> findUserById(String id) {
        return userRepository.findById(id);
    }

    public User createUser(String username, String email, final String passwordPlain) {
        if (anyExists(username.toLowerCase(), email.toLowerCase())) {
            throw new ResourceAlreadyExistsException("User with same username or email already exists");
        }

        User user = new User();
        user.setUsername(username.toLowerCase());
        user.setEmail(email.toLowerCase());
        user.setPassword(passwordEncoder.encode(passwordPlain));
        user.setTokenVersion(0);

        return userRepository.save(user);
    }

    public User setAvatar(String userId, MultipartFile file, String url) {
        User user = requireUser(userId);
        String previous = user.getAvatarImageId();
        user.setAvatarImageId(imageService.store(file, url).getId());
        imageService.deleteImage(previous);
        return user;
    }

    public User removeAvatar(String userId) {
        User user = requireUser(userId);
        imageService.deleteImage(user.getAvatarImageId());
        user.setAvatarImageId(null);
        return user;
    }

    private User requireUser(String userId) {
        return userRepository
                .findById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found: " + userId));
    }

    /** Checks the current password, then signs out every session (token version) except the caller's new one. */
    public User changePassword(String userId, String currentPassword, String newPassword) {
        User user = requireUser(userId);
        if (currentPassword == null || !passwordEncoder.matches(currentPassword, user.getPassword())) {
            throw new IllegalArgumentException("The current password is wrong");
        }
        requireValidPassword(newPassword);
        user.setPassword(passwordEncoder.encode(newPassword));
        user.setTokenVersion(user.getTokenVersion() + 1);
        return user;
    }

    public static void requireValidPassword(String password) {
        if (password == null || password.length() < MIN_PASSWORD_LENGTH) {
            throw new IllegalArgumentException("Passwords must have at least " + MIN_PASSWORD_LENGTH + " characters");
        }
        // BCrypt silently ignores everything after 72 bytes
        if (password.getBytes(StandardCharsets.UTF_8).length > 72) {
            throw new IllegalArgumentException("Passwords must be at most 72 bytes long");
        }
    }

    public void incrementTokenVersion(String userId) {
        userRepository.findById(userId).ifPresent(user -> {
            user.setTokenVersion(user.getTokenVersion() + 1);
            userRepository.save(user);
        });
    }

    public boolean anyExists(String... usernamesOrEmails) {
        for (String uoe : usernamesOrEmails) {
            if (userRepository.findUserByUsernameOrEmail(uoe).isPresent()) {
                return true;
            }
        }
        return false;
    }
}
