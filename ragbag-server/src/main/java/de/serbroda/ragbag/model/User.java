package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "users")
public class User extends AbstractBaseEntity {

    private String username;
    private String email;
    private String password;

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    @Override
    public String getId() {
        return doGetId();
    }
}
