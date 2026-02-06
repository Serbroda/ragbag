package de.serbroda.ragbag.service;

import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
public class DataInitializerImpl implements DataInitializer {

    private final UserService userService;
    private final RegisterService registerService;

    @Override
    public void initializeData() {
        // final String password = UUID.randomUUID().toString();
        final String password = "test123";
        registerService.register("admin", "admin@example.com", password);

        System.out.println("===================================================");
        System.out.println("User 'admin' created with password: " + password);
        System.out.println("===================================================");
    }
}
