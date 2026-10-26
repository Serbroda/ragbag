package de.serbroda.ragbag.dashboard;

import de.serbroda.ragbag.bookmark.Bookmark;
import de.serbroda.ragbag.shared.base.AbstractBaseEntity;
import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;

/** A bookmark pinned to a LINKS section; removed by the database when the bookmark is deleted. */
@Getter
@Setter
@Entity
@Table(name = "dashboard_items")
public class DashboardItem extends AbstractBaseEntity {

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "section_id", nullable = false)
    private DashboardSection section;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @JoinColumn(name = "bookmark_id", nullable = false)
    private Bookmark bookmark;

    @Column(nullable = false)
    private int position;
}
