package de.serbroda.ragbag.bookmark;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.net.URISyntaxException;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class BookmarkMetadataService {

    private static final Logger LOGGER = LoggerFactory.getLogger(BookmarkMetadataService.class);

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(30);
    private static final int MAX_BYTES = 2 * 1024 * 1024;
    private static final String USER_AGENT = "ragbag/1.0";

    private final HttpClient httpClient = HttpClient.newBuilder()
            .connectTimeout(REQUEST_TIMEOUT)
            .followRedirects(HttpClient.Redirect.NORMAL)
            .build();

    public BookmarkMetadata fetch(String url) {
        URI uri = toHttpUri(url);
        if (uri == null) {
            return BookmarkMetadata.empty();
        }

        HttpRequest request = HttpRequest.newBuilder(uri)
                .timeout(REQUEST_TIMEOUT)
                .header("User-Agent", USER_AGENT)
                .GET()
                .build();

        try {
            HttpResponse<InputStream> response = httpClient.send(request, HttpResponse.BodyHandlers.ofInputStream());
            if (response.statusCode() < 200 || response.statusCode() >= 300) {
                return BookmarkMetadata.empty();
            }

            byte[] body;
            try (InputStream stream = response.body()) {
                body = readLimited(stream, MAX_BYTES);
            }
            if (body == null) {
                return BookmarkMetadata.empty();
            }

            String charset = response.headers()
                    .firstValue("Content-Type")
                    .map(BookmarkMetadataService::extractCharset)
                    .orElse(null);
            Document document = Jsoup.parse(new ByteArrayInputStream(body), charset, uri.toString());

            String title = normalize(document.title());
            String description = normalize(metaContent(document, "name", "description"));
            String ogImage = normalize(metaContent(document, "property", "og:image"));
            String canonical = normalize(linkHref(document, "canonical"));
            String favicon = normalize(faviconHref(document));

            ogImage = resolveUrl(uri, ogImage);
            canonical = resolveUrl(uri, canonical);
            favicon = resolveUrl(uri, favicon);

            if (favicon == null) {
                favicon = resolveUrl(uri, "/favicon.ico");
            }

            return new BookmarkMetadata(title, description, ogImage, favicon, canonical);
        } catch (IOException e) {
            LOGGER.warn("Failed to fetch metadata for {}", url, e);
            return BookmarkMetadata.empty();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOGGER.warn("Metadata fetch interrupted for {}", url, e);
            return BookmarkMetadata.empty();
        }
    }

    private static String metaContent(Document document, String attribute, String value) {
        Element element = document.selectFirst("meta[" + attribute + "=" + value + "]");
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

    private static String resolveUrl(URI baseUri, String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        try {
            URI uri = new URI(value);
            if (uri.isAbsolute()) {
                return uri.toString();
            }
            if (baseUri == null) {
                return null;
            }
            return baseUri.resolve(uri).toString();
        } catch (URISyntaxException e) {
            return null;
        }
    }

    private static URI toHttpUri(String url) {
        try {
            URI uri = new URI(url);
            String scheme = uri.getScheme();
            if (scheme == null) {
                return null;
            }
            if (!scheme.equalsIgnoreCase("http") && !scheme.equalsIgnoreCase("https")) {
                return null;
            }
            if (uri.getHost() == null) {
                return null;
            }
            return uri;
        } catch (URISyntaxException e) {
            return null;
        }
    }

    private static String normalize(String value) {
        if (value == null) {
            return null;
        }
        String trimmed = value.trim();
        return trimmed.isEmpty() ? null : trimmed;
    }

    private static byte[] readLimited(InputStream stream, int maxBytes) throws IOException {
        ByteArrayOutputStream output = new ByteArrayOutputStream();
        byte[] buffer = new byte[8192];
        int total = 0;
        int read;
        while ((read = stream.read(buffer)) != -1) {
            total += read;
            if (total > maxBytes) {
                return null;
            }
            output.write(buffer, 0, read);
        }
        return output.toByteArray();
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

    public record BookmarkMetadata(String title, String description, String ogImage, String favicon, String canonical) {
        public static BookmarkMetadata empty() {
            return new BookmarkMetadata(null, null, null, null, null);
        }
    }
}
