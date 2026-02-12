package de.serbroda.ragbag.space;

import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import jakarta.transaction.Transactional;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

@Transactional
@RequiredArgsConstructor
@Service
public class SpaceService {

    private final UserService userService;
    private final SpaceRepository spaceRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public Optional<Space> findById(String spaceId) {
        return spaceRepository.findById(spaceId);
    }

    public Optional<SpaceWithPermissions> findById(String userId, String spaceId) {
        return spaceRepository.findById(spaceId)
                .map(space -> {
                    SpaceMemberRole role = space.getMembers().stream()
                            .filter(member -> member.getUser().getId().equals(userId))
                            .map(SpaceMember::getRole)
                            .findFirst()
                            .orElse(SpaceMemberRole.VIEWER);
                    Set<SpacePermission> permissions = getPermissionsForRole(role);
                    return new SpaceWithPermissions(space, permissions);
                });
    }


    public Space createSpace(String userId, CreateSpaceCommand cmd) {
        User user = userService
                .findUserById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + userId));

        Space space = new Space();
        space.setName(cmd.name());
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
                    return sm;
                });

        //        if (member.getRole() == role) {
        //            return; // User already has the same role
        //        }

        member.setRole(role);
        space.getMembers().add(member);
        spaceRepository.save(space);
    }

    public Space updateSpace(String spaceId, UpdateSpaceCommand cmd) {
        Space space = spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        space.setName(cmd.name());
        space.setDescription(cmd.description());
        return spaceRepository.save(space);
    }

    public void deleteSpace(String spaceId) {
        spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        spaceRepository.deleteById(spaceId);
    }

    public void leaveSpace(String userId, String spaceId) {
        User user = userService
                .findUserById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found: " + userId));
        Space space = spaceRepository
                .findById(spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        space.getMembers().removeIf(member -> member.getUser().getId().equals(user.getId()));
        spaceRepository.save(space);
    }

    public Set<SpaceWithPermissions> getSpacesForUser(String userId) {
        return spaceMemberRepository.findByUser_Id(userId).stream()
                .sorted((a, b) ->
                        Boolean.compare(b.getRole() == SpaceMemberRole.OWNER, a.getRole() == SpaceMemberRole.OWNER))
                .map(member -> {
                    Space space = member.getSpace();
                    SpaceMemberRole role = member.getRole();
                    Set<SpacePermission> permissions = getPermissionsForRole(role);
                    return new SpaceWithPermissions(space, permissions);
                })
                .collect(Collectors.toCollection(java.util.LinkedHashSet::new));
    }

    private Set<SpacePermission> getPermissionsForRole(SpaceMemberRole role) {
        return switch (role) {
            case OWNER ->
                Set.of(
                        SpacePermission.READ,
                        SpacePermission.EDIT_SPACE,
                        SpacePermission.CREATE_COLLECTIONS,
                        SpacePermission.EDIT_COLLECTIONS,
                        SpacePermission.DELETE_COLLECTIONS,
                        SpacePermission.CREATE_BOOKMARKS,
                        SpacePermission.EDIT_BOOKMARKS,
                        SpacePermission.DELETE_BOOKMARKS);
            case ADMIN ->
                Set.of(
                        SpacePermission.READ,
                        SpacePermission.EDIT_SPACE,
                        SpacePermission.CREATE_COLLECTIONS,
                        SpacePermission.EDIT_COLLECTIONS,
                        SpacePermission.DELETE_COLLECTIONS,
                        SpacePermission.CREATE_BOOKMARKS,
                        SpacePermission.EDIT_BOOKMARKS,
                        SpacePermission.DELETE_BOOKMARKS);
            case CONTRIBUTOR ->
                Set.of(
                        SpacePermission.READ,
                        SpacePermission.EDIT_SPACE,
                        SpacePermission.CREATE_COLLECTIONS,
                        SpacePermission.EDIT_COLLECTIONS,
                        SpacePermission.DELETE_COLLECTIONS,
                        SpacePermission.CREATE_BOOKMARKS,
                        SpacePermission.EDIT_BOOKMARKS,
                        SpacePermission.DELETE_BOOKMARKS);
            case VIEWER -> Set.of(SpacePermission.READ);
        };
    }

    public record CreateSpaceCommand(String name) {}

    public record UpdateSpaceCommand(String name, String description) {}

    public record SpaceWithPermissions(Space space, Set<SpacePermission> permissions) {}
}
