package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

import java.util.HashSet;
import java.util.Set;

@Getter
@Setter
@Entity
@Table(name = "users")
public class User extends AbstractBaseEntity {

    @Column(nullable = false, unique = true, length = 80)
    private String username;

    @Column(nullable = false, unique = true, length = 250)
    private String email;

    @Column(nullable = false, length = 60)
    private String password;

    @OneToMany(mappedBy = "user")
    private Set<SpaceMember> spaces = new HashSet<>();
}
