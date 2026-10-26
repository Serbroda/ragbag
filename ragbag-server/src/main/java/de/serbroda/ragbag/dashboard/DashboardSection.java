package de.serbroda.ragbag.dashboard;

import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import jakarta.persistence.*;
import java.util.ArrayList;
import java.util.List;
import lombok.Getter;
import lombok.Setter;

/** One section of a user's personal dashboard in a space. */
@Getter
@Setter
@Entity
@Table(name = "dashboard_sections")
public class DashboardSection extends AbstractBaseEntity {

    @Column(name = "user_id", nullable = false, length = 36)
    private String userId;

    @Column(name = "space_id", nullable = false, length = 36)
    private String spaceId;

    @Column(nullable = false, length = 80)
    private String title;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20)
    private DashboardSectionType type;

    @Column(nullable = false)
    private int position;

    @OneToMany(mappedBy = "section", cascade = CascadeType.ALL, orphanRemoval = true)
    @OrderBy("position")
    private List<DashboardItem> items = new ArrayList<>();
}
