package de.serbroda.ragbag.space;

public enum SpacePermission {
    READ, // Permission to read space details and collections
    EDIT_SPACE, // Permission to edit space details (name, description)
    CREATE_COLLECTIONS, // Permission to create collections within the space
    EDIT_COLLECTIONS, // Permission to create and manage collections within the space
    DELETE_COLLECTIONS, // Permission to delete collections within the space
    CREATE_BOOKMARKS, // Permission to create bookmarks within the space
    EDIT_BOOKMARKS, // Permission to create and manage bookmarks within the space
    DELETE_BOOKMARKS // Permission to delete bookmarks within the space
}
