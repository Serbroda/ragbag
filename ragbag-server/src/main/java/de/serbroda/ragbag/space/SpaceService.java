package de.serbroda.ragbag.space;

import de.serbroda.ragbag.shared.exception.AccessDeniedException;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import jakarta.transaction.Transactional;
import java.util.HashSet;
import java.util.Optional;
import java.util.Set;
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

    public Space createSpace(String userId, CreateSpaceCommand cmd) {
        User user = userService
                .findUserById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found: " + userId));

        Space space = new Space();
        space.setName(cmd.name());
        space.setCreatedBy(user);

        space = spaceRepository.save(space);

        joinSpaceInternal(user, space, SpaceMemberRole.ADMIN);

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

        if (role == SpaceMemberRole.VIEWER && !SpaceVisibility.PUBLIC.equals(space.getVisibility())) {
            throw new AccessDeniedException("Cannot join space as viewer because it is not public");
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

    public void leaveSpace(User user, Space space) {
        space.getMembers().removeIf(member -> member.getUser().getId().equals(user.getId()));
        spaceRepository.save(space);
    }

    public Set<Space> getSpacesForUser(String userId) {
        Set<Space> spaces = new HashSet<>(spaceRepository.findByCreatedBy_Id(userId));
        spaceMemberRepository.findByUser_Id(userId).forEach(sm -> {
            spaces.add(sm.getSpace());
        });
        return spaces;
    }

    public record CreateSpaceCommand(String name) {}
}
