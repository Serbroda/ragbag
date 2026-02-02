package de.serbroda.ragbag.controller;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.service.SpaceService;
import de.serbroda.ragbag.service.UserService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Optional;
import java.util.Set;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/spaces")
public class SpaceController {

    private final UserService userService;
    private final SpaceService spaceService;

    @GetMapping
    public ResponseEntity getSpaces() {
        Optional<User> admin = userService.findUserByUsernameOrEmail("admin");
        if (admin.isEmpty()) {
            return ResponseEntity.status(404).body("Admin user not found");
        }

        Set<Space> spaces = spaceService.getSpacesForUser(admin.get());
        return ResponseEntity.ok(spaces);
    }
}
