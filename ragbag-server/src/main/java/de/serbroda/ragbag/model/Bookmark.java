package de.serbroda.ragbag.model;

import de.serbroda.ragbag.model.base.AbstractBaseEntity;
import jakarta.persistence.Entity;
import jakarta.persistence.ManyToOne;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "bookmarks")
public class Bookmark extends AbstractBaseEntity {

    private String url;
    private String title;
    private String description;

    @ManyToOne
    private Collection collection;

}
