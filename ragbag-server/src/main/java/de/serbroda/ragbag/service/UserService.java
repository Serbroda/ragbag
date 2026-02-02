package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.Collection;
import de.serbroda.ragbag.model.Space;
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
    private final CollectionService collectionService;

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

        Space space = spaceService.createSpace(user, StringUtils.capitalize(user.getUsername()) + "'s Space");

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
