package de.serbroda.ragbag.space;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.SpaceApi;
import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.generated.model.UpdateSpaceDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.util.List;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class SpaceController implements SpaceApi {

    private final SpaceService spaceService;
    private final SpaceMapper mapper;

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<SpaceDto> getSpace(String spaceId) {
        SpaceService.SpaceWithPermissions space = spaceService
                .findById(SecurityUtils.currentUserId(), spaceId)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
        return ResponseEntity.ok(mapper.map(space));
    }

    @Override
    public ResponseEntity<List<SpaceDto>> getSpaces() {
        Set<SpaceService.SpaceWithPermissions> spaces = spaceService.getSpacesForUser(SecurityUtils.currentUserId());
        return ResponseEntity.ok(spaces.stream().map(mapper::map).toList());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'WRITE')")
    @Override
    public ResponseEntity<SpaceDto> updateSpace(String spaceId, UpdateSpaceDto updateSpaceDto) {
        Space space = spaceService.updateSpace(
                spaceId,
                new SpaceService.UpdateSpaceCommand(updateSpaceDto.getName(), updateSpaceDto.getDescription()));
        return ResponseEntity.ok(SpaceDto.builder()
                .id(space.getId())
                .name(space.getName())
                .description(space.getDescription())
                .build());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'DELETE')")
    @Override
    public ResponseEntity<Void> deleteSpace(String spaceId) {
        spaceService.deleteSpace(spaceId);
        return ResponseEntity.noContent().build();
    }

    @Override
    public ResponseEntity<Void> leaveSpace(String spaceId) {
        spaceService.leaveSpace(SecurityUtils.currentUserId(), spaceId);
        return ResponseEntity.noContent().build();
    }
}
