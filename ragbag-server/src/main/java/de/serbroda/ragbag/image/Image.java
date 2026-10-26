package de.serbroda.ragbag.image;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.Getter;
import lombok.Setter;

/** An avatar or space icon. Images are immutable: replacing one stores a new image under a new id. */
@Getter
@Setter
@Entity
@Table(name = "images")
public class Image extends AbstractBaseEntity {

    @Column(name = "content_type", nullable = false, length = 50)
    private String contentType;

    @Column(nullable = false)
    private byte[] data;
}
