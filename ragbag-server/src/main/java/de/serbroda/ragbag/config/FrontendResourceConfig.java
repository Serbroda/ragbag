package de.serbroda.ragbag.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.ResourceHandlerRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

@Configuration
public class FrontendResourceConfig implements WebMvcConfigurer {

    @Override
    public void addResourceHandlers(ResourceHandlerRegistry registry) {
        registry.addResourceHandler("/app/assets/**").addResourceLocations("classpath:/static/assets/");
        // Top-level files from the Vite public/ directory (favicon, icons)
        registry.addResourceHandler("/app/*.ico", "/app/*.svg", "/app/*.png")
                .addResourceLocations("classpath:/static/");
    }
}
