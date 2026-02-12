package de.serbroda.ragbag.space;

import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.generated.model.SpacePermissionEnum;
import org.springframework.stereotype.Component;

@Component
public class SpaceMapper {

    public SpaceDto map(Space space) {
        return SpaceDto.builder()
                .id(space.getId())
                .name(space.getName())
                .description(space.getDescription())
                .build();
    }

    public SpaceDto map(SpaceService.SpaceWithPermissions spaceWithPermissions) {
        Space space = spaceWithPermissions.space();
        return SpaceDto.builder()
                .id(space.getId())
                .name(space.getName())
                .description(space.getDescription())
                .permissions(spaceWithPermissions.permissions().stream()
                        .map(this::map)
                        .toList())
                .build();
    }

    public SpacePermissionEnum map(SpacePermission entityPermission) {
        return SpacePermissionEnum.fromValue(entityPermission.name());
    }
}
