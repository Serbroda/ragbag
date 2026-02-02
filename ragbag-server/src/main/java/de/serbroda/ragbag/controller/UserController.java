package de.serbroda.ragbag.controller;

import de.serbroda.ragbag.security.UserPrincipal;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/users")
public class UserController {

    @GetMapping("/me")
    public Map<String, String> me(Authentication auth) {
        UserPrincipal principal = (UserPrincipal) auth.getPrincipal();
        return Map.of(
                "userId", principal.getUserId(),
                "username", principal.getUsername()
        );
    }
}
