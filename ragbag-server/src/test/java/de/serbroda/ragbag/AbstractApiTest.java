package de.serbroda.ragbag;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import jakarta.servlet.http.Cookie;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

/** Shared helpers for end-to-end API tests against the in-memory H2 database. */
@SpringBootTest
@AutoConfigureMockMvc
public abstract class AbstractApiTest {

    @Autowired
    protected MockMvc mockMvc;

    public record Session(String userId, String username, String accessToken, Cookie refreshCookie) {}

    protected String createInvite(Session session, String spaceId, String role) throws Exception {
        String body = mockMvc.perform(post("/api/v1/invites")
                        .header("Authorization", bearer(session.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"targetType\":\"SPACE\",\"targetId\":\"" + spaceId + "\",\"role\":\"" + role
                                + "\"}"))
                .andExpect(status().isCreated())
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$.token");
    }

    protected void join(Session session, String spaceId, String role, Session owner) throws Exception {
        String token = createInvite(owner, spaceId, role);
        mockMvc.perform(post("/api/v1/invites/{token}", token).header("Authorization", bearer(session.accessToken())))
                .andExpect(status().isOk());
    }

    protected Session registerAndLogin() throws Exception {
        String username = "u" + UUID.randomUUID().toString().substring(0, 8);
        mockMvc.perform(post("/api/auth/register")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"" + username + "\",\"email\":\"" + username
                                + "@example.com\",\"password\":\"secret123\"}"))
                .andExpect(status().isOk());

        MvcResult login = mockMvc.perform(post("/api/auth/login")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"username\":\"" + username + "\",\"password\":\"secret123\"}"))
                .andExpect(status().isOk())
                .andReturn();

        String accessToken = JsonPath.read(login.getResponse().getContentAsString(), "$.access_token");
        Cookie refreshCookie = login.getResponse().getCookie("refresh_token");
        assertThat(refreshCookie).isNotNull();
        String userId = JsonPath.read(
                mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(accessToken)))
                        .andExpect(status().isOk())
                        .andReturn()
                        .getResponse()
                        .getContentAsString(),
                "$.id");
        return new Session(userId, username, accessToken, new Cookie("refresh_token", refreshCookie.getValue()));
    }

    protected String firstSpaceId(Session session) throws Exception {
        String body = mockMvc.perform(get("/api/v1/spaces").header("Authorization", bearer(session.accessToken())))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$[0].id");
    }

    protected String createCollection(Session session, String spaceId, String name, String parentId) throws Exception {
        String parent = parentId != null ? ",\"parentId\":\"" + parentId + "\"" : "";
        String body = mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(session.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"" + name + "\"" + parent + "}"))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$.id");
    }

    protected String createBookmark(Session session, String collectionId) throws Exception {
        // Loopback URL: the metadata fetcher must refuse it, so no network access happens in tests
        String body = mockMvc.perform(post("/api/v1/collections/{id}/bookmarks", collectionId)
                        .header("Authorization", bearer(session.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"http://127.0.0.1/page\",\"title\":\"Example\",\"tags\":[\"a\"]}"))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$.id");
    }

    protected static String bearer(String token) {
        return "Bearer " + token;
    }
}
