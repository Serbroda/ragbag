package de.serbroda.ragbag.collection;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

class PrivateCollectionTests extends AbstractApiTest {

    private Session owner;
    private Session giftee;
    private Session friend;
    private String spaceId;

    @BeforeEach
    void setUp() throws Exception {
        owner = registerAndLogin();
        giftee = registerAndLogin();
        friend = registerAndLogin();
        spaceId = firstSpaceId(owner);
        // The person the gifts are for is even an admin of the space
        join(giftee, spaceId, "ADMIN", owner);
        join(friend, spaceId, "CONTRIBUTOR", owner);
    }

    @Test
    void privateCollection_isInvisibleToEveryoneElse_includingAdmins() throws Exception {
        String gifts = createPrivate("Gift ideas", null, friend.userId());
        String sub = createCollection(friend, spaceId, "Books", gifts);
        String bookmark = createBookmark(friend, sub);

        tree(owner)
                .andExpect(jsonPath("$[0].name").value("Gift ideas"))
                .andExpect(jsonPath("$[0].restricted").value(true));
        tree(friend).andExpect(jsonPath("$[0].children[0].name").value("Books"));
        tree(giftee).andExpect(jsonPath("$.length()").value(0));

        // Direct access looks like the collection does not exist
        mockMvc.perform(get("/api/v1/collections/{id}", sub).header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNotFound());
        mockMvc.perform(get("/api/v1/bookmarks/{id}", bookmark).header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNotFound());
        mockMvc.perform(put("/api/v1/bookmarks/{id}/favorite", bookmark)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNotFound());
        mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(giftee.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"Peek\",\"parentId\":\"" + gifts + "\"}"))
                .andExpect(status().isNotFound());

        // ... and it never shows up in lists
        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks", spaceId)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks/search", spaceId)
                        .param("q", "Example")
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));
        mockMvc.perform(get("/api/v1/spaces/{id}/dashboard", spaceId)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(jsonPath("$.sections[1].bookmarks.length()").value(0));
        mockMvc.perform(put("/api/v1/spaces/{id}/dashboard", spaceId)
                        .header("Authorization", bearer(giftee.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"sections\":[{\"title\":\"x\",\"type\":\"LINKS\",\"bookmarkIds\":[\"" + bookmark
                                + "\"]}]}"))
                .andExpect(status().isBadRequest());

        mockMvc.perform(get("/api/v1/spaces/{id}/bookmarks/search", spaceId)
                        .param("q", "Example")
                        .header("Authorization", bearer(friend.accessToken())))
                .andExpect(jsonPath("$.length()").value(1));
    }

    @Test
    void favoritesAndPins_disappearWhenTheCollectionTurnsPrivate() throws Exception {
        String shared = createCollection(owner, spaceId, "Shared", null);
        String bookmark = createBookmark(owner, shared);
        mockMvc.perform(put("/api/v1/bookmarks/{id}/favorite", bookmark)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNoContent());

        updateAccess(owner, shared, "{\"restricted\":true,\"memberIds\":[]}")
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.members.length()").value(1))
                .andExpect(jsonPath("$.members[0].userId").value(owner.userId()));

        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));

        updateAccess(owner, shared, "{\"restricted\":false}")
                .andExpect(jsonPath("$.members.length()").value(0));
        mockMvc.perform(get("/api/v1/spaces/{id}/favorites", spaceId)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(jsonPath("$.length()").value(1));
    }

    @Test
    void onlyCreatorOrAdmins_mayMakeASharedCollectionPrivate() throws Exception {
        String shared = createCollection(owner, spaceId, "Team links", null);

        updateAccess(friend, shared, "{\"restricted\":true}").andExpect(status().isForbidden());
        mockMvc.perform(get("/api/v1/collections/{id}/access", shared)
                        .header("Authorization", bearer(friend.accessToken())))
                .andExpect(jsonPath("$.canManage").value(false));

        // Within a private collection, everyone who can see it may manage it
        String gifts = createPrivate("Gift ideas", null, friend.userId());
        updateAccess(friend, gifts, "{\"restricted\":true,\"memberIds\":[\"" + owner.userId() + "\"]}")
                .andExpect(status().isOk());
        updateAccess(
                        friend,
                        gifts,
                        "{\"restricted\":true,\"memberIds\":[\""
                                + registerAndLogin().userId() + "\"]}")
                .andExpect(status().isBadRequest());
    }

    @Test
    void deletingAParent_neverTakesHiddenCollectionsWithIt() throws Exception {
        String family = createCollection(owner, spaceId, "Family", null);
        createPrivate("Gift ideas", family, friend.userId());

        mockMvc.perform(delete("/api/v1/collections/{id}", family)
                        .header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isConflict());
        mockMvc.perform(delete("/api/v1/collections/{id}", family).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());
    }

    @Test
    void leavingTheSpace_removesAccess_andDeletesCollectionsNobodyCanSee() throws Exception {
        String mine = createPrivate("Only mine", null);
        String ours = createPrivate("Ours", null, friend.userId());

        mockMvc.perform(post("/api/v1/spaces/{id}/leave", spaceId)
                        .header("Authorization", bearer(friend.accessToken())))
                .andExpect(status().isNoContent());
        join(friend, spaceId, "CONTRIBUTOR", owner);
        tree(friend).andExpect(jsonPath("$.length()").value(0));

        // Hand the space over and leave: nobody is left who could see "Only mine"
        mockMvc.perform(put("/api/v1/spaces/{id}/members/{userId}", spaceId, giftee.userId())
                        .header("Authorization", bearer(owner.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"role\":\"OWNER\"}"))
                .andExpect(status().isOk());
        mockMvc.perform(post("/api/v1/spaces/{id}/leave", spaceId).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());
        join(owner, spaceId, "ADMIN", giftee);

        tree(owner).andExpect(jsonPath("$.length()").value(0));
        mockMvc.perform(get("/api/v1/collections/{id}", mine).header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNotFound());
        mockMvc.perform(get("/api/v1/collections/{id}", ours).header("Authorization", bearer(giftee.accessToken())))
                .andExpect(status().isNotFound());
    }

    private String createPrivate(String name, String parentId, String... memberIds) throws Exception {
        String parent = parentId != null ? ",\"parentId\":\"" + parentId + "\"" : "";
        String members = String.join("\",\"", memberIds);
        String body = mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(owner.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"" + name + "\"" + parent + ",\"restricted\":true,\"memberIds\":["
                                + (memberIds.length > 0 ? "\"" + members + "\"" : "") + "]}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.restricted").value(true))
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$.id");
    }

    private ResultActions tree(Session session) throws Exception {
        return mockMvc.perform(get("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(session.accessToken())))
                .andExpect(status().isOk());
    }

    private ResultActions updateAccess(Session session, String collectionId, String body) throws Exception {
        return mockMvc.perform(put("/api/v1/collections/{id}/access", collectionId)
                .header("Authorization", bearer(session.accessToken()))
                .contentType(MediaType.APPLICATION_JSON)
                .content(body));
    }
}
