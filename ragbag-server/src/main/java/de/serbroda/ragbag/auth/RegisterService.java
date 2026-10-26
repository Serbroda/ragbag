package de.serbroda.ragbag.auth;

import de.serbroda.ragbag.space.SpaceService;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import lombok.RequiredArgsConstructor;
import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@RequiredArgsConstructor
@Transactional
public class RegisterService {

    private final UserService userService;
    private final SpaceService spaceService;

    public User register(String username, String email, String password) {
        User user = userService.createUser(username, email, password);

        spaceService.createSpace(
                user.getId(),
                new SpaceService.CreateSpaceCommand(StringUtils.capitalize(user.getUsername()) + "'s Space"));

        return user;
    }
}
