package de.serbroda.ragbag.space;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;
import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;

import de.serbroda.ragbag.generated.api.SpacesApi;
import de.serbroda.ragbag.generated.model.SpaceDto;
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
public class SpaceController implements SpacesApi {

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

    @Override
    public ResponseEntity<Void> joinSpace(String spaceId) {
        spaceService.joinSpace(SecurityUtils.currentUserId(), spaceId, null);
        return ResponseEntity.noContent().build();
    }
}
