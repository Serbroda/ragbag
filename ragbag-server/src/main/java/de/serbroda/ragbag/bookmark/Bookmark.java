package de.serbroda.ragbag.bookmark;

import de.serbroda.ragbag.collection.Collection;
import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import de.serbroda.ragbag.user.User;
import jakarta.persistence.*;
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

    @Column(name = "og_image")
    private String ogImage;

    @Column(name = "favicon")
    private String favicon;

    @Column(name = "canonical")
    private String canonical;

    @ManyToOne
    private Collection collection;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;
}
