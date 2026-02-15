package de.serbroda.ragbag.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.web.servlet.config.annotation.ResourceHandlerRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

@Configuration
public class FrontendResourceConfig implements WebMvcConfigurer {

    @Override
    public void addResourceHandlers(ResourceHandlerRegistry registry) {
        registry.addResourceHandler("/app/assets/**").addResourceLocations("classpath:/static/assets/");
        registry.addResourceHandler("/app/favicon.ico").addResourceLocations("classpath:/static/favicon.ico");
    }
}
