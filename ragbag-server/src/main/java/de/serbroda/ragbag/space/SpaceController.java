package de.serbroda.ragbag.space;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.SpaceApi;
import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.generated.model.UpdateSpaceDto;
import de.serbroda.ragbag.security.SecurityUtils;
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

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<SpaceDto> getSpace(String spaceId) {
        return spaceService
                .findById(spaceId)
                .map(space -> ResponseEntity.ok(new SpaceDto(space.getId(), space.getName(), space.getDescription())))
                .orElse(ResponseEntity.notFound().build());
    }

    @Override
    public ResponseEntity<List<SpaceDto>> getSpaces() {
        Set<Space> spaces = spaceService.getSpacesForUser(SecurityUtils.currentUserId());
        return ResponseEntity.ok(spaces.stream()
                .map(s -> new SpaceDto(s.getId(), s.getName(), s.getDescription()))
                .toList());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'WRITE')")
    @Override
    public ResponseEntity<SpaceDto> updateSpace(String spaceId, UpdateSpaceDto updateSpaceDto) {
        Space space = spaceService.updateSpace(
                spaceId,
                new SpaceService.UpdateSpaceCommand(updateSpaceDto.getName(), updateSpaceDto.getDescription()));
        return ResponseEntity.ok(new SpaceDto(space.getId(), space.getName(), space.getDescription()));
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
