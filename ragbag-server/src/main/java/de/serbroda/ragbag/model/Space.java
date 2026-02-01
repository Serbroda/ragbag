package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import de.serbroda.ragbag.model.shared.SpaceVisibility;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "spaces")
public class Space extends AbstractBaseEntity {

    private String name;
    private String description;

    @Enumerated(EnumType.STRING)
    private SpaceVisibility visibility = SpaceVisibility.PRIVATE;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    @Override
    public String getId() {
        return doGetId();
    }
}
