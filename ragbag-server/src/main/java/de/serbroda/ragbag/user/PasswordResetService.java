package de.serbroda.ragbag.user;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.time.Duration;
import java.time.Instant;
import java.util.Base64;
import java.util.HexFormat;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Password reset without email: an instance admin creates a one-time link and hands it over (messenger, in
 * person). Only a hash of the token is stored, so a leaked database does not contain usable links.
 */
@Transactional
@RequiredArgsConstructor
@Service
public class PasswordResetService {

    static final Duration VALIDITY = Duration.ofHours(24);

    private static final SecureRandom RANDOM = new SecureRandom();

    private final PasswordResetRepository resetRepository;
    private final UserRepository userRepository;
    private final PasswordEncoder passwordEncoder;

    /** Replaces an older link of the user. Returns the token, which is not stored anywhere. */
    public CreatedReset create(String adminId, String userId) {
        if (!userRepository.existsById(userId)) {
            throw new ResourceNotFoundException("User not found: " + userId);
        }
        Instant now = Instant.now();
        resetRepository.deleteForUserOrExpired(userId, now);
        resetRepository.flush();

        byte[] bytes = new byte[32];
        RANDOM.nextBytes(bytes);
        String token = Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);

        PasswordReset reset = new PasswordReset();
        reset.setTokenHash(hash(token));
        reset.setUserId(userId);
        reset.setCreatedBy(adminId);
        reset.setExpiresAt(now.plus(VALIDITY));
        resetRepository.save(reset);
        return new CreatedReset(token, reset.getExpiresAt());
    }

    /** The user a still valid link belongs to. */
    @Transactional(readOnly = true)
    public Optional<User> findUser(String token) {
        return findValid(token).flatMap(reset -> userRepository.findById(reset.getUserId()));
    }

    /** Sets the new password, uses up the link and signs the user out everywhere. */
    public void reset(String token, String newPassword) {
        PasswordReset reset = findValid(token)
                .orElseThrow(() -> new IllegalArgumentException("This reset link is invalid, expired or already used"));
        UserService.requireValidPassword(newPassword);
        User user = userRepository
                .findById(reset.getUserId())
                .orElseThrow(() -> new IllegalArgumentException("This reset link is invalid"));
        user.setPassword(passwordEncoder.encode(newPassword));
        user.setTokenVersion(user.getTokenVersion() + 1);
        resetRepository.delete(reset);
    }

    private Optional<PasswordReset> findValid(String token) {
        if (token == null || token.isBlank()) {
            return Optional.empty();
        }
        return resetRepository
                .findById(hash(token))
                .filter(reset -> reset.getExpiresAt().isAfter(Instant.now()));
    }

    private static String hash(String token) {
        try {
            return HexFormat.of()
                    .formatHex(MessageDigest.getInstance("SHA-256").digest(token.getBytes(StandardCharsets.UTF_8)));
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException(e);
        }
    }

    public record CreatedReset(String token, Instant expiresAt) {}
}
