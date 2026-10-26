package de.serbroda.ragbag.bookmark;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import de.serbroda.ragbag.AbstractApiTest;
import org.hamcrest.Matchers;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.http.MediaType;

class BookmarkApiTests extends AbstractApiTest {

    private Session user;
    private String spaceId;
    private String collectionId;

    @BeforeEach
    void setUp() throws Exception {
        user = registerAndLogin();
        spaceId = firstSpaceId(user);
        collectionId = createCollection(user, spaceId, "Links", null);
    }

    @ParameterizedTest
    @ValueSource(strings = {"javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "ftp://example.com"})
    void createBookmark_withNonHttpUrl_isRejected(String url) throws Exception {
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"" + url + "\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void updateBookmark_toJavascriptUrl_isRejected() throws Exception {
        String bookmarkId = createBookmark(user, collectionId);
        mockMvc.perform(put("/api/v1/bookmarks/{id}", bookmarkId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"javascript:alert(1)\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void createBookmark_withoutScheme_defaultsToHttps() throws Exception {
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"127.0.0.1/docs\",\"title\":\"Docs\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.url").value("https://127.0.0.1/docs"))
                .andExpect(jsonPath("$.createdBy").value(user.username()))
                .andExpect(jsonPath("$.createdAt").isNotEmpty());
    }

    @Test
    void recentBookmarks_areNewestFirst_andLimited() throws Exception {
        String other = createCollection(user, spaceId, "Other", null);
        createBookmark(user, collectionId);
        createBookmark(user, other);
        String newest = createBookmark(user, collectionId);

        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks", spaceId)
                        .param("limit", "2")
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(2))
                .andExpect(jsonPath("$[0].id").value(newest));
    }

    @Test
    void search_treatsWildcardsLiterally() throws Exception {
        createBookmark(user, collectionId);
        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks/search", spaceId)
                        .param("q", "%")
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(0));
        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks/search", spaceId)
                        .param("q", "examp")
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(jsonPath("$.length()").value(1));
    }

    @Test
    void getCollection_includesParentId() throws Exception {
        String child = createCollection(user, spaceId, "Child", collectionId);
        mockMvc.perform(get("/api/v1/collections/{id}", child).header("Authorization", bearer(user.accessToken())))
                .andExpect(jsonPath("$.parentId").value(collectionId));
    }

    @Test
    void createCollection_withBlankName_isRejected() throws Exception {
        mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"  \"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void fetchMetadata_normalizesUrl_withoutSaving() throws Exception {
        // Loopback is refused by the SSRF guard, so this only checks normalization and the empty result
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks/metadata", collectionId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\" 127.0.0.1/page \"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.url").value("https://127.0.0.1/page"))
                .andExpect(jsonPath("$.found").value(false));
        mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
    }

    @Test
    void fetchMetadata_rejectsJavascriptUrl() throws Exception {
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks/metadata", collectionId)
                        .header("Authorization", bearer(user.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"javascript:alert(1)\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void fetchMetadata_requiresWriteAccess() throws Exception {
        Session viewer = registerAndLogin();
        join(viewer, spaceId, "VIEWER", user);
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks/metadata", collectionId)
                        .header("Authorization", bearer(viewer.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"https://example.com\"}"))
                .andExpect(status().isForbidden());
    }

    @Test
    void logout_clearsRefreshCookie_withoutToken() throws Exception {
        mockMvc.perform(post("/api/auth/logout"))
                .andExpect(status().isNoContent())
                .andExpect(header().string("Set-Cookie", Matchers.containsString("Max-Age=0")));
    }
}
