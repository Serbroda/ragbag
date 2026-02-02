package de.serbroda.ragbag.model.dto;

import com.fasterxml.jackson.annotation.JsonIdentityInfo;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.ObjectIdGenerators;
import de.serbroda.ragbag.model.shared.CollectionVisibility;

import java.util.List;

@JsonIdentityInfo(generator = ObjectIdGenerators.PropertyGenerator.class, property = "id")
public record CollectionTreeDto(
        @JsonProperty("id") String id,
        @JsonProperty("name") String name,
        @JsonProperty("visibility") CollectionVisibility visibility,
        @JsonProperty("parent_id") String parentId,
        @JsonProperty("children") List<CollectionTreeDto> children
) {
}
