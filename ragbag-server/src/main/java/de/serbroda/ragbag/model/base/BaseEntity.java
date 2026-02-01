package de.serbroda.ragbag.model.base;

import java.util.Date;

public interface BaseEntity<ID> {

    ID getId();

    void setId(ID id);

    int getVersion();

    void setVersion(int version);

    Date getCreatedAt();

    void setCreatedAt(Date createdAt);

    Date getUpdatedAt();

    void setUpdatedAt(Date updatedAt);

}