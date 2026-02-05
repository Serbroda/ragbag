package de.serbroda.ragbag.model.keys;

import jakarta.persistence.Embeddable;
import java.io.Serializable;
import lombok.*;

@Getter
@Setter
@AllArgsConstructor
@NoArgsConstructor
@Embeddable
@EqualsAndHashCode
public class SpaceMemberId implements Serializable {

    private String spaceId;
    private String userId;
}
