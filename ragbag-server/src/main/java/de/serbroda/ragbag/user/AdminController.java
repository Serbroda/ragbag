package de.serbroda.ragbag.user;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.AdminApi;
import de.serbroda.ragbag.generated.model.AdminUserDto;
import de.serbroda.ragbag.generated.model.PasswordResetDto;
import de.serbroda.ragbag.security.SecurityUtils;
import java.time.ZoneOffset;
import java.util.Comparator;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class AdminController implements AdminApi {

    private final InstanceAdminService adminService;
    private final PasswordResetService passwordResetService;
    private final UserRepository userRepository;

    @Override
    public ResponseEntity<List<AdminUserDto>> getAllUsers() {
        adminService.requireAdmin(SecurityUtils.currentUserId());
        return ResponseEntity.ok(userRepository.findAll().stream()
                .sorted(Comparator.comparing(User::getUsername))
                .map(user -> new AdminUserDto.Builder()
                        .id(user.getId())
                        .username(user.getUsername())
                        .email(user.getEmail())
                        .avatarId(user.getAvatarImageId())
                        .instanceAdmin(adminService.isAdmin(user))
                        .createdAt(
                                user.getCreatedAt() != null
                                        ? user.getCreatedAt().atOffset(ZoneOffset.UTC)
                                        : null)
                        .spaces(user.getSpaces().size())
                        .build())
                .toList());
    }

    @Override
    public ResponseEntity<PasswordResetDto> createPasswordReset(String userId) {
        String adminId = SecurityUtils.currentUserId();
        adminService.requireAdmin(adminId);
        PasswordResetService.CreatedReset reset = passwordResetService.create(adminId, userId);
        return ResponseEntity.ok(new PasswordResetDto.Builder()
                .token(reset.token())
                .expiresAt(reset.expiresAt().atOffset(ZoneOffset.UTC))
                .build());
    }
}
