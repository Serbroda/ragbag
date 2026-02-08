package de.serbroda.ragbag.space;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import de.serbroda.ragbag.user.User;
import jakarta.persistence.*;
import java.util.HashSet;
import java.util.Set;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "spaces")
public class Space extends AbstractBaseEntity {

    @Column(nullable = false, length = 80)
    private String name;

    private String description;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;

    @OneToMany(mappedBy = "space", cascade = CascadeType.ALL, orphanRemoval = true)
    private Set<SpaceMember> members = new HashSet<>();
}
