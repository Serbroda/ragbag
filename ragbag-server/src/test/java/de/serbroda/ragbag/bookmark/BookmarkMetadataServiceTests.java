package de.serbroda.ragbag.bookmark;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

class BookmarkMetadataServiceTests {

    private final BookmarkMetadataService service = new BookmarkMetadataService();

    @ParameterizedTest
    @ValueSource(
            strings = {
                "http://localhost:8080/actuator/health",
                "http://127.0.0.1/",
                "http://10.0.0.1/",
                "http://192.168.1.1/",
                "http://169.254.169.254/latest/meta-data/",
                "http://[::1]/",
                "http://[fd00::1]/",
                "file:///etc/passwd"
            })
    void fetch_refusesNonPublicTargets(String url) {
        assertThat(service.fetch(url)).isEqualTo(BookmarkMetadataService.BookmarkMetadata.empty());
    }
}
