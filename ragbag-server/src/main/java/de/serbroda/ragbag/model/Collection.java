package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "collections")
public class Collection extends AbstractBaseEntity {

    private String name;
    private String description;
    private String visibility;

    @ManyToOne
    private Space space;

}
