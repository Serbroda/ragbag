package de.serbroda.ragbag.space;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.SpaceApi;
import de.serbroda.ragbag.generated.model.CreateSpaceDto;
import de.serbroda.ragbag.generated.model.SpaceDto;
import de.serbroda.ragbag.generated.model.SpaceMemberDto;
import de.serbroda.ragbag.generated.model.UpdateSpaceDto;
import de.serbroda.ragbag.generated.model.UpdateSpaceMemberDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.shared.exception.ResourceNotFoundException;
import java.util.List;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class SpaceController implements SpaceApi {

    private final SpaceService spaceService;
    private final SpaceMapper mapper;

    @Override
    public ResponseEntity<SpaceDto> createSpace(CreateSpaceDto createSpaceDto) {
        String userId = SecurityUtils.currentUserId();
        Space space = spaceService.createSpace(
                userId, new SpaceService.CreateSpaceCommand(createSpaceDto.getName(), createSpaceDto.getDescription()));
        return ResponseEntity.status(HttpStatus.CREATED).body(currentUserView(space.getId()));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<SpaceDto> getSpace(String spaceId) {
        return ResponseEntity.ok(currentUserView(spaceId));
    }

    @Override
    public ResponseEntity<List<SpaceDto>> getSpaces() {
        return ResponseEntity.ok(spaceService.getSpacesForUser(SecurityUtils.currentUserId()).stream()
                .map(mapper::map)
                .toList());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'EDIT_SPACE')")
    @Override
    public ResponseEntity<SpaceDto> updateSpace(String spaceId, UpdateSpaceDto updateSpaceDto) {
        spaceService.updateSpace(
                spaceId,
                new SpaceService.UpdateSpaceCommand(updateSpaceDto.getName(), updateSpaceDto.getDescription()));
        return ResponseEntity.ok(currentUserView(spaceId));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'EDIT_SPACE')")
    @Override
    public ResponseEntity<SpaceDto> setSpaceIcon(String spaceId, MultipartFile file, String url) {
        spaceService.setIcon(spaceId, file, url);
        return ResponseEntity.ok(currentUserView(spaceId));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'EDIT_SPACE')")
    @Override
    public ResponseEntity<SpaceDto> removeSpaceIcon(String spaceId) {
        spaceService.removeIcon(spaceId);
        return ResponseEntity.ok(currentUserView(spaceId));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'DELETE_SPACE')")
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

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<List<SpaceMemberDto>> getSpaceMembers(String spaceId) {
        return ResponseEntity.ok(
                spaceService.getMembers(spaceId).stream().map(mapper::map).toList());
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'MANAGE_MEMBERS')")
    @Override
    public ResponseEntity<SpaceMemberDto> updateSpaceMember(
            String spaceId, String userId, UpdateSpaceMemberDto updateSpaceMemberDto) {
        SpaceMember member = spaceService.updateMemberRole(
                SecurityUtils.currentUserId(), spaceId, userId, mapper.map(updateSpaceMemberDto.getRole()));
        return ResponseEntity.ok(mapper.map(member));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'MANAGE_MEMBERS')")
    @Override
    public ResponseEntity<Void> removeSpaceMember(String spaceId, String userId) {
        spaceService.removeMember(SecurityUtils.currentUserId(), spaceId, userId);
        return ResponseEntity.noContent().build();
    }

    private SpaceDto currentUserView(String spaceId) {
        return spaceService
                .findById(SecurityUtils.currentUserId(), spaceId)
                .map(mapper::map)
                .orElseThrow(() -> new ResourceNotFoundException("Space not found: " + spaceId));
    }
}
