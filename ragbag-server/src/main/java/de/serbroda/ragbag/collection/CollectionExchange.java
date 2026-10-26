package de.serbroda.ragbag.collection;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.time.Instant;
import java.util.List;

/**
 * Exchange format of a collection export: the collection with its links and sub-collections, importable into
 * any collection of any installation.
 *
 * <p>{@code format_version} is raised on every incompatible change; the import converts older files step by
 * step (see {@link CollectionExchangeService#parse}) and rejects newer ones. {@code app_version} is informational.
 * Member lists of private collections are installation specific and therefore not exported: a private collection
 * is imported as private to the importing user.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record CollectionExchange(
        String format,
        @JsonProperty("format_version") int formatVersion,
        @JsonProperty("app_version") String appVersion,
        @JsonProperty("exported_at") Instant exportedAt,
        Node collection) {

    public static final String FORMAT = "ragbag-collection";
    public static final int FORMAT_VERSION = 1;

    @JsonIgnoreProperties(ignoreUnknown = true)
    public record Node(
            String name,
            String description,
            @JsonProperty("private") Boolean restricted,
            List<Link> links,
            List<Node> collections) {}

    @JsonIgnoreProperties(ignoreUnknown = true)
    public record Link(
            String url,
            String title,
            String description,
            List<String> tags,
            @JsonProperty("og_image") String ogImage,
            String favicon,
            String canonical,
            @JsonProperty("created_at") Instant createdAt) {}
}
