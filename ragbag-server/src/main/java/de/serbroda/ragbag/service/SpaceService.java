package de.serbroda.ragbag.service;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.SpaceMember;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.model.keys.SpaceMemberId;
import de.serbroda.ragbag.model.shared.SpaceMemberRole;
import de.serbroda.ragbag.repository.SpaceMemberRepository;
import de.serbroda.ragbag.repository.SpaceRepository;
import jakarta.transaction.Transactional;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.util.HashSet;
import java.util.Set;

@Transactional
@RequiredArgsConstructor
@Service
public class SpaceService {

    private final SpaceRepository spaceRepository;
    private final SpaceMemberRepository spaceMemberRepository;

    public Space createSpace(User user, String name) {
        Space space = new Space();
        space.setName(name);
        space.setCreatedBy(user);

        space = spaceRepository.save(space);

        joinSpace(user, space, SpaceMemberRole.ADMIN);

        return space;
    }

    public void joinSpace(User user, Space space, SpaceMemberRole role) {
        SpaceMember member = spaceMemberRepository.findBySpaceAndUser_Id(space, user.getId())
                .orElseGet(() -> {
                    SpaceMember sm = new SpaceMember();
                    sm.setId(new SpaceMemberId(space.getId(), user.getId()));
                    sm.setSpace(space);
                    sm.setUser(user);
                    return sm;
                });

        if (member.getRole() == role) {
            return; // User already has the same role
        }

        member.setRole(role);
        space.getMembers().add(member);
        spaceRepository.save(space);
    }

    public void leaveSpace(User user, Space space) {
        space.getMembers().removeIf(member -> member.getUser().getId().equals(user.getId()));
        spaceRepository.save(space);
    }

    public Set<Space> getSpacesForUser(User user) {
        Set<Space> spaces = new HashSet<>(spaceRepository.findByCreatedBy(user));
        spaceMemberRepository.findByUser_Id(user.getId()).forEach(sm -> spaces.add(sm.getSpace()));
        return spaces;
    }

}
