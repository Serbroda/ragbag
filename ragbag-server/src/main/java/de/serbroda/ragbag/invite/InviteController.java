package de.serbroda.ragbag.invite;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.InviteApi;
import de.serbroda.ragbag.generated.model.CreateInviteRequest;
import de.serbroda.ragbag.generated.model.InviteDto;
import de.serbroda.ragbag.security.SecurityUtils;
import java.security.NoSuchAlgorithmException;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RequestMapping(PUBLIC_API_PREFIX)
@RestController
public class InviteController implements InviteApi {

    private final InviteService inviteService;
    private final InviteMapper inviteMapper;

    @Override
    public ResponseEntity<Void> acceptInvite(String token) {
        inviteService.acceptInvite(SecurityUtils.currentUserId(), token);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<InviteDto> createInvite(CreateInviteRequest createInviteRequest) {
        try {
            Invite invite = inviteService.createInvite(
                    SecurityUtils.currentUserId(),
                    new InviteService.CreateInviteCommand(
                            InviteTargetType.valueOf(
                                    createInviteRequest.getTargetType().name()),
                            createInviteRequest.getTargetId(),
                            createInviteRequest.getExpiresAt(),
                            createInviteRequest.getMaxUses(),
                            createInviteRequest.getRole().name()));

            return ResponseEntity.ok(inviteMapper.toDto(invite));
        } catch (NoSuchAlgorithmException e) {
            return ResponseEntity.status(500).build();
        }
    }

    @Override
    public ResponseEntity<Void> deleteInvite(String token) {
        inviteService.deleteInvite(SecurityUtils.currentUserId(), token);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<List<InviteDto>> getInvites() {
        List<InviteDto> invites = inviteService.getInvitesByUser(SecurityUtils.currentUserId()).stream()
                .map(inviteMapper::toDto)
                .toList();
        return ResponseEntity.ok(invites);
    }
}
