package de.serbroda.ragbag.space;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import org.hamcrest.Matchers;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

class SpaceManagementTests extends AbstractApiTest {

    private Session owner;
    private Session admin;
    private Session contributor;
    private Session viewer;
    private String spaceId;

    @BeforeEach
    void setUp() throws Exception {
        owner = registerAndLogin();
        admin = registerAndLogin();
        contributor = registerAndLogin();
        viewer = registerAndLogin();
        spaceId = firstSpaceId(owner);
        join(admin, spaceId, "ADMIN", owner);
        join(contributor, spaceId, "CONTRIBUTOR", owner);
        join(viewer, spaceId, "VIEWER", owner);
    }

    @Test
    void createSpace_makesCreatorOwner() throws Exception {
        mockMvc.perform(post("/api/v1/spaces")
                        .header("Authorization", bearer(viewer.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"  Reading list \",\"description\":\"Stuff\"}"))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.name").value("Reading list"))
                .andExpect(jsonPath("$.role").value("OWNER"))
                .andExpect(jsonPath("$.permissions", Matchers.hasItems("DELETE_SPACE", "MANAGE_MEMBERS")));

        mockMvc.perform(get("/api/v1/spaces").header("Authorization", bearer(viewer.accessToken())))
                .andExpect(jsonPath("$.length()").value(3));
    }

    @Test
    void createSpace_withBlankName_isRejected() throws Exception {
        mockMvc.perform(post("/api/v1/spaces")
                        .header("Authorization", bearer(owner.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"   \"}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void members_areListedForEveryMember_sortedByRole() throws Exception {
        mockMvc.perform(get("/api/v1/spaces/{id}/members", spaceId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.length()").value(4))
                .andExpect(jsonPath("$[0].username").value(owner.username()))
                .andExpect(jsonPath("$[0].role").value("OWNER"))
                .andExpect(jsonPath("$[3].role").value("VIEWER"));
    }

    @Test
    void members_areHiddenFromNonMembers() throws Exception {
        Session stranger = registerAndLogin();
        mockMvc.perform(get("/api/v1/spaces/{id}/members", spaceId)
                        .header("Authorization", bearer(stranger.accessToken())))
                .andExpect(status().isForbidden());
    }

    @Test
    void contributor_cannotEditSpaceOrManageMembers_butCanCreateCollections() throws Exception {
        mockMvc.perform(put("/api/v1/spaces/{id}", spaceId)
                        .header("Authorization", bearer(contributor.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"Hijacked\"}"))
                .andExpect(status().isForbidden());
        changeRole(contributor, viewer, "CONTRIBUTOR").andExpect(status().isForbidden());
        createCollection(contributor, spaceId, "Allowed", null);
    }

    @Test
    void admin_canEditSpace_butNotDeleteIt() throws Exception {
        mockMvc.perform(put("/api/v1/spaces/{id}", spaceId)
                        .header("Authorization", bearer(admin.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"Renamed\",\"description\":\"New\"}"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.name").value("Renamed"))
                .andExpect(jsonPath("$.role").value("ADMIN"));
        mockMvc.perform(delete("/api/v1/spaces/{id}", spaceId).header("Authorization", bearer(admin.accessToken())))
                .andExpect(status().isForbidden());
    }

    @Test
    void admin_canManageLowerRoles_only() throws Exception {
        changeRole(admin, viewer, "CONTRIBUTOR")
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.role").value("CONTRIBUTOR"));
        changeRole(admin, viewer, "OWNER").andExpect(status().isForbidden());
        changeRole(admin, owner, "VIEWER").andExpect(status().isForbidden());

        mockMvc.perform(delete("/api/v1/spaces/{id}/members/{userId}", spaceId, contributor.userId())
                        .header("Authorization", bearer(admin.accessToken())))
                .andExpect(status().isNoContent());
        mockMvc.perform(get("/api/v1/spaces/{id}", spaceId).header("Authorization", bearer(contributor.accessToken())))
                .andExpect(status().isForbidden());
    }

    @Test
    void owner_canPromoteAnotherOwner_whoCanThenDemoteTheFirst() throws Exception {
        changeRole(owner, admin, "OWNER").andExpect(status().isOk());
        changeRole(admin, owner, "VIEWER").andExpect(status().isOk());
        mockMvc.perform(post("/api/v1/spaces/{id}/leave", spaceId).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());
    }

    @Test
    void ownChangingOwnRole_isRejected() throws Exception {
        changeRole(owner, owner, "VIEWER").andExpect(status().isBadRequest());
    }

    @Test
    void invites_respectRoleCeiling() throws Exception {
        mockMvc.perform(post("/api/v1/invites")
                        .header("Authorization", bearer(contributor.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"targetType\":\"SPACE\",\"targetId\":\"" + spaceId + "\",\"role\":\"VIEWER\"}"))
                .andExpect(status().isForbidden());
        createInvite(admin, spaceId, "ADMIN");
    }

    @Test
    void spaceInvites_listPreviewAndRevoke() throws Exception {
        String token = createInvite(owner, spaceId, "CONTRIBUTOR");

        mockMvc.perform(get("/api/v1/spaces/{id}/invites", spaceId)
                        .header("Authorization", bearer(admin.accessToken())))
                .andExpect(status().isOk())
                .andExpect(
                        jsonPath("$[?(@.token == '" + token + "')].createdBy").value(owner.username()));
        mockMvc.perform(get("/api/v1/spaces/{id}/invites", spaceId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isForbidden());

        Session newcomer = registerAndLogin();
        mockMvc.perform(get("/api/v1/invites/{token}", token).header("Authorization", bearer(newcomer.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.targetId").value(spaceId))
                .andExpect(jsonPath("$.role").value("CONTRIBUTOR"))
                .andExpect(jsonPath("$.invitedBy").value(owner.username()))
                .andExpect(jsonPath("$.valid").value(true))
                .andExpect(jsonPath("$.alreadyMember").value(false));

        // Admins may revoke invites created by others
        mockMvc.perform(delete("/api/v1/invites/{token}", token).header("Authorization", bearer(admin.accessToken())))
                .andExpect(status().isNoContent());
        mockMvc.perform(post("/api/v1/invites/{token}", token).header("Authorization", bearer(newcomer.accessToken())))
                .andExpect(status().isNotFound());
    }

    @Test
    void invite_withMaxUses_isUsedUp() throws Exception {
        String body = mockMvc.perform(post("/api/v1/invites")
                        .header("Authorization", bearer(owner.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"targetType\":\"SPACE\",\"targetId\":\"" + spaceId
                                + "\",\"role\":\"VIEWER\",\"maxUses\":1}"))
                .andExpect(status().isCreated())
                .andReturn()
                .getResponse()
                .getContentAsString();
        String token = JsonPath.read(body, "$.token");

        Session first = registerAndLogin();
        Session second = registerAndLogin();
        mockMvc.perform(post("/api/v1/invites/{token}", token).header("Authorization", bearer(first.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.alreadyMember").value(true));
        mockMvc.perform(post("/api/v1/invites/{token}", token).header("Authorization", bearer(second.accessToken())))
                .andExpect(status().isConflict());
    }

    @Test
    void deleteSpace_removesItsInvites() throws Exception {
        String token = createInvite(owner, spaceId, "VIEWER");
        mockMvc.perform(delete("/api/v1/spaces/{id}", spaceId).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNoContent());
        mockMvc.perform(get("/api/v1/invites/{token}", token).header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isNotFound());
    }

    private ResultActions changeRole(Session actor, Session target, String role) throws Exception {
        return mockMvc.perform(put("/api/v1/spaces/{id}/members/{userId}", spaceId, target.userId())
                .header("Authorization", bearer(actor.accessToken()))
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"role\":\"" + role + "\"}"));
    }
}
