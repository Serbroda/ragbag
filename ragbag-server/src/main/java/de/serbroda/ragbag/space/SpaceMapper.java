package de.serbroda.ragbag.space;

import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.generated.model.SpaceMemberDto;
import de.serbroda.ragbag.generated.model.SpaceMemberRoleEnum;
import de.serbroda.ragbag.generated.model.SpacePermissionEnum;
import org.springframework.stereotype.Component;

@Component
public class SpaceMapper {

    public SpaceDto map(SpaceService.SpaceWithPermissions spaceWithPermissions) {
        Space space = spaceWithPermissions.space();
        return SpaceDto.builder()
                .id(space.getId())
                .name(space.getName())
                .description(space.getDescription())
                .iconId(space.getIconImageId())
                .role(map(spaceWithPermissions.role()))
                .permissions(spaceWithPermissions.permissions().stream()
                        .sorted()
                        .map(this::map)
                        .toList())
                .build();
    }

    public SpaceMemberDto map(SpaceMember member) {
        return SpaceMemberDto.builder()
                .userId(member.getUser().getId())
                .username(member.getUser().getUsername())
                .avatarId(member.getUser().getAvatarImageId())
                .role(map(member.getRole()))
                .build();
    }

    public SpacePermissionEnum map(SpacePermission entityPermission) {
        return SpacePermissionEnum.fromValue(entityPermission.name());
    }

    public SpaceMemberRoleEnum map(SpaceMemberRole role) {
        return SpaceMemberRoleEnum.fromValue(role.name());
    }

    public SpaceMemberRole map(SpaceMemberRoleEnum role) {
        return SpaceMemberRole.valueOf(role.getValue());
    }
}
