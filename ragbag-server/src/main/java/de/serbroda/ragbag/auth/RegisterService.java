package de.serbroda.ragbag.auth;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.collection.CollectionService;
import de.serbroda.ragbag.space.Space;
import de.serbroda.ragbag.space.SpaceService;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
@Transactional
public class RegisterService {

    private final UserService userService;
    private final SpaceService spaceService;
    private final CollectionService collectionService;

    public User register(String username, String email, String password) {
        User user = userService.createUser(username, email, password);

        Space space = spaceService.createSpace(
                user.getId(),
                new SpaceService.CreateSpaceCommand(StringUtils.capitalize(user.getUsername()) + "'s Space"));

        Collection food = createCollection(user, space, "Food", null);
        createCollection(user, space, "Fruits", food);
        createCollection(user, space, "Vegetables", food);
        Collection meat = createCollection(user, space, "Meat", food);
        createCollection(user, space, "Beef", meat);
        createCollection(user, space, "Pork", meat);

        createCollection(user, space, "Books", null);

        return user;
    }

    private Collection createCollection(User user, Space space, String name, Collection parent) {
        Collection collection = new Collection();
        collection.setSpace(space);
        collection.setName(name);
        collection.setCreatedBy(user);

        if (parent != null) {
            collection.setParent(parent);
        }
        return collectionService.createCollection(collection);
    }
}
