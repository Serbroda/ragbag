package de.serbroda.ragbag.bookmark;

import jakarta.persistence.Column;
import jakarta.persistence.Embeddable;
import jakarta.persistence.EmbeddedId;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import java.io.Serializable;
import java.time.Instant;
import lombok.AllArgsConstructor;
import lombok.EqualsAndHashCode;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

/** A bookmark the user starred; favorites are personal and never visible to other members. */
@Getter
@Setter
@NoArgsConstructor
@Entity
@Table(name = "bookmark_favorites")
public class BookmarkFavorite {

    @EmbeddedId
    private Id id;

    @Column(name = "created_at", nullable = false)
    private Instant createdAt;

    public BookmarkFavorite(String userId, String bookmarkId) {
        this.id = new Id(userId, bookmarkId);
        this.createdAt = Instant.now();
    }

    @Getter
    @Setter
    @NoArgsConstructor
    @AllArgsConstructor
    @EqualsAndHashCode
    @Embeddable
    public static class Id implements Serializable {

        @Column(name = "user_id", length = 36)
        private String userId;

        @Column(name = "bookmark_id", length = 36)
        private String bookmarkId;
    }
}
