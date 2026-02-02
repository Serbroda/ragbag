package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import de.serbroda.ragbag.model.shared.CollectionVisibility;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "collections")
public class Collection extends AbstractBaseEntity {

    @Column(nullable = false, length = 80)
    private String name;
    private String description;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 80)
    private CollectionVisibility visibility = CollectionVisibility.INTERNAL;

    @ManyToOne
    private Space space;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;

}
