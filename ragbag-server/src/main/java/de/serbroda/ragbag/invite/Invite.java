package de.serbroda.ragbag.invite;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import de.serbroda.ragbag.user.User;
import jakarta.persistence.*;
import java.time.Instant;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
@Entity
@Table(name = "invites")
public class Invite extends AbstractBaseEntity {

    @Column(nullable = false, length = 64)
    private String token;

    @Column(nullable = false)
    @Enumerated(EnumType.STRING)
    private InviteTargetType targetType;

    @Column(nullable = false, length = 36)
    private String targetId;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "created_by", nullable = false)
    private User createdBy;

    @Column(name = "expires_at")
    protected Instant expiresAt;

    @Column(nullable = false)
    private int maxUses = -1;

    @Column(nullable = false)
    private int usedCount = 0;

    @Column(nullable = false, length = 80)
    private String role;

    @Transient
    public boolean isExpired() {
        if (expiresAt != null && expiresAt.isBefore(Instant.now())) {
            return true;
        }
        if (maxUses > -1 && usedCount >= maxUses) {
            return true;
        }
        return false;
    }
}
