package de.serbroda.ragbag.config;

import de.serbroda.ragbag.service.DataInitializer;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.context.annotation.Configuration;

@Configuration
public class AppConfig {

    @Autowired
    void init(DataInitializer initializer) {
        initializer.initializeData();
    }
}
