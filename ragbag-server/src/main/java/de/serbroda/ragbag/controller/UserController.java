package de.serbroda.ragbag.controller;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.security.UserPrincipal;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@SecurityRequirement(name = "basicAuth")
@Tag(name = "Users", description = "Manage users")
@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/users")
public class UserController {

    @Operation(summary = "Get current user info")
    @GetMapping("/me")
    public Map<String, String> me() {
        UserPrincipal principal = (UserPrincipal)
                SecurityContextHolder.getContext().getAuthentication().getPrincipal();
        return Map.of(
                "userId", principal.getUserId(),
                "username", principal.getUsername());
    }
}
