package de.serbroda.ragbag.image;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.multipart;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import java.awt.image.BufferedImage;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import javax.imageio.ImageIO;
import org.junit.jupiter.api.Test;
import org.springframework.http.HttpMethod;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.web.servlet.ResultActions;

class ImageApiTests extends AbstractApiTest {

    @Test
    void avatar_isCroppedScaledAndServedPublicly() throws Exception {
        Session user = registerAndLogin();

        String body = uploadAvatar(user, png(800, 400))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        String imageId = JsonPath.read(body, "$.avatarId");

        byte[] served = mockMvc.perform(get("/api/v1/images/{id}", imageId))
                .andExpect(status().isOk())
                .andExpect(header().string("Content-Type", "image/png"))
                .andExpect(header().string("Cache-Control", org.hamcrest.Matchers.containsString("immutable")))
                .andReturn()
                .getResponse()
                .getContentAsByteArray();
        BufferedImage image = ImageIO.read(new ByteArrayInputStream(served));
        assertThat(image.getWidth()).isEqualTo(ImageService.SIZE);
        assertThat(image.getHeight()).isEqualTo(ImageService.SIZE);

        mockMvc.perform(get("/api/v1/users/me").header("Authorization", bearer(user.accessToken())))
                .andExpect(jsonPath("$.avatarId").value(imageId));
    }

    @Test
    void replacingOrRemovingAvatar_deletesTheOldImage() throws Exception {
        Session user = registerAndLogin();
        String first = JsonPath.read(
                uploadAvatar(user, png(64, 64)).andReturn().getResponse().getContentAsString(), "$.avatarId");
        String second = JsonPath.read(
                uploadAvatar(user, png(32, 32)).andReturn().getResponse().getContentAsString(), "$.avatarId");

        assertThat(second).isNotEqualTo(first);
        mockMvc.perform(get("/api/v1/images/{id}", first)).andExpect(status().isNotFound());

        mockMvc.perform(delete("/api/v1/users/me/avatar").header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.avatarId").doesNotExist());
        mockMvc.perform(get("/api/v1/images/{id}", second)).andExpect(status().isNotFound());
    }

    @Test
    void nonImages_areRejected() throws Exception {
        Session user = registerAndLogin();
        uploadAvatar(user, "<svg onload=alert(1)></svg>".getBytes()).andExpect(status().isBadRequest());
        // The URL variant goes through the SSRF guard
        mockMvc.perform(multipart(HttpMethod.PUT, "/api/v1/users/me/avatar")
                        .param("url", "http://127.0.0.1/avatar.png")
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isBadRequest());
        mockMvc.perform(multipart(HttpMethod.PUT, "/api/v1/users/me/avatar")
                        .header("Authorization", bearer(user.accessToken())))
                .andExpect(status().isBadRequest());
    }

    @Test
    void spaceIcon_needsEditPermission_andIsVisibleToMembers() throws Exception {
        Session owner = registerAndLogin();
        Session viewer = registerAndLogin();
        String spaceId = firstSpaceId(owner);
        join(viewer, spaceId, "VIEWER", owner);

        mockMvc.perform(multipart(HttpMethod.PUT, "/api/v1/spaces/{id}/icon", spaceId)
                        .file(new MockMultipartFile("file", "icon.png", "image/png", png(16, 16)))
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isForbidden());

        String body = mockMvc.perform(multipart(HttpMethod.PUT, "/api/v1/spaces/{id}/icon", spaceId)
                        .file(new MockMultipartFile("file", "icon.png", "image/png", png(16, 16)))
                        .header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        String iconId = JsonPath.read(body, "$.iconId");

        mockMvc.perform(get("/api/v1/spaces").header("Authorization", bearer(viewer.accessToken())))
                .andExpect(jsonPath("$[?(@.id == '" + spaceId + "')].iconId").value(iconId));
        mockMvc.perform(get("/api/v1/spaces/{id}/members", spaceId)
                        .header("Authorization", bearer(viewer.accessToken())))
                .andExpect(status().isOk());

        mockMvc.perform(delete("/api/v1/spaces/{id}/icon", spaceId)
                        .header("Authorization", bearer(owner.accessToken())))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.iconId").doesNotExist());
    }

    private ResultActions uploadAvatar(Session session, byte[] content) throws Exception {
        return mockMvc.perform(multipart(HttpMethod.PUT, "/api/v1/users/me/avatar")
                .file(new MockMultipartFile("file", "avatar.png", "image/png", content))
                .header("Authorization", bearer(session.accessToken())));
    }

    private static byte[] png(int width, int height) throws Exception {
        BufferedImage image = new BufferedImage(width, height, BufferedImage.TYPE_INT_RGB);
        ByteArrayOutputStream output = new ByteArrayOutputStream();
        ImageIO.write(image, "png", output);
        return output.toByteArray();
    }
}
