package de.serbroda.ragbag.init;

import de.serbroda.ragbag.auth.RegisterService;
import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionService;
import de.serbroda.ragbag.space.SpaceService;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Service;

/** Creates a demo admin account with sample collections for local development. Never active in production. */
@Slf4j
@Profile("!prod")
@Service
@RequiredArgsConstructor
public class DataInitializerImpl implements DataInitializer {

    private static final String PASSWORD = "test123";

    private final UserService userService;
    private final RegisterService registerService;
    private final SpaceService spaceService;
    private final CollectionService collectionService;

    @Override
    public void initializeData() {
        if (userService.anyExists("admin")) {
            return;
        }

        User admin = registerService.register("admin", "admin@example.com", PASSWORD);
        String spaceId = spaceService
                .getSpacesForUser(admin.getId())
                .iterator()
                .next()
                .space()
                .getId();

        Collection food = createCollection(admin, spaceId, "Food", null);
        createCollection(admin, spaceId, "Fruits", food);
        createCollection(admin, spaceId, "Vegetables", food);
        Collection meat = createCollection(admin, spaceId, "Meat", food);
        createCollection(admin, spaceId, "Beef", meat);
        createCollection(admin, spaceId, "Pork", meat);
        createCollection(admin, spaceId, "Books", null);

        log.info("Demo user 'admin' created with password '{}'", PASSWORD);
    }

    private Collection createCollection(User user, String spaceId, String name, Collection parent) {
        return collectionService.createCollection(
                user.getId(),
                new CollectionService.CreateCollectionCommand(spaceId, name, parent != null ? parent.getId() : null));
    }
}
