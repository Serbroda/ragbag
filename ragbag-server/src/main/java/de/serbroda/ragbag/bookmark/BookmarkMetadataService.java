package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.shared.http.SafeHttpClient;
import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.net.URI;
import java.util.Arrays;
import java.util.List;
import java.util.Locale;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class BookmarkMetadataService {

    private static final Logger LOGGER = LoggerFactory.getLogger(BookmarkMetadataService.class);

    private static final int MAX_KEYWORDS = 8;
    private static final int MAX_BYTES = 2 * 1024 * 1024;

    private final SafeHttpClient httpClient;

    /** Defaults to refusing private networks; tests use this constructor. */
    public BookmarkMetadataService() {
        this(false);
    }

    public BookmarkMetadataService(boolean allowPrivateHosts) {
        this(new SafeHttpClient(allowPrivateHosts));
    }

    @Autowired
    public BookmarkMetadataService(SafeHttpClient httpClient) {
        this.httpClient = httpClient;
    }

    public BookmarkMetadata fetch(String url) {
        return httpClient
                .get(url, MAX_BYTES)
                .map(response -> {
                    try {
                        Document document = Jsoup.parse(
                                new ByteArrayInputStream(response.body()),
                                extractCharset(response.contentType()),
                                response.uri().toString());
                        return parse(document, response.uri());
                    } catch (IOException e) {
                        LOGGER.warn("Failed to parse metadata for {}", url, e);
                        return BookmarkMetadata.empty();
                    }
                })
                .orElseGet(BookmarkMetadata::empty);
    }

    /** Extracts metadata from a parsed page; relative URLs are resolved against {@code baseUri}. */
    static BookmarkMetadata parse(Document document, URI baseUri) {
        // Open Graph values are usually cleaner than <title> (no " | Site name" suffix)
        String title = firstNonBlank(metaContent(document, "property", "og:title"), document.title());
        String description = firstNonBlank(
                metaContent(document, "name", "description"), metaContent(document, "property", "og:description"));
        String ogImage = SafeHttpClient.resolveUrl(baseUri, normalize(metaContent(document, "property", "og:image")));
        String canonical = SafeHttpClient.resolveUrl(baseUri, normalize(linkHref(document, "canonical")));
        String favicon = SafeHttpClient.resolveUrl(baseUri, normalize(faviconHref(document)));
        if (favicon == null) {
            favicon = SafeHttpClient.resolveUrl(baseUri, "/favicon.ico");
        }
        return new BookmarkMetadata(title, description, ogImage, favicon, canonical, keywords(document));
    }

    private static List<String> keywords(Document document) {
        String content = metaContent(document, "name", "keywords");
        if (content == null) {
            return List.of();
        }
        return Arrays.stream(content.split(","))
                .map(String::trim)
                .filter(keyword -> !keyword.isEmpty() && keyword.length() <= 40)
                .map(keyword -> keyword.toLowerCase(Locale.ROOT))
                .distinct()
                .limit(MAX_KEYWORDS)
                .toList();
    }

    private static String firstNonBlank(String... values) {
        for (String value : values) {
            String normalized = normalize(value);
            if (normalized != null) {
                return normalized;
            }
        }
        return null;
    }

    private static String metaContent(Document document, String attribute, String value) {
        Element element = document.selectFirst("meta[" + attribute + "=\"" + value + "\"]");
        if (element == null) {
            return null;
        }
        return element.attr("content");
    }

    private static String linkHref(Document document, String relValue) {
        Element element = document.selectFirst("link[rel~=(?i)" + relValue + "]");
        if (element == null) {
            return null;
        }
        return element.attr("href");
    }

    private static String faviconHref(Document document) {
        Element element = document.selectFirst("link[rel~=(?i)icon]");
        if (element == null) {
            return null;
        }
        return element.attr("href");
    }

    private static String normalize(String value) {
        if (value == null) {
            return null;
        }
        String trimmed = value.trim();
        return trimmed.isEmpty() ? null : trimmed;
    }

    private static String extractCharset(String contentType) {
        if (contentType == null) {
            return null;
        }
        String[] parts = contentType.split(";");
        for (String part : parts) {
            String trimmed = part.trim();
            if (trimmed.toLowerCase().startsWith("charset=")) {
                String value = trimmed.substring("charset=".length()).trim();
                return value.isEmpty() ? null : value;
            }
        }
        return null;
    }

    public record BookmarkMetadata(
            String title, String description, String ogImage, String favicon, String canonical, List<String> keywords) {
        public static BookmarkMetadata empty() {
            return new BookmarkMetadata(null, null, null, null, null, List.of());
        }
    }
}
