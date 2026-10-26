package de.serbroda.ragbag.collection;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.jayway.jsonpath.JsonPath;
import de.serbroda.ragbag.AbstractApiTest;
import java.util.List;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.ResultActions;

class CollectionExchangeTests extends AbstractApiTest {

    private Session owner;
    private String spaceId;
    private String root;

    @BeforeEach
    void setUp() throws Exception {
        owner = registerAndLogin();
        spaceId = firstSpaceId(owner);
        root = createCollection(owner, spaceId, "Homelab", null);
        createBookmark(owner, root);
        mockMvc.perform(post("/api/v1/collections/{id}/bookmarks", root)
                        .header("Authorization", bearer(owner.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"url\":\"http://127.0.0.1/newest\",\"title\":\"Newest\"}"))
                .andExpect(status().isOk());
        String child = createCollection(owner, spaceId, "Monitoring", root);
        createBookmark(owner, child);
    }

    @Test
    void export_isVersioned_andContainsTheTree() throws Exception {
        export(owner, root)
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.format").value("ragbag-collection"))
                .andExpect(jsonPath("$.format_version").value(1))
                .andExpect(jsonPath("$.app_version").isString())
                .andExpect(jsonPath("$.exported_at").isString())
                .andExpect(jsonPath("$.collection.name").value("Homelab"))
                .andExpect(jsonPath("$.collection.links[0].url").value("http://127.0.0.1/page"))
                .andExpect(jsonPath("$.collection.links[1].url").value("http://127.0.0.1/newest"))
                .andExpect(jsonPath("$.collection.links[0].tags[0]").value("a"))
                .andExpect(jsonPath("$.collection.collections[0].name").value("Monitoring"))
                .andExpect(
                        jsonPath("$.collection.collections[0].links.length()").value(1));
    }

    @Test
    void import_roundTrip_withPreviewAndDuplicates() throws Exception {
        String file = exportBody(owner, root);
        Session other = registerAndLogin();
        String otherSpace = firstSpaceId(other);
        String target = createCollection(other, otherSpace, "Imported", null);

        importFile(other, target, file, true)
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.preview").value(true))
                .andExpect(jsonPath("$.collectionsCreated").value(1))
                .andExpect(jsonPath("$.bookmarksImported").value(3));
        // The preview changed nothing
        mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", target)
                        .header("Authorization", bearer(other.accessToken())))
                .andExpect(jsonPath("$.length()").value(0));

        importFile(other, target, file, false)
                .andExpect(jsonPath("$.preview").value(false))
                .andExpect(jsonPath("$.bookmarksImported").value(3));
        mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", target)
                        .header("Authorization", bearer(other.accessToken())))
                .andExpect(jsonPath("$[0].title").value("Newest"))
                .andExpect(jsonPath("$[1].title").value("Example"));
        mockMvc.perform(get("/api/v1/spaces/{id}/collections", otherSpace)
                        .header("Authorization", bearer(other.accessToken())))
                .andExpect(jsonPath("$[0].children[0].name").value("Monitoring"));

        // Importing again merges the sub-collection and skips known links
        importFile(other, target, file, false)
                .andExpect(jsonPath("$.collectionsCreated").value(0))
                .andExpect(jsonPath("$.collectionsMerged").value(1))
                .andExpect(jsonPath("$.bookmarksImported").value(0))
                .andExpect(jsonPath("$.bookmarksSkipped").value(3));
    }

    @Test
    void import_rejectsForeignAndNewerFiles_andSkipsInvalidLinks() throws Exception {
        importFile(owner, root, "{\"format\":\"jucosuma\",\"format_version\":2}", false)
                .andExpect(status().isBadRequest());
        importFile(owner, root, "[]", false).andExpect(status().isBadRequest());
        importFile(owner, root, "{\"format\":\"ragbag-collection\",\"format_version\":99}", false)
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.detail").value(org.hamcrest.Matchers.containsString("newer version")));

        importFile(owner, root, """
						{"format":"ragbag-collection","format_version":1,"collection":{"name":"x","links":[
						{"url":"javascript:alert(1)"},{"url":"example.org","favicon":"javascript:alert(1)"}]}}
						""", false)
                .andExpect(jsonPath("$.bookmarksImported").value(1))
                .andExpect(jsonPath("$.bookmarksInvalid").value(1));
        String body = mockMvc.perform(get("/api/v1/collections/{id}/bookmarks", root)
                        .header("Authorization", bearer(owner.accessToken())))
                .andReturn()
                .getResponse()
                .getContentAsString();
        List<String> urls = JsonPath.read(body, "$[*].url");
        assertThat(urls).contains("https://example.org");
        List<Object> favicons = JsonPath.read(body, "$[?(@.url == 'https://example.org')].favicon");
        assertThat(favicons).containsOnlyNulls();
    }

    @Test
    void privateCollections_stayPrivate_andHiddenOnesAreNotExported() throws Exception {
        Session member = registerAndLogin();
        join(member, spaceId, "ADMIN", owner);
        String secret = createCollectionAs(owner, "Gifts for " + member.username(), root, true);
        createBookmark(owner, secret);

        // The admin cannot see the private collection, so it is not part of their export
        export(member, root)
                .andExpect(jsonPath("$.collection.collections.length()").value(1));
        String ownerExport = exportBody(owner, root);
        assertThat((List<Object>) JsonPath.read(ownerExport, "$.collection.collections[?(@.private == true)]"))
                .hasSize(1);

        Session other = registerAndLogin();
        String otherSpace = firstSpaceId(other);
        String target = createCollection(other, otherSpace, "Copy", null);
        importFile(other, target, ownerExport, false)
                .andExpect(jsonPath("$.collectionsCreated").value(2));
        mockMvc.perform(get("/api/v1/spaces/{id}/collections", otherSpace)
                        .header("Authorization", bearer(other.accessToken())))
                .andExpect(jsonPath("$[0].children[?(@.restricted == true)].name")
                        .value("Gifts for " + member.username()));
    }

    @Test
    void contributors_mayNeitherExportNorImport() throws Exception {
        Session contributor = registerAndLogin();
        join(contributor, spaceId, "CONTRIBUTOR", owner);
        export(contributor, root).andExpect(status().isForbidden());
        importFile(contributor, root, exportBody(owner, root), true).andExpect(status().isForbidden());
    }

    private String createCollectionAs(Session session, String name, String parentId, boolean restricted)
            throws Exception {
        String body = mockMvc.perform(post("/api/v1/spaces/{id}/collections", spaceId)
                        .header("Authorization", bearer(session.accessToken()))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"" + name + "\",\"parentId\":\"" + parentId + "\",\"restricted\":"
                                + restricted + "}"))
                .andExpect(status().isOk())
                .andReturn()
                .getResponse()
                .getContentAsString();
        return JsonPath.read(body, "$.id");
    }

    private ResultActions export(Session session, String collectionId) throws Exception {
        return mockMvc.perform(get("/api/v1/collections/{id}/export", collectionId)
                .header("Authorization", bearer(session.accessToken())));
    }

    private String exportBody(Session session, String collectionId) throws Exception {
        return export(session, collectionId).andReturn().getResponse().getContentAsString();
    }

    private ResultActions importFile(Session session, String collectionId, String file, boolean preview)
            throws Exception {
        return mockMvc.perform(post("/api/v1/collections/{id}/import", collectionId)
                .param("preview", String.valueOf(preview))
                .header("Authorization", bearer(session.accessToken()))
                .contentType(MediaType.APPLICATION_JSON)
                .content(file));
    }
}
