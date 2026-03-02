package de.serbroda.ragbag.config.properties;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "app")
public record AppProperties(FeaturesProperties features) {}
