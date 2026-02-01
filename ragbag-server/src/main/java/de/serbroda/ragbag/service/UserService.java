package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.model.keys.SpaceMemberId;
import de.serbroda.ragbag.repository.SpaceRepository;
import de.serbroda.ragbag.repository.UserRepository;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.apache.commons.lang3.StringUtils;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.util.Optional;

@RequiredArgsConstructor
@Service
public class UserService {

    private final UserRepository userRepository;
    private final SpaceRepository spaceRepository;
    private final PasswordEncoder passwordEncoder;

    public Optional<User> findUserByUsernameOrEmail(String usernameOrEmail) {
        return userRepository.findUserByUsernameOrEmail(usernameOrEmail);
    }

    @Transactional
    public User createUser(String username, String email, final String passwordPlain) {
        User user = new User();
        user.setUsername(username.toLowerCase());
        user.setEmail(email.toLowerCase());
        user.setPassword(passwordEncoder.encode(passwordPlain));

        user = userRepository.save(user);

        Space space = new Space();
        space.setName(StringUtils.capitalize(user.getUsername()) + "'s Space");
        space.setCreatedBy(user);

        SpaceMember member = new SpaceMember();
        member.setSpace(space);
        member.setUser(user);
        member.setRole("ADMIN");

        SpaceMemberId id = new SpaceMemberId();
        id.setSpaceId(space.getId());
        id.setUserId(user.getId());
        member.setId(id);

        space.getMembers().add(member);

        spaceRepository.save(space);

        return user;
    }
}
