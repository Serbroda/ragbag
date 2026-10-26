package de.serbroda.ragbag.user;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.UserApi;
import de.serbroda.ragbag.generated.model.UserDto;
import de.serbroda.ragbag.security.SecurityUtils;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class UserController implements UserApi {

    private final UserService userService;
    private final InstanceAdminService adminService;

    @Override
    public ResponseEntity<UserDto> me() {
        User user = userService
                .findUserById(SecurityUtils.currentUserId())
                .orElseThrow(() -> new IllegalStateException("Current user not found"));
        return ResponseEntity.ok(toDto(user));
    }

    @Override
    public ResponseEntity<UserDto> setAvatar(MultipartFile file, String url) {
        return ResponseEntity.ok(toDto(userService.setAvatar(SecurityUtils.currentUserId(), file, url)));
    }

    @Override
    public ResponseEntity<UserDto> removeAvatar() {
        return ResponseEntity.ok(toDto(userService.removeAvatar(SecurityUtils.currentUserId())));
    }

    private UserDto toDto(User user) {
        return new UserDto.Builder()
                .id(user.getId())
                .username(user.getUsername())
                .email(user.getEmail())
                .avatarId(user.getAvatarImageId())
                .instanceAdmin(adminService.isAdmin(user))
                .build();
    }
}
