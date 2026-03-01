package de.serbroda.ragbag;

import de.serbroda.ragbag.config.properties.SecurityProperties;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;

@EnableConfigurationProperties({SecurityProperties.class})
@SpringBootApplication
public class RagbagServerApplication {

    public static void main(String[] args) {
        SpringApplication.run(RagbagServerApplication.class, args);
    }
}
