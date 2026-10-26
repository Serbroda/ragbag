package de.serbroda.ragbag.dashboard;

import de.serbroda.ragbag.bookmark.Bookmark;
import de.serbroda.ragbag.bookmark.BookmarkFavoriteRepository;
import de.serbroda.ragbag.bookmark.BookmarkRepository;
import de.serbroda.ragbag.bookmark.BookmarkService;
import de.serbroda.ragbag.collection.CollectionAccessService;
import de.serbroda.ragbag.space.SpaceMembershipEndedEvent;
import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Function;
import java.util.stream.Collectors;
import lombok.RequiredArgsConstructor;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Transactional
@RequiredArgsConstructor
@Service
public class DashboardService {

    static final int MAX_SECTIONS = 30;
    static final int MAX_LINKS_PER_SECTION = 100;
    static final int RECENT_COUNT = 12;

    private final DashboardSectionRepository sectionRepository;
    private final BookmarkRepository bookmarkRepository;
    private final BookmarkFavoriteRepository favoriteRepository;
    private final BookmarkService bookmarkService;
    private final CollectionAccessService collectionAccessService;

    /** The user's dashboard, or the default layout (favorites + recent) until they customize it. */
    @Transactional(readOnly = true)
    public Dashboard getDashboard(String userId, String spaceId) {
        List<DashboardSection> sections = sectionRepository.findByUserIdAndSpaceIdOrderByPosition(userId, spaceId);
        // Pinned links stay pinned when their collection turns private, but are only shown to those who can see it
        Set<String> hidden = collectionAccessService.hiddenCollectionIds(spaceId, userId);
        if (sections.isEmpty()) {
            return new Dashboard(
                    false,
                    List.of(
                            resolve(userId, spaceId, null, "Favorites", DashboardSectionType.FAVORITES, List.of()),
                            resolve(userId, spaceId, null, "Recently added", DashboardSectionType.RECENT, List.of())));
        }
        return new Dashboard(
                true,
                sections.stream()
                        .map(section -> resolve(
                                userId,
                                spaceId,
                                section.getId(),
                                section.getTitle(),
                                section.getType(),
                                section.getItems().stream()
                                        .map(DashboardItem::getBookmark)
                                        .filter(bookmark -> !hidden.contains(
                                                bookmark.getCollection().getId()))
                                        .toList()))
                        .toList());
    }

    public Dashboard updateDashboard(String userId, String spaceId, List<SectionCommand> commands) {
        if (commands.size() > MAX_SECTIONS) {
            throw new IllegalArgumentException("A dashboard can have at most " + MAX_SECTIONS + " sections");
        }

        // Resolve all referenced bookmarks at once and make sure they belong to this space
        List<String> referencedIds = commands.stream()
                .filter(cmd -> cmd.type() == DashboardSectionType.LINKS && cmd.bookmarkIds() != null)
                .flatMap(cmd -> cmd.bookmarkIds().stream())
                .distinct()
                .toList();
        Set<String> hidden = collectionAccessService.hiddenCollectionIds(spaceId, userId);
        Map<String, Bookmark> bookmarks = bookmarkRepository.findAllById(referencedIds).stream()
                .filter(bookmark -> bookmark.getCollection().getSpace().getId().equals(spaceId))
                .filter(bookmark -> !hidden.contains(bookmark.getCollection().getId()))
                .collect(Collectors.toMap(Bookmark::getId, Function.identity()));
        if (bookmarks.size() != referencedIds.size()) {
            throw new IllegalArgumentException("Dashboard links must be bookmarks of this space");
        }

        sectionRepository.deleteByUserIdAndSpaceId(userId, spaceId);
        sectionRepository.flush();

        List<DashboardSection> sections = new ArrayList<>();
        for (int i = 0; i < commands.size(); i++) {
            SectionCommand cmd = commands.get(i);
            DashboardSection section = new DashboardSection();
            section.setUserId(userId);
            section.setSpaceId(spaceId);
            section.setTitle(requireTitle(cmd.title()));
            if (cmd.type() == null) {
                throw new IllegalArgumentException("Section type is required");
            }
            section.setType(cmd.type());
            section.setPosition(i);

            if (cmd.type() == DashboardSectionType.LINKS && cmd.bookmarkIds() != null) {
                // Keep the first occurrence if a link was added twice
                List<String> ids = new ArrayList<>(new LinkedHashSet<>(cmd.bookmarkIds()));
                if (ids.size() > MAX_LINKS_PER_SECTION) {
                    throw new IllegalArgumentException(
                            "A section can hold at most " + MAX_LINKS_PER_SECTION + " links");
                }
                for (int position = 0; position < ids.size(); position++) {
                    DashboardItem item = new DashboardItem();
                    item.setSection(section);
                    item.setBookmark(bookmarks.get(ids.get(position)));
                    item.setPosition(position);
                    section.getItems().add(item);
                }
            }
            sections.add(section);
        }
        sectionRepository.saveAll(sections);
        return getDashboard(userId, spaceId);
    }

    public void resetDashboard(String userId, String spaceId) {
        sectionRepository.deleteByUserIdAndSpaceId(userId, spaceId);
    }

    /** Personal data of former members would otherwise linger and point at a space they cannot see. */
    @EventListener
    public void onMembershipEnded(SpaceMembershipEndedEvent event) {
        sectionRepository.deleteByUserIdAndSpaceId(event.userId(), event.spaceId());
        favoriteRepository.deleteAllInSpace(event.userId(), event.spaceId());
    }

    private Section resolve(
            String userId, String spaceId, String id, String title, DashboardSectionType type, List<Bookmark> pinned) {
        List<Bookmark> bookmarks = switch (type) {
            case LINKS -> pinned;
            case FAVORITES -> bookmarkService.getFavoritesInSpace(userId, spaceId);
            case RECENT -> bookmarkService.getRecentInSpace(spaceId, userId, RECENT_COUNT);
        };
        return new Section(id, title, type, bookmarks);
    }

    private static String requireTitle(String title) {
        if (title == null || title.isBlank()) {
            throw new IllegalArgumentException("Section title must not be empty");
        }
        String trimmed = title.trim();
        if (trimmed.length() > 80) {
            throw new IllegalArgumentException("Section title must be at most 80 characters");
        }
        return trimmed;
    }

    public record SectionCommand(String title, DashboardSectionType type, List<String> bookmarkIds) {}

    public record Section(String id, String title, DashboardSectionType type, List<Bookmark> bookmarks) {}

    public record Dashboard(boolean customized, List<Section> sections) {}
}
