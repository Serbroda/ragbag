package de.serbroda.ragbag.model.dto;

import com.fasterxml.jackson.annotation.JsonIdentityInfo;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.ObjectIdGenerators;

@JsonIdentityInfo(generator = ObjectIdGenerators.PropertyGenerator.class, property = "id")
public record CollectionDto(@JsonProperty("id") String id,
                            @JsonProperty("name") String name,
                            @JsonProperty("description") String description) {
}
