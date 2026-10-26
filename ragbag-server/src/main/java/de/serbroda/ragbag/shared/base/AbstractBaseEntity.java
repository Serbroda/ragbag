package de.serbroda.ragbag.shared.base;

import jakarta.persistence.*;
import java.time.Instant;
import java.util.Objects;
import lombok.Getter;

@MappedSuperclass
public abstract class AbstractBaseEntity {

    @Getter
    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    @Column(length = 36, nullable = false, updatable = false)
    protected String id;

    @Version
    protected int version;

    @Getter
    @Column(name = "created_at", updatable = false)
    protected Instant createdAt;

    @Getter
    @Column(name = "updated_at")
    protected Instant updatedAt;

    @PrePersist
    protected void setCreatedAndLastModifiedOnCreate() {
        Instant now = Instant.now();
        createdAt = now;
        updatedAt = now;
    }

    @PreUpdate
    protected void setLastModifiedOnUpdate() {
        updatedAt = Instant.now();
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        AbstractBaseEntity that = (AbstractBaseEntity) o;
        return id != null && Objects.equals(id, that.id);
    }

    @Override
    public int hashCode() {
        // Constant per class so the hash stays stable when the id is assigned on persist
        return getClass().hashCode();
    }
}
