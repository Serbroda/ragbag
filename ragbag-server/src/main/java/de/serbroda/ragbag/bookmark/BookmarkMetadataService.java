package de.serbroda.ragbag.bookmark;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.Inet6Address;
import java.net.InetAddress;
import java.net.URI;
import java.net.URISyntaxException;
import java.net.UnknownHostException;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.Arrays;
import java.util.List;
import java.util.Locale;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

@Service
public class BookmarkMetadataService {

    private static final Logger LOGGER = LoggerFactory.getLogger(BookmarkMetadataService.class);

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(10);
    private static final int MAX_REDIRECTS = 5;
    private static final int MAX_KEYWORDS = 8;
    private static final int MAX_BYTES = 2 * 1024 * 1024;
    private static final String USER_AGENT = "ragbag/1.0";

    private final boolean allowPrivateHosts;

    /** Defaults to refusing private networks; tests use this constructor. */
    public BookmarkMetadataService() {
        this(false);
    }

    /**
     * @param allowPrivateHosts lets self-hosters (homelabs) fetch metadata from their own network; loopback
     *     and cloud metadata addresses stay blocked either way
     */
    @Autowired
    public BookmarkMetadataService(@Value("${app.metadata.allow-private-hosts:false}") boolean allowPrivateHosts) {
        this.allowPrivateHosts = allowPrivateHosts;
    }

    private final HttpClient httpClient = HttpClient.newBuilder()
            .connectTimeout(REQUEST_TIMEOUT)
            // Redirects are followed manually so every hop passes the address check
            .followRedirects(HttpClient.Redirect.NEVER)
            .build();

    public BookmarkMetadata fetch(String url) {
        URI uri = toHttpUri(url);
        if (uri == null) {
            return BookmarkMetadata.empty();
        }

        try {
            HttpResponse<InputStream> response = null;
            for (int hop = 0; hop <= MAX_REDIRECTS; hop++) {
                if (!isAllowedHost(uri.getHost())) {
                    LOGGER.warn("Refusing to fetch metadata from disallowed address {}", uri);
                    return BookmarkMetadata.empty();
                }
                HttpRequest request = HttpRequest.newBuilder(uri)
                        .timeout(REQUEST_TIMEOUT)
                        .header("User-Agent", USER_AGENT)
                        .GET()
                        .build();
                response = httpClient.send(request, HttpResponse.BodyHandlers.ofInputStream());
                int status = response.statusCode();
                if (status < 300 || status >= 400) {
                    break;
                }
                response.body().close();
                URI current = uri;
                URI next = response.headers()
                        .firstValue("Location")
                        .map(location -> toHttpUri(resolveUrl(current, location)))
                        .orElse(null);
                if (next == null) {
                    return BookmarkMetadata.empty();
                }
                uri = next;
            }
            if (response == null || response.statusCode() < 200 || response.statusCode() >= 300) {
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

            return parse(document, uri);
        } catch (IOException e) {
            LOGGER.warn("Failed to fetch metadata for {}", url, e);
            return BookmarkMetadata.empty();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOGGER.warn("Metadata fetch interrupted for {}", url, e);
            return BookmarkMetadata.empty();
        }
    }

    /** Extracts metadata from a parsed page; relative URLs are resolved against {@code baseUri}. */
    static BookmarkMetadata parse(Document document, URI baseUri) {
        // Open Graph values are usually cleaner than <title> (no " | Site name" suffix)
        String title = firstNonBlank(metaContent(document, "property", "og:title"), document.title());
        String description = firstNonBlank(
                metaContent(document, "name", "description"), metaContent(document, "property", "og:description"));
        String ogImage = resolveUrl(baseUri, normalize(metaContent(document, "property", "og:image")));
        String canonical = resolveUrl(baseUri, normalize(linkHref(document, "canonical")));
        String favicon = resolveUrl(baseUri, normalize(faviconHref(document)));
        if (favicon == null) {
            favicon = resolveUrl(baseUri, "/favicon.ico");
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

    private boolean isAllowedHost(String host) {
        try {
            for (InetAddress address : InetAddress.getAllByName(host)) {
                // Never reachable: the server itself and link-local (incl. 169.254.169.254 cloud metadata)
                if (address.isLoopbackAddress()
                        || address.isAnyLocalAddress()
                        || address.isLinkLocalAddress()
                        || address.isMulticastAddress()) {
                    return false;
                }
                if (!allowPrivateHosts && (address.isSiteLocalAddress() || isUniqueLocalIpv6(address))) {
                    return false;
                }
            }
            return true;
        } catch (UnknownHostException e) {
            return false;
        }
    }

    private static boolean isUniqueLocalIpv6(InetAddress address) {
        // fc00::/7 is not covered by isSiteLocalAddress()
        return address instanceof Inet6Address && (address.getAddress()[0] & 0xfe) == 0xfc;
    }

    private static URI toHttpUri(String url) {
        if (url == null) {
            return null;
        }
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

    public record BookmarkMetadata(
            String title, String description, String ogImage, String favicon, String canonical, List<String> keywords) {
        public static BookmarkMetadata empty() {
            return new BookmarkMetadata(null, null, null, null, null, List.of());
        }
    }
}
