package de.serbroda.ragbag.config;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.config.properties.AppProperties;
import de.serbroda.ragbag.generated.api.AppApi;
import java.util.Map;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RequiredArgsConstructor
@RequestMapping(PUBLIC_API_PREFIX)
@RestController
public class AppConfigController implements AppApi {

    private final AppProperties appProperties;

    @Override
    public ResponseEntity<Map<String, String>> featureFlags() {
        return ResponseEntity.ok(Map.of(
                "registration_enabled", String.valueOf(appProperties.features().registrationEnabled())));
    }
}
