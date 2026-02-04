package de.serbroda.ragbag.model.dto;

import com.fasterxml.jackson.annotation.JsonIdentityInfo;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.ObjectIdGenerators;
import io.swagger.v3.oas.annotations.media.Schema;

@Schema(description = "A space")
@JsonIdentityInfo(generator = ObjectIdGenerators.PropertyGenerator.class, property = "id")
public record SpaceDto(@JsonProperty("id") String id,
                       @JsonProperty("name") String name,
                       @JsonProperty("description") String description) {

}
