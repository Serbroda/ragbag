package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.keys.SpaceMemberId;
import de.serbroda.ragbag.model.shared.SpaceMemberRole;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(
        name = "spaces_members",
        uniqueConstraints =
                @UniqueConstraint(
                        name = "uc_spaces_members",
                        columnNames = {"space_id", "user_id"}))
public class SpaceMember {

    @EmbeddedId
    private SpaceMemberId id;

    @ManyToOne(fetch = FetchType.LAZY)
    @MapsId("spaceId")
    @JoinColumn(name = "space_id", nullable = false)
    private Space space;

    @ManyToOne(fetch = FetchType.LAZY)
    @MapsId("userId")
    @JoinColumn(name = "user_id", nullable = false)
    private User user;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 80)
    private SpaceMemberRole role = SpaceMemberRole.VIEWER;
}
