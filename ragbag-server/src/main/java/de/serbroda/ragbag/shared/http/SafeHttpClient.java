package de.serbroda.ragbag.shared.http;

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
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

/**
 * Fetches user supplied URLs on the server without turning it into a proxy into its own network (SSRF): every
 * redirect hop is checked against the address rules and responses are size limited.
 */
@Component
public class SafeHttpClient {

    private static final Logger LOGGER = LoggerFactory.getLogger(SafeHttpClient.class);

    private static final Duration REQUEST_TIMEOUT = Duration.ofSeconds(10);
    private static final int MAX_REDIRECTS = 5;
    private static final String USER_AGENT = "ragbag/1.0";

    private final boolean allowPrivateHosts;

    private final HttpClient httpClient = HttpClient.newBuilder()
            .connectTimeout(REQUEST_TIMEOUT)
            // Redirects are followed manually so every hop passes the address check
            .followRedirects(HttpClient.Redirect.NEVER)
            .build();

    /**
     * @param allowPrivateHosts lets self-hosters (homelabs) fetch from their own network; loopback and cloud
     *     metadata addresses stay blocked either way
     */
    @Autowired
    public SafeHttpClient(@Value("${app.metadata.allow-private-hosts:false}") boolean allowPrivateHosts) {
        this.allowPrivateHosts = allowPrivateHosts;
    }

    /**
     * GETs the URL and returns the body of a 2xx response, or empty if the target is not allowed, unreachable or
     * larger than {@code maxBytes}.
     */
    public Optional<Response> get(String url, int maxBytes) {
        URI uri = toHttpUri(url);
        if (uri == null) {
            return Optional.empty();
        }

        try {
            HttpResponse<InputStream> response = null;
            for (int hop = 0; hop <= MAX_REDIRECTS; hop++) {
                if (!isAllowedHost(uri.getHost())) {
                    LOGGER.warn("Refusing to fetch disallowed address {}", uri);
                    return Optional.empty();
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
                    return Optional.empty();
                }
                uri = next;
            }
            if (response == null || response.statusCode() < 200 || response.statusCode() >= 300) {
                if (response != null) {
                    response.body().close();
                }
                return Optional.empty();
            }

            byte[] body;
            try (InputStream stream = response.body()) {
                body = readLimited(stream, maxBytes);
            }
            if (body == null) {
                return Optional.empty();
            }
            return Optional.of(new Response(
                    uri, response.headers().firstValue("Content-Type").orElse(null), body));
        } catch (IOException | IllegalArgumentException e) {
            LOGGER.warn("Failed to fetch {}", url, e);
            return Optional.empty();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LOGGER.warn("Fetch interrupted for {}", url, e);
            return Optional.empty();
        }
    }

    /**
     * Normalizes user input to an absolute http(s) URL (adding https:// if the scheme is missing). Anything else
     * (e.g. {@code javascript:}) would be rendered as a clickable link for every member of a space.
     */
    public static String requireHttpUrl(String value) {
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException("URL must not be empty");
        }
        String trimmed = value.trim();
        if (!trimmed.contains("://")) {
            trimmed = "https://" + trimmed;
        }
        try {
            URI uri = new URI(trimmed);
            String scheme = uri.getScheme();
            if (scheme == null
                    || !(scheme.equalsIgnoreCase("http") || scheme.equalsIgnoreCase("https"))
                    || uri.getHost() == null) {
                throw new IllegalArgumentException("Only http and https URLs are supported");
            }
            return trimmed;
        } catch (URISyntaxException e) {
            throw new IllegalArgumentException("Invalid URL: " + value);
        }
    }

    /** Resolves a possibly relative URL against {@code baseUri}; null if it cannot be parsed. */
    public static String resolveUrl(URI baseUri, String value) {
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

    /** A successful response; {@code uri} is the final address after redirects. */
    public record Response(URI uri, String contentType, byte[] body) {}
}
