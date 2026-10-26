package de.serbroda.ragbag.auth;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

/** The dev demo user "admin" is the configured instance admin (app.admin-users). */
class PasswordResetTests extends AbstractApiTest {

    private String adminToken;

    @BeforeEach
    void loginAdmin() throws Exception {
        String body = login("admin", "test123").andReturn().getResponse().getContentAsString();
        adminToken = JsonPath.read(body, "$.access_token");
    }

    @Test
    void resetLink_setsANewPassword_once_andSignsOutEverywhere() throws Exception {
        Session user = registerAndLogin();
        String token = createReset(adminToken, user.userId())
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.expiresAt").isString())
                .andReturn()
                .getResponse()
                .getContentAsString();
        token = JsonPath.read(token, "$.token");

        mockMvc.perform(get("/api/auth/password-reset/{token}", token))
                .andExpect(jsonPath("$.valid").value(true))
                .andExpect(jsonPath("$.username").value(user.username()));
        reset(token, "short").andExpect(status().isBadRequest());
        reset(token, "reset-password-1").andExpect(status().isNoContent());

        login(user.username(), "reset-password-1").andExpect(status().isOk());
        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isUnauthorized());
        // Used up
        mockMvc.perform(get("/api/auth/password-reset/{token}", token))
                .andExpect(jsonPath("$.valid").value(false));
        reset(token, "reset-password-2").andExpect(status().isBadRequest());
    }

    @Test
    void newLink_replacesTheOldOne() throws Exception {
        Session user = registerAndLogin();
        String first = JsonPath.read(
                createReset(adminToken, user.userId()).andReturn().getResponse().getContentAsString(), "$.token");
        String second = JsonPath.read(
                createReset(adminToken, user.userId()).andReturn().getResponse().getContentAsString(), "$.token");

        reset(first, "reset-password-1").andExpect(status().isBadRequest());
        reset(second, "reset-password-1").andExpect(status().isNoContent());
    }

    @Test
    void onlyInstanceAdmins_manageUsers() throws Exception {
        Session user = registerAndLogin();
        Session other = registerAndLogin();

        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(user.accessToken())))
                .andExpect(jsonPath("$.instanceAdmin").value(false));
        mockMvc.perform(get("/api/v1/admin/users").header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isForbidden());
        createReset(user.accessToken(), other.userId()).andExpect(status().isForbidden());

        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(adminToken)))
                .andExpect(jsonPath("$.instanceAdmin").value(true));
        mockMvc.perform(get("/api/v1/admin/users").header("Authorization", bearer(adminToken)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[?(@.username == '" + user.username() + "')].spaces")
                        .value(1));
    }

    @Test
    void unknownLinks_areInvalid() throws Exception {
        mockMvc.perform(get("/api/auth/password-reset/{token}", "not-a-token"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.valid").value(false));
        reset("not-a-token", "reset-password-1").andExpect(status().isBadRequest());
    }

    private ResultActions createReset(String token, String userId) throws Exception {
        return mockMvc.perform(
                post("/api/v1/admin/users/{id}/password-reset", userId).header("Authorization", bearer(token)));
    }

    private ResultActions reset(String token, String password) throws Exception {
        return mockMvc.perform(post("/api/auth/password-reset/{token}", token)
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"newPassword\":\"" + password + "\"}"));
    }

    private ResultActions login(String username, String password) throws Exception {
        return mockMvc.perform(post("/api/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"username\":\"" + username + "\",\"password\":\"" + password + "\"}"));
    }
}
