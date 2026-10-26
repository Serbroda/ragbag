package de.serbroda.ragbag.dashboard;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import de.serbroda.ragbag.AbstractApiTest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

class DashboardApiTests extends AbstractApiTest {

    private Session owner;
    private String spaceId;
    private String collectionId;

    @BeforeEach
    void setUp() throws Exception {
        owner = registerAndLogin();
        spaceId = firstSpaceId(owner);
        collectionId = createCollection(owner, spaceId, "Homelab", null);
    }

    @Test
    void defaultDashboard_showsFavoritesAndRecent() throws Exception {
        String bookmark = createBookmark(owner, collectionId);

        getDashboard(owner)
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.customized").value(false))
                .andExpect(jsonPath("$.sections[0].type").value("FAVORITES"))
                .andExpect(jsonPath("$.sections[0].bookmarks.length()").value(0))
                .andExpect(jsonPath("$.sections[1].type").value("RECENT"))
                .andExpect(jsonPath("$.sections[1].bookmarks[0].id").value(bookmark));
    }

    @Test
    void favorites_arePersonal_andFlaggedOnBookmarks() throws Exception {
        Session viewer = registerAndLogin();
        join(viewer, spaceId, "VIEWER", owner);
        String bookmark = createBookmark(owner, collectionId);

        // Viewers may star links even though they cannot edit them
        mockMvc.perform(put("/api/v1/bookmarks/{id}/favorite", bookmark)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isNoContent());

        mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(jsonPath("$[0].favorite").value(true));
        mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(owner.accessToken())))
                .andExpect(jsonPath("$[0].favorite").value(false));
        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(jsonPath("$.length()").value(1));
        getDashboard(viewer).andExpect(jsonPath("$.sections[0].bookmarks[0].id").value(bookmark));

        mockMvc.perform(delete("/api/v1/bookmarks/{id}/favorite", bookmark)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isNoContent());
        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void customDashboard_keepsOrder_andIsPerUser() throws Exception {
        String first = createBookmark(owner, collectionId);
        String second = createBookmark(owner, collectionId);

        saveDashboard(owner, """
						{"sections":[
						{"title":"Homelab","type":"LINKS","bookmarkIds":["%s","%s","%s"]},
						{"title":"Starred","type":"FAVORITES"}
						]}
						""".formatted(second, first, second))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.customized").value(true))
                .andExpect(jsonPath("$.sections[0].title").value("Homelab"))
                .andExpect(jsonPath("$.sections[0].bookmarks.length()").value(2))
                .andExpect(jsonPath("$.sections[0].bookmarks[0].id").value(second))
                .andExpect(jsonPath("$.sections[0].bookmarks[1].id").value(first))
                .andExpect(jsonPath("$.sections[1].type").value("FAVORITES"));

        Session other = registerAndLogin();
        join(other, spaceId, "CONTRIBUTOR", owner);
        getDashboard(other).andExpect(jsonPath("$.customized").value(false));

        mockMvc.perform(delete("/api/v1/spaces/{id}/dashboard", spaceId)
                        .header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());
        getDashboard(owner).andExpect(jsonPath("$.customized").value(false));
    }

    @Test
    void customDashboard_rejectsBookmarksOfOtherSpaces() throws Exception {
        Session stranger = registerAndLogin();
        String foreign = createBookmark(stranger, createCollection(stranger, firstSpaceId(stranger), "Private", null));

        saveDashboard(owner, """
						{"sections":[{"title":"Sneaky","type":"LINKS","bookmarkIds":["%s"]}]}
						""".formatted(foreign)).andExpect(status().isBadRequest());
    }

    @Test
    void customDashboard_rejectsBlankTitles() throws Exception {
        saveDashboard(owner, "{\"sections\":[{\"title\":\" \",\"type\":\"RECENT\"}]}")
                .andExpect(status().isBadRequest());
    }

    @Test
    void deletingBookmark_removesItFromDashboardAndFavorites() throws Exception {
        String bookmark = createBookmark(owner, collectionId);
        mockMvc.perform(
                put("/api/v1/bookmarks/{id}/favorite", bookmark).header("Authorization", bearer(owner.accessToken())));
        saveDashboard(owner, """
						{"sections":[{"title":"Pinned","type":"LINKS","bookmarkIds":["%s"]}]}
						""".formatted(bookmark)).andExpect(status().isOk());

        mockMvc.perform(delete("/api/v1/bookmarks/{id}", bookmark).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());

        getDashboard(owner)
                .andExpect(jsonPath("$.customized").value(true))
                .andExpect(jsonPath("$.sections[0].bookmarks.length()").value(0));
        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(owner.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void leavingSpace_removesPersonalDashboardAndFavorites() throws Exception {
        Session member = registerAndLogin();
        join(member, spaceId, "VIEWER", owner);
        String bookmark = createBookmark(owner, collectionId);
        mockMvc.perform(
                put("/api/v1/bookmarks/{id}/favorite", bookmark).header("Authorization", bearer(member.accessToken())));
        saveDashboard(member, "{\"sections\":[{\"title\":\"Mine\",\"type\":\"RECENT\"}]}")
                .andExpect(status().isOk());

        mockMvc.perform(post("/api/v1/spaces/{id}/leave", spaceId)
                        .header("Authorization", bearer(member.accessToken())))
                .andExpect(status().isNoContent());
        join(member, spaceId, "VIEWER", owner);

        getDashboard(member).andExpect(jsonPath("$.customized").value(false));
        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(member.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void dashboard_isHiddenFromNonMembers() throws Exception {
        getDashboard(registerAndLogin()).andExpect(status().isForbidden());
    }

    private ResultActions getDashboard(Session session) throws Exception {
        return mockMvc.perform(
                get("/api/v1/spaces/{id}/dashboard", spaceId).header("Authorization", bearer(session.accessToken())));
    }

    private ResultActions saveDashboard(Session session, String body) throws Exception {
        return mockMvc.perform(put("/api/v1/spaces/{id}/dashboard", spaceId)
                .header("Authorization", bearer(session.accessToken()))
                .contentType(MediaType.APPLICATION_JSON)
                .content(body));
    }
}
