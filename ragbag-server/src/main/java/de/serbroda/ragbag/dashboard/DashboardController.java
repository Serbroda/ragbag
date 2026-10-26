package de.serbroda.ragbag.dashboard;

import static de.serbroda.ragbag.security.permission.DomainPermissionEvaluator.DOMAIN_PREFIX_SPACE;
import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.bookmark.Bookmark;
import de.serbroda.ragbag.bookmark.BookmarkMapper;
import de.serbroda.ragbag.generated.api.DashboardApi;
import de.serbroda.ragbag.generated.model.BookmarkDto;
import de.serbroda.ragbag.generated.model.DashboardDto;
import de.serbroda.ragbag.generated.model.DashboardSectionDto;
import de.serbroda.ragbag.generated.model.DashboardSectionTypeEnum;
import de.serbroda.ragbag.generated.model.UpdateDashboardDto;
import de.serbroda.ragbag.security.SecurityUtils;
import java.util.List;
import java.util.Map;
import java.util.function.Function;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Every member, viewers included, may arrange their own dashboard. */
@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class DashboardController implements DashboardApi {

    private final DashboardService dashboardService;
    private final BookmarkMapper bookmarkMapper;

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<DashboardDto> getDashboard(String spaceId) {
        String userId = SecurityUtils.currentUserId();
        return ResponseEntity.ok(toDto(dashboardService.getDashboard(userId, spaceId), userId));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<DashboardDto> updateDashboard(String spaceId, UpdateDashboardDto updateDashboardDto) {
        String userId = SecurityUtils.currentUserId();
        List<DashboardService.SectionCommand> commands = updateDashboardDto.getSections().stream()
                .map(section -> new DashboardService.SectionCommand(
                        section.getTitle(),
                        section.getType() != null
                                ? DashboardSectionType.valueOf(section.getType().getValue())
                                : null,
                        section.getBookmarkIds()))
                .toList();
        return ResponseEntity.ok(toDto(dashboardService.updateDashboard(userId, spaceId, commands), userId));
    }

    @PreAuthorize("hasPermission(#spaceId, '" + DOMAIN_PREFIX_SPACE + "', 'READ')")
    @Override
    public ResponseEntity<Void> resetDashboard(String spaceId) {
        dashboardService.resetDashboard(SecurityUtils.currentUserId(), spaceId);
        return ResponseEntity.noContent().build();
    }

    private DashboardDto toDto(DashboardService.Dashboard dashboard, String userId) {
        // Map every bookmark once so favorite flags are resolved in a single query
        List<Bookmark> all = dashboard.sections().stream()
                .flatMap(section -> section.bookmarks().stream())
                .distinct()
                .toList();
        Map<String, BookmarkDto> dtos = bookmarkMapper.toDtos(all, userId).stream()
                .collect(Collectors.toMap(BookmarkDto::getId, Function.identity()));

        return new DashboardDto()
                .customized(dashboard.customized())
                .sections(dashboard.sections().stream()
                        .map(section -> new DashboardSectionDto()
                                .id(section.id())
                                .title(section.title())
                                .type(DashboardSectionTypeEnum.fromValue(
                                        section.type().name()))
                                .bookmarks(section.bookmarks().stream()
                                        .map(bookmark -> dtos.get(bookmark.getId()))
                                        .toList()))
                        .toList());
    }
}
