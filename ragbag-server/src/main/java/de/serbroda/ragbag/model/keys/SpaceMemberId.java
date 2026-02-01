package de.serbroda.ragbag.model.keys;

import jakarta.persistence.Embeddable;
import lombok.EqualsAndHashCode;
import lombok.Getter;
import lombok.Setter;

import java.io.Serializable;

@Getter
@Setter
@Embeddable
@EqualsAndHashCode
public class SpaceMemberId implements Serializable {

    private String spaceId;
    private String userId;
}