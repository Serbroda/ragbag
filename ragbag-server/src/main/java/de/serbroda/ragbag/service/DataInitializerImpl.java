package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.repository.UserRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.util.Optional;
import java.util.UUID;

@Service
@RequiredArgsConstructor
public class DataInitializerImpl implements DataInitializer {

    private final UserService userService;

    @Override
    public void initializeData() {
        if (userService.findUserByUsernameOrEmail("admin").isEmpty()) {
            final String password = UUID.randomUUID().toString();
            userService.createUser("admin", "admin@example.com", password);

            System.out.println("===================================================");
            System.out.println("User 'admin' created with password: " + password);
            System.out.println("===================================================");
        }
    }

}
