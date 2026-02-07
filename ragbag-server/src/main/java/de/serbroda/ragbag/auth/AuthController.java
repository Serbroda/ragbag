package de.serbroda.ragbag.auth;

import static de.serbroda.ragbag.shared.ApiConstants.PUBLIC_API_PREFIX;

import de.serbroda.ragbag.generated.api.AuthApi;
import de.serbroda.ragbag.generated.model.LoginRequest;
import de.serbroda.ragbag.generated.model.LoginResponse;
import de.serbroda.ragbag.generated.model.RegisterRequest;
import de.serbroda.ragbag.generated.model.UserDto;
import de.serbroda.ragbag.security.SecurityUtils;
import de.serbroda.ragbag.user.User;
import de.serbroda.ragbag.user.UserService;
import java.time.Duration;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.BadCredentialsException;
import org.springframework.security.authentication.DisabledException;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.Authentication;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@RequestMapping(PUBLIC_API_PREFIX)
public class AuthController implements AuthApi {

    private final AuthenticationManager authenticationManager;
    private final JwtService jwtService;
    private final JwtDecoder jwtDecoder;
    private final UserService userService;
    private final RegisterService registerService;

    @Override
    public ResponseEntity<LoginResponse> login(LoginRequest loginRequest) {
        Authentication auth = authenticationManager.authenticate(
                new UsernamePasswordAuthenticationToken(loginRequest.getUsername(), loginRequest.getPassword()));

        UserPrincipal principal = (UserPrincipal) auth.getPrincipal();

        User user = userService
                .findUserById(principal.getUserId())
                .orElseThrow(() -> new DisabledException("User blocked"));

        String accessToken = jwtService.generateAccessToken(user.getId(), user.getUsername(), user.getTokenVersion());

        String refreshToken = jwtService.generateRefreshToken(principal.getUserId());

        ResponseCookie refreshCookie = ResponseCookie.from("refresh_token", refreshToken)
                .httpOnly(true)
                .secure(false) // prod: true
                .sameSite("Lax") // oder Strict
                .path("/api/auth/refresh")
                .maxAge(Duration.ofDays(14))
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

        Jwt jwt = jwtDecoder.decode(refreshToken);

        if (!"REFRESH".equals(jwt.getClaimAsString("type"))) {
            throw new BadCredentialsException("Invalid refresh token");
        }

        String userId = jwt.getSubject();

        User user = userService.findUserById(userId).orElseThrow(() -> new DisabledException("User blocked"));

        String newAccessToken =
                jwtService.generateAccessToken(user.getId(), user.getUsername(), user.getTokenVersion());

        return ResponseEntity.ok(new LoginResponse(newAccessToken));
    }

    @Override
    public ResponseEntity<Void> logoutAll() {
        userService.incrementTokenVersion(SecurityUtils.currentUserId());

        ResponseCookie deleteCookie = ResponseCookie.from("refresh_token", "")
                .path("/api/auth/refresh")
                .maxAge(0)
                .httpOnly(true)
                .build();

        return ResponseEntity.noContent()
                .header(HttpHeaders.SET_COOKIE, deleteCookie.toString())
                .build();
    }

    @Override
    public ResponseEntity<UserDto> register(RegisterRequest registerRequest) {
        User user = registerService.register(
                registerRequest.getUsername(), registerRequest.getEmail(), registerRequest.getPassword());
        return ResponseEntity.ok(new UserDto(user.getId(), user.getUsername(), user.getEmail()));
    }
}
