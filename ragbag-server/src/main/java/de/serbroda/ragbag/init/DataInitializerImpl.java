package de.serbroda.ragbag.init;

import de.serbroda.ragbag.auth.RegisterService;
import de.serbroda.ragbag.user.UserService;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class DataInitializerImpl implements DataInitializer {

    private final UserService userService;
    private final RegisterService registerService;

    @Override
    public void initializeData() {
        if (userService.anyExists("admin")) {
            return;
        }

        // final String password = UUID.randomUUID().toString();
        final String password = "test123";
        registerService.register("admin", "admin@example.com", password);

        System.out.println("===================================================");
        System.out.println("User 'admin' created with password: " + password);
        System.out.println("===================================================");
    }
}
