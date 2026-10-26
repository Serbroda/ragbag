package de.serbroda.ragbag.auth;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.config.properties.AppProperties;
import de.serbroda.ragbag.generated.api.AuthenticationApi;
import de.serbroda.ragbag.generated.model.ChangePasswordRequest;
import de.serbroda.ragbag.generated.model.LoginRequest;
import de.serbroda.ragbag.generated.model.LoginResponse;
import de.serbroda.ragbag.generated.model.RegisterRequest;
import de.serbroda.ragbag.generated.model.UserDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import java.time.Duration;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.DisabledException;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtException;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.context.request.RequestContextHolder;
import org.springframework.web.context.request.ServletRequestAttributes;

@Slf4j
@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class AuthController implements AuthenticationApi {

    private static final String REFRESH_COOKIE = "refresh_token";
    private static final String REFRESH_COOKIE_PATH = PUBLIC_API_PREFIX + "/auth/refresh";

    private final AuthenticationManager authenticationManager;
    private final JwtService jwtService;
    private final JwtDecoder jwtDecoder;
    private final UserService userService;
    private final RegisterService registerService;
    private final AppProperties appProperties;

    @Value("${security.jwt.secure-cookie:false}")
    private boolean secureCookie;

    @Override
    public ResponseEntity<LoginResponse> login(LoginRequest loginRequest) {
        Authentication auth = authenticationManager.authenticate(
                new UsernamePasswordAuthenticationToken(loginRequest.getUsername(), loginRequest.getPassword()));

        UserPrincipal principal = (UserPrincipal) auth.getPrincipal();

        User user = userService
                .findUserById(principal.getUserId())
                .orElseThrow(() -> new DisabledException("User blocked"));

        return issueTokens(user);
    }

    @Override
    public ResponseEntity<LoginResponse> changePassword(ChangePasswordRequest changePasswordRequest) {
        User user = userService.changePassword(
                SecurityUtils.currentUserId(),
                changePasswordRequest.getCurrentPassword(),
                changePasswordRequest.getNewPassword());
        // The old tokens of this device are invalid now as well, so it gets fresh ones
        return issueTokens(user);
    }

    private ResponseEntity<LoginResponse> issueTokens(User user) {
        String accessToken = jwtService.generateAccessToken(user.getId(), user.getUsername(), user.getTokenVersion());

        String refreshToken = jwtService.generateRefreshToken(user.getId(), user.getTokenVersion());

        ResponseCookie refreshCookie = ResponseCookie.from(REFRESH_COOKIE, refreshToken)
                .httpOnly(true)
                .secure(secureCookie())
                .sameSite("Strict")
                .path(REFRESH_COOKIE_PATH)
                .maxAge(Duration.ofMinutes(jwtService.getExpirationRefreshTokenMinutes()))
                .build();

        return ResponseEntity.ok()
                .header(HttpHeaders.SET_COOKIE, refreshCookie.toString())
                .body(new LoginResponse(accessToken));
    }

    @Override
    public ResponseEntity<LoginResponse> refresh(String refreshToken) {
        if (refreshToken == null) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED).build();
        }

        Jwt jwt;
        try {
            jwt = jwtDecoder.decode(refreshToken);
        } catch (JwtException e) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED).build();
        }

        if (!JwtService.TokenType.REFRESH.name().equals(jwt.getClaimAsString("type"))) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED).build();
        }

        User user = userService.findUserById(jwt.getSubject()).orElse(null);
        Number tokenVersion = jwt.getClaim("token_version");
        if (user == null || tokenVersion == null || tokenVersion.longValue() < user.getTokenVersion()) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED).build();
        }

        String newAccessToken =
                jwtService.generateAccessToken(user.getId(), user.getUsername(), user.getTokenVersion());

        return ResponseEntity.ok(new LoginResponse(newAccessToken));
    }

    @Override
    public ResponseEntity<Void> logout() {
        return ResponseEntity.noContent()
                .header(HttpHeaders.SET_COOKIE, deleteRefreshCookie().toString())
                .build();
    }

    @Override
    public ResponseEntity<Void> logoutAll() {
        userService.incrementTokenVersion(SecurityUtils.currentUserId());

        return ResponseEntity.noContent()
                .header(HttpHeaders.SET_COOKIE, deleteRefreshCookie().toString())
                .build();
    }

    /** Forced by configuration, otherwise whenever the browser talks HTTPS (directly or via a reverse proxy). */
    private boolean secureCookie() {
        return secureCookie
                || (RequestContextHolder.getRequestAttributes() instanceof ServletRequestAttributes attributes
                        && attributes.getRequest().isSecure());
    }

    private ResponseCookie deleteRefreshCookie() {
        return ResponseCookie.from(REFRESH_COOKIE, "")
                .path(REFRESH_COOKIE_PATH)
                .maxAge(0)
                .httpOnly(true)
                .secure(secureCookie())
                .sameSite("Strict")
                .build();
    }

    @Override
    public ResponseEntity<UserDto> register(RegisterRequest registerRequest) {
        if (!appProperties.features().registrationEnabled()) {
            log.warn("Registration attempt while registration is disabled");
            return ResponseEntity.status(HttpStatus.FORBIDDEN).build();
        }

        // Checked here rather than in RegisterService, which also creates the dev demo user (admin/test123)
        UserService.requireValidPassword(registerRequest.getPassword());
        User user = registerService.register(
                registerRequest.getUsername(), registerRequest.getEmail(), registerRequest.getPassword());
        return ResponseEntity.ok(new UserDto(user.getId(), user.getUsername(), user.getEmail()));
    }
}
