package de.serbroda.ragbag.invite;

import de.serbroda.ragbag.collection.CollectionService;
import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.space.SpaceMember;
import de.serbroda.ragbag.space.SpaceMemberRole;
import de.serbroda.ragbag.space.SpaceService;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserRepository;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.Date;
import java.util.List;
import java.util.Optional;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@RequiredArgsConstructor
@Service
public class InviteService {

    private final UserRepository userRepository;
    private final InviteRepository inviteRepository;
    private final SpaceService spaceService;
    private final CollectionService collectionService;

    public Invite createInvite(String userId, CreateInviteCommand cmd) throws NoSuchAlgorithmException {
        User user = userRepository.findById(userId).orElseThrow(() -> new IllegalArgumentException("User not found"));

        // TODO: check if user is allowed to create invite for the target (space/collection)

        final InviteTargetType targetType = cmd.targetType();
        final String targetId = cmd.targetId();

        if (!checkPermission(user, targetType, targetId)) {
            throw new AccessDeniedException("User does not have permission to create invite for the target");
        }

        Invite invite = new Invite();
        invite.setCreatedBy(user);
        invite.setTargetType(cmd.targetType());
        invite.setTargetId(cmd.targetId());
        invite.setExpiresAt(cmd.expiresAt());
        invite.setMaxUses(cmd.maxUses() != null ? cmd.maxUses() : -1);
        invite.setToken(generateInviteToken());
        invite.setRole(cmd.role());

        return inviteRepository.save(invite);
    }

    private boolean checkPermission(User user, final InviteTargetType targetType, final String targetId) {
        return switch (targetType) {
            case SPACE -> {
                Optional<SpaceMemberRole> roleOpt = spaceService
                        .findById(targetId)
                        .flatMap(space -> space.getMembers().stream()
                                .filter(member -> member.getUser().getId().equals(user.getId()))
                                .findFirst()
                                .map(SpaceMember::getRole));
                yield roleOpt.map(role -> role == SpaceMemberRole.OWNER || role == SpaceMemberRole.ADMIN)
                        .orElse(false);
            }
            // case COLLECTION -> collectionService.findById(targetId).isPresent();
            default -> false;
        };
    }

    public void acceptInvite(String userId, String token) {
        Invite invite =
                inviteRepository.findByToken(token).orElseThrow(() -> new IllegalArgumentException("Invite not found"));
        if (invite.isExpired()) {
            throw new IllegalStateException("Invite is expired");
        }

        switch (invite.getTargetType()) {
            case SPACE ->
                spaceService.joinSpace(userId, invite.getTargetId(), SpaceMemberRole.valueOf(invite.getRole()));
            // case COLLECTION -> collectionService.joinCollection(userId, invite.getTargetId(), invite.getRole());
            default -> throw new IllegalStateException("Unknown invite target type");
        }

        invite.setUsedCount(invite.getUsedCount() + 1);
        inviteRepository.save(invite);
    }

    public List<Invite> getInvitesByUser(String userId) {
        return inviteRepository.findByCreatedBy_Id(userId);
    }

    public void deleteInvite(String userId, String token) {
        Invite invite = inviteRepository
                .findByToken(token)
                .orElseThrow(() -> new ResourceNotFoundException("Invite not found"));
        if (!invite.getCreatedBy().getId().equals(userId)) {
            throw new AccessDeniedException("User is not the creator of this invite");
        }
        inviteRepository.delete(invite);
    }

    public Optional<Invite> findByToken(String token) {
        return inviteRepository.findByToken(token);
    }

    private String generateInviteToken() throws NoSuchAlgorithmException {
        byte[] randomBytes = new byte[32];
        SecureRandom.getInstanceStrong().nextBytes(randomBytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(randomBytes);
    }

    public record CreateInviteCommand(
            InviteTargetType targetType, String targetId, Date expiresAt, Integer maxUses, String role) {}
}
