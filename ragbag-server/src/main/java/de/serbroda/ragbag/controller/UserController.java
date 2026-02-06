package de.serbroda.ragbag.controller;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.security.JwtService;
import de.serbroda.ragbag.security.SecurityUtils;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@SecurityRequirement(name = "basicAuth")
@Tag(name = "Users", description = "Manage users")
@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/users")
public class UserController {

    private final JwtService jwtService;

    @Operation(summary = "Get current user info")
    @GetMapping("/me")
    public Map<String, String> me() {
        return Map.of(
                "userId", SecurityUtils.currentUserId(),
                "username", SecurityUtils.currentUsername());
    }
}
