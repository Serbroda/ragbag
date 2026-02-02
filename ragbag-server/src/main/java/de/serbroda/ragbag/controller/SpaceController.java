package de.serbroda.ragbag.controller;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.model.dto.CollectionTreeDto;
import de.serbroda.ragbag.model.dto.SpaceDto;
import de.serbroda.ragbag.security.UserPrincipal;
import de.serbroda.ragbag.service.CollectionService;
import de.serbroda.ragbag.service.SpaceService;
import de.serbroda.ragbag.service.UserService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Optional;
import java.util.Set;

import static de.serbroda.ragbag.config.AppConstants.PUBLIC_API_PREFIX;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/spaces")
public class SpaceController {

    private final UserService userService;
    private final SpaceService spaceService;
    private final CollectionService collectionService;

    @GetMapping
    public ResponseEntity<List<SpaceDto>> getSpaces() {
        Optional<User> admin = userService.findUserByUsernameOrEmail("admin");
        if (admin.isEmpty()) {
            return ResponseEntity.status(404).body(List.of());
        }

        Set<Space> spaces = spaceService.getSpacesForUser(admin.get());
        return ResponseEntity.ok(spaces.stream()
                .map(s -> new SpaceDto(
                        s.getId(),
                        s.getName(),
                        s.getDescription()
                ))
                .toList()
        );
    }

    @GetMapping("/{spaceId}/collections")
    public List<CollectionTreeDto> getCollections(
            @PathVariable String spaceId,
            Authentication authentication
    ) {
        UserPrincipal principal = (UserPrincipal) authentication.getPrincipal();
        return collectionService.getAllowedCollectionTree(
                spaceId,
                principal.getUserId()
        );
    }
}
