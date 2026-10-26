package de.serbroda.ragbag.invite;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.InviteApi;
import de.serbroda.ragbag.generated.model.CreateInviteRequest;
import de.serbroda.ragbag.generated.model.InviteDto;
import de.serbroda.ragbag.generated.model.InvitePreviewDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.space.SpaceMemberRole;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RequestMapping(PUBLIC_API_PREFIX)
@RestController
public class InviteController implements InviteApi {

    private final InviteService inviteService;
    private final InviteMapper inviteMapper;

    @Override
    public ResponseEntity<InvitePreviewDto> getInvite(String token) {
        return ResponseEntity.ok(inviteMapper.toDto(inviteService.previewInvite(SecurityUtils.currentUserId(), token)));
    }

    @Override
    public ResponseEntity<InvitePreviewDto> acceptInvite(String token) {
        return ResponseEntity.ok(inviteMapper.toDto(inviteService.acceptInvite(SecurityUtils.currentUserId(), token)));
    }

    @Override
    public ResponseEntity<InviteDto> createInvite(CreateInviteRequest createInviteRequest) {
        Invite invite = inviteService.createInvite(
                SecurityUtils.currentUserId(),
                new InviteService.CreateInviteCommand(
                        InviteTargetType.valueOf(
                                createInviteRequest.getTargetType().name()),
                        createInviteRequest.getTargetId(),
                        createInviteRequest.getExpiresAt() != null
                                ? createInviteRequest.getExpiresAt().toInstant()
                                : null,
                        createInviteRequest.getMaxUses(),
                        createInviteRequest.getRole() != null
                                ? SpaceMemberRole.valueOf(
                                        createInviteRequest.getRole().getValue())
                                : null));

        return ResponseEntity.status(HttpStatus.CREATED).body(inviteMapper.toDto(invite));
    }

    @Override
    public ResponseEntity<Void> deleteInvite(String token) {
        inviteService.deleteInvite(SecurityUtils.currentUserId(), token);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<List<InviteDto>> getInvites() {
        return ResponseEntity.ok(inviteService.getInvitesByUser(SecurityUtils.currentUserId()).stream()
                .map(inviteMapper::toDto)
                .toList());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'MANAGE_MEMBERS')")
    @Override
    public ResponseEntity<List<InviteDto>> getSpaceInvites(String spaceId) {
        return ResponseEntity.ok(inviteService.getInvitesForSpace(spaceId).stream()
                .map(inviteMapper::toDto)
                .toList());
    }
}
