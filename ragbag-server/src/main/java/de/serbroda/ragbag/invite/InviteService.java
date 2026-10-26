package de.serbroda.ragbag.invite;

import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import de.serbroda.ragbag.shared.exception.ResourceConflictException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.Space;
import de.serbroda.ragbag.space.SpaceMemberRole;
import de.serbroda.ragbag.space.SpacePermission;
import de.serbroda.ragbag.space.SpaceService;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import java.security.SecureRandom;
import java.time.Instant;
import java.util.Base64;
import java.util.List;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Transactional
@RequiredArgsConstructor
@Service
public class InviteService {

    private static final SecureRandom RANDOM = new SecureRandom();

    private final UserRepository userRepository;
    private final InviteRepository inviteRepository;
    private final SpaceService spaceService;

    public Invite createInvite(String userId, CreateInviteCommand cmd) {
        User user = userRepository.findById(userId).orElseThrow(() -> new IllegalArgumentException("User not found"));

        if (cmd.targetType() != InviteTargetType.SPACE) {
            throw new IllegalArgumentException("Only space invites are supported");
        }
        SpaceMemberRole role = cmd.role() != null ? cmd.role() : SpaceMemberRole.VIEWER;
        if (role == SpaceMemberRole.OWNER) {
            throw new IllegalArgumentException("Ownership cannot be granted via invite");
        }
        if (cmd.maxUses() != null && cmd.maxUses() < 1) {
            throw new IllegalArgumentException("maxUses must be at least 1");
        }
        if (cmd.expiresAt() != null && cmd.expiresAt().isBefore(Instant.now())) {
            throw new IllegalArgumentException("expiresAt must be in the future");
        }

        SpaceMemberRole inviterRole = spaceService
                .findRole(userId, cmd.targetId())
                .orElseThrow(() -> new AccessDeniedException("Not a member of the target space"));
        // Only member managers may invite, and nobody may hand out a role above their own
        if (!inviterRole.has(SpacePermission.MANAGE_MEMBERS) || !inviterRole.isAtLeast(role)) {
            throw new AccessDeniedException("User does not have permission to create this invite");
        }

        Invite invite = new Invite();
        invite.setCreatedBy(user);
        invite.setTargetType(cmd.targetType());
        invite.setTargetId(cmd.targetId());
        invite.setExpiresAt(cmd.expiresAt());
        invite.setMaxUses(cmd.maxUses() != null ? cmd.maxUses() : -1);
        invite.setToken(generateInviteToken());
        invite.setRole(role.name());

        return inviteRepository.save(invite);
    }

    @Transactional(readOnly = true)
    public InvitePreview previewInvite(String userId, String token) {
        Invite invite = requireInvite(token);
        String targetName =
                spaceService.findById(invite.getTargetId()).map(Space::getName).orElse(null);
        boolean alreadyMember =
                spaceService.findRole(userId, invite.getTargetId()).isPresent();
        return new InvitePreview(
                invite, targetName, invite.getCreatedBy().getUsername(), !invite.isExpired(), alreadyMember);
    }

    public InvitePreview acceptInvite(String userId, String token) {
        Invite invite = requireInvite(token);
        if (invite.isExpired()) {
            throw new ResourceConflictException("Invite is expired");
        }

        switch (invite.getTargetType()) {
            case SPACE ->
                spaceService.joinSpace(userId, invite.getTargetId(), SpaceMemberRole.valueOf(invite.getRole()));
            default -> throw new IllegalArgumentException("Unsupported invite target type");
        }

        // Optimistic locking (@Version) rejects concurrent acceptances that would exceed maxUses
        invite.setUsedCount(invite.getUsedCount() + 1);
        inviteRepository.save(invite);
        return previewInvite(userId, token);
    }

    @Transactional(readOnly = true)
    public List<Invite> getInvitesByUser(String userId) {
        return inviteRepository.findByCreatedBy_Id(userId);
    }

    @Transactional(readOnly = true)
    public List<Invite> getInvitesForSpace(String spaceId) {
        return inviteRepository.findByTargetTypeAndTargetIdOrderByCreatedAtDesc(InviteTargetType.SPACE, spaceId);
    }

    public void deleteInvite(String userId, String token) {
        Invite invite = requireInvite(token);
        boolean creator = invite.getCreatedBy().getId().equals(userId);
        boolean manager = invite.getTargetType() == InviteTargetType.SPACE
                && spaceService
                        .findRole(userId, invite.getTargetId())
                        .map(role -> role.has(SpacePermission.MANAGE_MEMBERS))
                        .orElse(false);
        if (!creator && !manager) {
            throw new AccessDeniedException("Missing permission to revoke this invite");
        }
        inviteRepository.delete(invite);
    }

    public Optional<Invite> findByToken(String token) {
        return inviteRepository.findByToken(token);
    }

    private Invite requireInvite(String token) {
        return inviteRepository.findByToken(token).orElseThrow(() -> new ResourceNotFoundException("Invite not found"));
    }

    private static String generateInviteToken() {
        byte[] randomBytes = new byte[32];
        RANDOM.nextBytes(randomBytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(randomBytes);
    }

    public record CreateInviteCommand(
            InviteTargetType targetType, String targetId, Instant expiresAt, Integer maxUses, SpaceMemberRole role) {}

    public record InvitePreview(
            Invite invite, String targetName, String invitedBy, boolean valid, boolean alreadyMember) {}
}
