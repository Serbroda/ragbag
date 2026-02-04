package de.serbroda.ragbag.controller;

import de.serbroda.ragbag.model.Space;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.model.dto.CollectionTreeDto;
import de.serbroda.ragbag.model.dto.SpaceDto;
import de.serbroda.ragbag.security.UserPrincipal;
import de.serbroda.ragbag.service.CollectionService;
import de.serbroda.ragbag.service.SpaceService;
import de.serbroda.ragbag.service.UserService;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.media.ArraySchema;
import io.swagger.v3.oas.annotations.media.Content;
import io.swagger.v3.oas.annotations.media.Schema;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import lombok.RequiredArgsConstructor;
import org.hibernate.annotations.Array;
import org.springframework.http.MediaType;
import org.springframework.http.ProblemDetail;
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

@SecurityRequirement(name = "basicAuth")
@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX + "/v1/spaces")
public class SpaceController {

    private final UserService userService;
    private final SpaceService spaceService;
    private final CollectionService collectionService;

    @Operation(
            summary = "Get all spaces",
            responses = {
                    @ApiResponse(
                            responseCode = "200",
                            description = "List of spaces",
                            content = @Content(
                                    mediaType = "application/json",
                                    array = @ArraySchema(
                                            schema = @Schema(implementation = SpaceDto.class)
                                    )
                            )
                    )
            }
    )
    @GetMapping(produces = MediaType.APPLICATION_JSON_VALUE)
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

    @Operation(
            responses = {
                    @ApiResponse(responseCode = "404", description = "Not found",
                            content = @Content(schema = @Schema(implementation = ProblemDetail.class))
                    )
            }
    )
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
