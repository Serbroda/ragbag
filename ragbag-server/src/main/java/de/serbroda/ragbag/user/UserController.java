package de.serbroda.ragbag.user;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.UsersApi;
import de.serbroda.ragbag.generated.model.UserDto;
import de.serbroda.ragbag.security.SecurityUtils;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class UserController implements UsersApi {

    private final UserService userService;

    @Override
    public ResponseEntity<UserDto> me() {
        User user = userService
                .findUserById(SecurityUtils.currentUserId())
                .orElseThrow(() -> new IllegalStateException("Current user not found"));
        return ResponseEntity.ok(new UserDto.Builder()
                .id(user.getId())
                .username(user.getUsername())
                .email(user.getEmail())
                .build());
    }
}
