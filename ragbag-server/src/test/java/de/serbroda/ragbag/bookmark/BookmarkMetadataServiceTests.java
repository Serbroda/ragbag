package de.serbroda.ragbag.bookmark;

import static org.assertj.core.api.Assertions.assertThat;

import java.net.URI;
import org.jsoup.Jsoup;
import org.junit.jupiter.api.Test;
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

    @ParameterizedTest
    @ValueSource(
            strings = {
                "http://localhost:8080/actuator/health",
                "http://127.0.0.1/",
                "http://169.254.169.254/latest/meta-data/",
                "http://[::1]/"
            })
    void fetch_withPrivateHostsAllowed_stillRefusesLoopbackAndLinkLocal(String url) {
        assertThat(new BookmarkMetadataService(true).fetch(url))
                .isEqualTo(BookmarkMetadataService.BookmarkMetadata.empty());
    }

    @Test
    void parse_prefersOpenGraph_andResolvesRelativeUrls() {
        String html = """
				<html><head>
				<title>Svelte docs | Svelte</title>
				<meta property="og:title" content="Svelte docs">
				<meta property="og:description" content="Learn Svelte">
				<meta property="og:image" content="/og.png">
				<meta name="keywords" content="Svelte, JavaScript , svelte, ,Compiler">
				<link rel="icon" href="favicon.svg">
				</head></html>
				""";

        BookmarkMetadataService.BookmarkMetadata metadata =
                BookmarkMetadataService.parse(Jsoup.parse(html), URI.create("https://svelte.dev/docs/intro"));

        assertThat(metadata.title()).isEqualTo("Svelte docs");
        assertThat(metadata.description()).isEqualTo("Learn Svelte");
        assertThat(metadata.ogImage()).isEqualTo("https://svelte.dev/og.png");
        assertThat(metadata.favicon()).isEqualTo("https://svelte.dev/docs/favicon.svg");
        assertThat(metadata.keywords()).containsExactly("svelte", "javascript", "compiler");
    }

    @Test
    void parse_fallsBackToTitleTag_andDefaultFavicon() {
        BookmarkMetadataService.BookmarkMetadata metadata = BookmarkMetadataService.parse(
                Jsoup.parse("<html><head><title> Plain </title><meta name=description content=Desc></head></html>"),
                URI.create("https://example.com/a/b"));

        assertThat(metadata.title()).isEqualTo("Plain");
        assertThat(metadata.description()).isEqualTo("Desc");
        assertThat(metadata.favicon()).isEqualTo("https://example.com/favicon.ico");
        assertThat(metadata.keywords()).isEmpty();
    }
}
