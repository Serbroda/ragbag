package de.serbroda.ragbag;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import jakarta.servlet.http.Cookie;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MvcResult;

class ApiIntegrationTests extends AbstractApiTest {

    private Session alice;
    private Session bob;

    @BeforeEach
    void setUp() throws Exception {
        alice = registerAndLogin();
        bob = registerAndLogin();
    }

    @Test
    void login_withWrongPassword_returns401() throws Exception {
        mockMvc.perform(post("/api/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"admin\",\"password\":\"wrong\"}"))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void refreshToken_cannotBeUsedAsBearerToken() throws Exception {
        mockMvc.perform(get("/api/v1/users/me")
                        .header("Authorization", bearer(alice.refreshCookie().getValue())))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void accessToken_cannotBeUsedToRefresh() throws Exception {
        mockMvc.perform(post("/api/auth/refresh").cookie(new Cookie("refresh_token", alice.accessToken())))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void refresh_withValidCookie_issuesAccessToken() throws Exception {
        mockMvc.perform(post("/api/auth/refresh").cookie(alice.refreshCookie()))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.access_token").isNotEmpty());
    }

    @Test
    void logoutAll_invalidatesRefreshToken() throws Exception {
        mockMvc.perform(get("/api/auth/logout/all").header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isNoContent());

        mockMvc.perform(post("/api/auth/refresh").cookie(alice.refreshCookie())).andExpect(status().isUnauthorized());
        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isUnauthorized());
    }

    @Test
    void bookmarkOfOtherUser_cannotBeReadUpdatedOrDeleted() throws Exception {
        String collectionId = createCollection(alice, firstSpaceId(alice), "Private", null);
        String bookmarkId = createBookmark(alice, collectionId);

        mockMvc.perform(get("/api/v1/bookmarks/{id}", bookmarkId).header("Authorization", bearer(bob.accessToken())))
                .andExpect(status().isForbidden());
        mockMvc.perform(put("/api/v1/bookmarks/{id}", bookmarkId)
                        .header("Authorization", bearer(bob.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"http://127.0.0.1/evil\",\"title\":\"pwned\"}"))
                .andExpect(status().isForbidden());
        mockMvc.perform(delete("/api/v1/bookmarks/{id}", bookmarkId).header("Authorization", bearer(bob.accessToken())))
                .andExpect(status().isForbidden());

        mockMvc.perform(get("/api/v1/bookmarks/{id}", bookmarkId).header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.title").value("Example"));
    }

    @Test
    void createCollection_respectsParentId() throws Exception {
        String spaceId = firstSpaceId(alice);
        String parentId = createCollection(alice, spaceId, "Parent", null);

        MvcResult result = mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(alice.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"Child\",\"parentId\":\"" + parentId + "\"}"))
                .andExpect(status().isOk())
                .andReturn();

        assertThat((String) JsonPath.read(result.getResponse().getContentAsString(), "$.parentId"))
                .isEqualTo(parentId);
    }

    @Test
    void moveCollection_intoOwnDescendant_isRejected() throws Exception {
        String spaceId = firstSpaceId(alice);
        String parentId = createCollection(alice, spaceId, "Parent", null);
        String childId = createCollection(alice, spaceId, "Child", parentId);

        mockMvc.perform(post("/api/v1/collections/{id}/move", parentId)
                        .header("Authorization", bearer(alice.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"parentId\":\"" + childId + "\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void moveCollection_intoForeignSpace_isRejected() throws Exception {
        String aliceCollection = createCollection(alice, firstSpaceId(alice), "Mine", null);
        String bobCollection = createCollection(bob, firstSpaceId(bob), "Bobs", null);

        mockMvc.perform(post("/api/v1/collections/{id}/move", aliceCollection)
                        .header("Authorization", bearer(alice.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"parentId\":\"" + bobCollection + "\"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void deleteCollection_withBookmarksAndChildren_succeeds() throws Exception {
        String spaceId = firstSpaceId(alice);
        String parentId = createCollection(alice, spaceId, "Parent", null);
        String childId = createCollection(alice, spaceId, "Child", parentId);
        createBookmark(alice, parentId);
        createBookmark(alice, childId);

        mockMvc.perform(delete("/api/v1/collections/{id}", parentId)
                        .header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isNoContent());
    }

    @Test
    void deleteSpace_withContent_succeeds() throws Exception {
        String spaceId = firstSpaceId(alice);
        createBookmark(alice, createCollection(alice, spaceId, "Parent", null));

        mockMvc.perform(delete("/api/v1/spaces/{id}", spaceId).header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isNoContent());
    }

    @Test
    void lastOwner_cannotLeaveSpace() throws Exception {
        mockMvc.perform(post("/api/v1/spaces/{id}/leave", firstSpaceId(alice))
                        .header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isConflict());
    }

    @Test
    void acceptingInvite_doesNotDowngradeOwner() throws Exception {
        String spaceId = firstSpaceId(alice);
        MvcResult invite = mockMvc.perform(post("/api/v1/invites")
                        .header("Authorization", bearer(alice.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"targetType\":\"SPACE\",\"targetId\":\"" + spaceId + "\",\"role\":\"VIEWER\"}"))
                .andExpect(status().isCreated())
                .andReturn();
        String token = JsonPath.read(invite.getResponse().getContentAsString(), "$.token");

        mockMvc.perform(post("/api/v1/invites/{token}", token).header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isOk());

        mockMvc.perform(delete("/api/v1/spaces/{id}", spaceId).header("Authorization", bearer(alice.accessToken())))
                .andExpect(status().isNoContent());
    }
}
