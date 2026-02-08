package de.serbroda.ragbag.collection;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import de.serbroda.ragbag.space.Space;
import de.serbroda.ragbag.user.User;
import jakarta.persistence.*;
import java.util.HashSet;
import java.util.Set;
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

    @ManyToOne
    private Space space;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "parent_id")
    private Collection parent;

    @OneToMany(mappedBy = "parent", cascade = CascadeType.ALL, orphanRemoval = true)
    private Set<Collection> children = new HashSet<>();

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;
}
