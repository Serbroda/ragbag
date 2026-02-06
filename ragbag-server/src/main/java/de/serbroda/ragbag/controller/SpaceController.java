package de.serbroda.ragbag.controller;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.SpacesApi;
import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.service.SpaceService;
import de.serbroda.ragbag.service.UserService;
import java.util.List;
import java.util.Set;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class SpaceController implements SpacesApi {

    private final UserService userService;
    private final SpaceService spaceService;

    @Override
    public ResponseEntity<SpaceDto> getSpace(String spaceId) {
        return SpacesApi.super.getSpace(spaceId);
    }

    @Override
    public ResponseEntity<List<SpaceDto>> getSpaces() {
        Set<Space> spaces = spaceService.getSpacesForUser(SecurityUtils.currentUserId());
        return ResponseEntity.ok(spaces.stream()
                .map(s -> new SpaceDto(s.getId(), s.getName(), s.getDescription()))
                .toList());
    }
}
