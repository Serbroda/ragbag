package de.serbroda.ragbag.config;

import static org.assertj.core.api.Assertions.assertThat;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.web.server.LocalServerPort;

/** Runs against the real Tomcat: forwarded headers are evaluated there, MockMvc would bypass it. */
@SpringBootTest(webEnvironment = SpringBootTest.WebEnvironment.RANDOM_PORT)
class ReverseProxyTests {

    private static final String LOGIN = "{\"username\":\"admin\",\"password\":\"test123\"}";

    @LocalServerPort
    private int port;

    private final HttpClient client = HttpClient.newHttpClient();

    @Test
    void login_behindHttpsReverseProxy_isSameOrigin_andSetsSecureCookie() throws Exception {
        // What nginx proxy manager sends for https://ragbag.example.com
        HttpResponse<String> response = login(HttpRequest.newBuilder(loginUri())
                .header("Origin", "https://ragbag.example.com")
                .header("X-Forwarded-Host", "ragbag.example.com")
                .header("X-Forwarded-Proto", "https")
                .header("X-Forwarded-For", "203.0.113.5"));

        assertThat(response.statusCode()).isEqualTo(200);
        assertThat(response.headers().firstValue("Set-Cookie"))
                .hasValueSatisfying(
                        cookie -> assertThat(cookie).contains("refresh_token=").contains("Secure"));
    }

    @Test
    void login_overPlainHttp_keepsTheCookieUsable() throws Exception {
        HttpResponse<String> response =
                login(HttpRequest.newBuilder(loginUri()).header("Origin", "http://localhost:" + port));

        assertThat(response.statusCode()).isEqualTo(200);
        assertThat(response.headers().firstValue("Set-Cookie"))
                .hasValueSatisfying(cookie -> assertThat(cookie).doesNotContain("Secure"));
    }

    @Test
    void foreignOrigins_areStillRejected() throws Exception {
        HttpResponse<String> response =
                login(HttpRequest.newBuilder(loginUri()).header("Origin", "https://evil.example"));

        assertThat(response.statusCode()).isEqualTo(403);
    }

    private URI loginUri() {
        return URI.create("http://localhost:" + port + "/api/auth/login");
    }

    private HttpResponse<String> login(HttpRequest.Builder request) throws Exception {
        return client.send(
                request.header("Content-Type", "application/json")
                        .POST(HttpRequest.BodyPublishers.ofString(LOGIN))
                        .build(),
                HttpResponse.BodyHandlers.ofString());
    }
}
