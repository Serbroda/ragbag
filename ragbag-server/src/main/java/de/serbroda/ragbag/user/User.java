package de.serbroda.ragbag.user;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import de.serbroda.ragbag.space.SpaceMember;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.OneToMany;
import jakarta.persistence.Table;
import java.util.HashSet;
import java.util.Set;
import lombok.Getter;
import lombok.Setter;

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

    @Column(nullable = false)
    private int tokenVersion = 0;

    @Column(name = "avatar_image_id", length = 36)
    private String avatarImageId;

    @OneToMany(mappedBy = "user")
    private Set<SpaceMember> spaces = new HashSet<>();
}
