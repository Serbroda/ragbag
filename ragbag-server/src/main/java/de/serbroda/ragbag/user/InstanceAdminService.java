package de.serbroda.ragbag.user;

import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.stream.Collectors;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

/**
 * Instance admins manage the users of the whole installation (e.g. password reset links). They are configured by
 * username (app.admin-users / ADMIN_USERS) rather than stored, so an open registration can never create one.
 */
@Service
public class InstanceAdminService {

    private final Set<String> adminUsernames;
    private final UserRepository userRepository;

    public InstanceAdminService(
            @Value("${app.admin-users:}") List<String> adminUsernames, UserRepository userRepository) {
        this.adminUsernames = adminUsernames.stream()
                .map(String::trim)
                .filter(name -> !name.isEmpty())
                .map(name -> name.toLowerCase(Locale.ROOT))
                .collect(Collectors.toUnmodifiableSet());
        this.userRepository = userRepository;
    }

    public boolean isAdmin(User user) {
        return adminUsernames.contains(user.getUsername().toLowerCase(Locale.ROOT));
    }

    public void requireAdmin(String userId) {
        boolean admin = userRepository.findById(userId).map(this::isAdmin).orElse(false);
        if (!admin) {
            throw new AccessDeniedException("Only instance admins (ADMIN_USERS) may do this");
        }
    }
}
