package de.serbroda.ragbag.space;

/** Published when a user leaves a space or is removed from it, so personal data can be cleaned up. */
public record SpaceMembershipEndedEvent(String userId, String spaceId) {}
