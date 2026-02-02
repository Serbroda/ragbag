package de.serbroda.ragbag.model;

import com.fasterxml.jackson.annotation.JsonIdentityInfo;
import com.fasterxml.jackson.annotation.ObjectIdGenerators;
import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import de.serbroda.ragbag.model.shared.SpaceVisibility;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

import java.util.HashSet;
import java.util.Set;

@JsonIdentityInfo(generator = ObjectIdGenerators.PropertyGenerator.class, property = "id")
@Getter
@Setter
@Entity
@Table(name = "spaces")
public class Space extends AbstractBaseEntity {

    @Column(nullable = false, length = 80)
    private String name;

    private String description;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 80)
    private SpaceVisibility visibility = SpaceVisibility.PRIVATE;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;

    @OneToMany(
            mappedBy = "space",
            cascade = CascadeType.ALL,
            orphanRemoval = true
    )
    private Set<SpaceMember> members = new HashSet<>();

}
