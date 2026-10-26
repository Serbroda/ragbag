package de.serbroda.ragbag.auth;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import org.hamcrest.Matchers;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

class PasswordTests extends AbstractApiTest {

    @Test
    void changePassword_signsOutOtherSessions_andKeepsThisOne() throws Exception {
        Session session = registerAndLogin();

        String body = changePassword(session, "secret123", "a-new-password")
                .andExpect(status().isOk())
                .andExpect(header().string("Set-Cookie", Matchers.containsString("refresh_token=")))
                .andReturn()
                .getResponse()
                .getContentAsString();
        String newToken = JsonPath.read(body, "$.access_token");

        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(newToken)))
                .andExpect(status().isOk());
        // Tokens issued before the change (other devices) are invalid
        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(session.accessToken())))
                .andExpect(status().isUnauthorized());
        mockMvc.perform(post("/api/auth/refresh").cookie(session.refreshCookie()))
                .andExpect(status().isUnauthorized());

        login(session.username(), "secret123").andExpect(status().isUnauthorized());
        login(session.username(), "a-new-password").andExpect(status().isOk());
    }

    @Test
    void changePassword_requiresTheCurrentPassword_andAValidNewOne() throws Exception {
        Session session = registerAndLogin();

        // 400, not 401: the client would treat a 401 as an expired session
        changePassword(session, "wrong-password", "a-new-password").andExpect(status().isBadRequest());
        changePassword(session, "secret123", "short").andExpect(status().isBadRequest());
        changePassword(session, "secret123", "x".repeat(73)).andExpect(status().isBadRequest());

        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(session.accessToken())))
                .andExpect(status().isOk());
    }

    @Test
    void register_rejectsShortPasswords() throws Exception {
        mockMvc.perform(post("/api/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"shorty\",\"email\":\"shorty@example.com\",\"password\":\"1234\"}"))
                .andExpect(status().isBadRequest());
    }

    private ResultActions changePassword(Session session, String current, String next) throws Exception {
        return mockMvc.perform(post("/api/auth/password")
                .header("Authorization", bearer(session.accessToken()))
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"currentPassword\":\"" + current + "\",\"newPassword\":\"" + next + "\"}"));
    }

    private ResultActions login(String username, String password) throws Exception {
        return mockMvc.perform(post("/api/auth/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content("{\"username\":\"" + username + "\",\"password\":\"" + password + "\"}"));
    }
}
