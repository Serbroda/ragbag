package de.serbroda.ragbag.space;

import de.serbroda.ragbag.invite.InviteRepository;
import de.serbroda.ragbag.invite.InviteTargetType;
import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import de.serbroda.ragbag.shared.exception.ResourceConflictException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import java.util.Comparator;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Transactional
@RequiredArgsConstructor
@Service
public class SpaceService {

    private final UserService userService;
    private final SpaceRepository spaceRepository;
    private final SpaceMemberRepository spaceMemberRepository;
    private final InviteRepository inviteRepository;

    @Transactional(readOnly = true)
    public Optional<Space> findById(String spaceId) {
        return spaceRepository.findById(spaceId);
    }

    @Transactional(readOnly = true)
    public Optional<SpaceWithPermissions> findById(String userId, String spaceId) {
        return spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .map(member -> new SpaceWithPermissions(member.getSpace(), member.getRole()));
    }

    @Transactional(readOnly = true)
    public Optional<SpaceMemberRole> findRole(String userId, String spaceId) {
        return spaceMemberRepository.findBySpace_IdAndUser_Id(spaceId, userId).map(SpaceMember::getRole);
    }

    public Space createSpace(String userId, CreateSpaceCommand cmd) {
        User user = userService
                .findUserById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + userId));

        Space space = new Space();
        space.setName(requireName(cmd.name()));
        space.setDescription(cmd.description());
        space.setCreatedBy(user);

        space = spaceRepository.save(space);

        joinSpaceInternal(user, space, SpaceMemberRole.OWNER);

        return space;
    }

    public void joinSpace(String userId, String spaceId, SpaceMemberRole role) {
        User user = userService
                .findUserById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + userId));

        Space space = spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));

        if (role == null) {
            role = SpaceMemberRole.VIEWER;
        }

        joinSpaceInternal(user, space, role);
    }

    private void joinSpaceInternal(User user, Space space, SpaceMemberRole role) {
        SpaceMember member = spaceMemberRepository
                .findBySpaceAndUser_Id(space, user.getId())
                .orElseGet(() -> {
                    SpaceMember sm = new SpaceMember();
                    sm.setId(new SpaceMemberId(space.getId(), user.getId()));
                    sm.setSpace(space);
                    sm.setUser(user);
                    sm.setRole(null);
                    return sm;
                });

        // Joining never downgrades an existing membership
        if (member.getRole() == null || !member.getRole().isAtLeast(role)) {
            member.setRole(role);
        }
        space.getMembers().add(member);
        spaceRepository.save(space);
    }

    public Space updateSpace(String spaceId, UpdateSpaceCommand cmd) {
        Space space = spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        space.setName(requireName(cmd.name()));
        space.setDescription(cmd.description());
        return spaceRepository.save(space);
    }

    public void deleteSpace(String spaceId) {
        spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        inviteRepository.deleteByTargetTypeAndTargetId(InviteTargetType.SPACE, spaceId);
        spaceRepository.deleteById(spaceId);
    }

    public void leaveSpace(String userId, String spaceId) {
        SpaceMember member = spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .orElseThrow(() -> new ResourceNotFoundException("Not a member of space: " + spaceId));
        if (member.getRole() == SpaceMemberRole.OWNER && countOwners(member.getSpace()) <= 1) {
            throw new ResourceConflictException("The last owner cannot leave the space");
        }
        member.getSpace().getMembers().remove(member);
    }

    @Transactional(readOnly = true)
    public List<SpaceMember> getMembers(String spaceId) {
        Space space = spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        return space.getMembers().stream()
                .sorted(Comparator.comparing(SpaceMember::getRole)
                        .thenComparing(member -> member.getUser().getUsername()))
                .toList();
    }

    public SpaceMember updateMemberRole(String actorId, String spaceId, String userId, SpaceMemberRole newRole) {
        SpaceMember actor = requireMember(spaceId, actorId);
        SpaceMember target = requireMember(spaceId, userId);
        requireCanManage(actor, target);

        if (!actor.getRole().isAtLeast(newRole)) {
            throw new AccessDeniedException("You cannot assign a role above your own");
        }
        if (target.getRole() == SpaceMemberRole.OWNER
                && newRole != SpaceMemberRole.OWNER
                && countOwners(target.getSpace()) <= 1) {
            throw new ResourceConflictException("A space needs at least one owner");
        }

        target.setRole(newRole);
        return target;
    }

    public void removeMember(String actorId, String spaceId, String userId) {
        SpaceMember actor = requireMember(spaceId, actorId);
        SpaceMember target = requireMember(spaceId, userId);
        requireCanManage(actor, target);
        target.getSpace().getMembers().remove(target);
    }

    /**
     * Owners may manage everybody else; everyone else only members ranked strictly below them. Changing your
     * own membership goes through {@link #leaveSpace}.
     */
    private static void requireCanManage(SpaceMember actor, SpaceMember target) {
        if (actor.getUser().getId().equals(target.getUser().getId())) {
            throw new IllegalArgumentException("You cannot change your own membership; leave the space instead");
        }
        if (!actor.getRole().has(SpacePermission.MANAGE_MEMBERS)) {
            throw new AccessDeniedException("Missing permission to manage members");
        }
        boolean owner = actor.getRole() == SpaceMemberRole.OWNER;
        boolean outranks = actor.getRole().ordinal() < target.getRole().ordinal();
        if (!owner && !outranks) {
            throw new AccessDeniedException("You can only manage members with a lower role");
        }
    }

    private SpaceMember requireMember(String spaceId, String userId) {
        return spaceMemberRepository
                .findBySpace_IdAndUser_Id(spaceId, userId)
                .orElseThrow(() -> new ResourceNotFoundException("Member not found: " + userId));
    }

    private static long countOwners(Space space) {
        return space.getMembers().stream()
                .filter(member -> member.getRole() == SpaceMemberRole.OWNER)
                .count();
    }

    private static String requireName(String name) {
        if (name == null || name.isBlank()) {
            throw new IllegalArgumentException("Space name must not be empty");
        }
        String trimmed = name.trim();
        if (trimmed.length() > 80) {
            throw new IllegalArgumentException("Space name must be at most 80 characters");
        }
        return trimmed;
    }

    @Transactional(readOnly = true)
    public Set<SpaceWithPermissions> getSpacesForUser(String userId) {
        return spaceMemberRepository.findByUser_Id(userId).stream()
                .sorted(Comparator.comparing(SpaceMember::getRole)
                        .thenComparing(member -> member.getSpace().getName(), String.CASE_INSENSITIVE_ORDER))
                .map(member -> new SpaceWithPermissions(member.getSpace(), member.getRole()))
                .collect(Collectors.toCollection(LinkedHashSet::new));
    }

    public record CreateSpaceCommand(String name, String description) {
        public CreateSpaceCommand(String name) {
            this(name, null);
        }
    }

    public record UpdateSpaceCommand(String name, String description) {}

    public record SpaceWithPermissions(Space space, SpaceMemberRole role) {
        public Set<SpacePermission> permissions() {
            return role.permissions();
        }
    }
}
