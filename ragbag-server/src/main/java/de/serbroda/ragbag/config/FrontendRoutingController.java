package de.serbroda.ragbag.config;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;

@Controller
public class FrontendRoutingController {

    @GetMapping({"/app", "/app/"})
    public String forwardRoot() {
        return "forward:/index.html";
    }

    @GetMapping("/")
    public String redirectToApp() {
        return "redirect:/app/";
    }

    // First segment without a dot so files like /app/vite.svg reach the resource handler
    @GetMapping("/app/{path:^(?!assets$)[^.]*$}/**")
    public String forwardRoutes() {
        return "forward:/index.html";
    }
}
