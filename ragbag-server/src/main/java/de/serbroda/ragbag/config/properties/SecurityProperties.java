package de.serbroda.ragbag.config.properties;

import java.util.Set;
import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "security")
public record SecurityProperties(Set<String> apiKeys) {}
