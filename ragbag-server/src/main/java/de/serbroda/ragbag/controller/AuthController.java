package de.serbroda.ragbag.controller;

import com.fasterxml.jackson.annotation.JsonProperty;
import de.serbroda.ragbag.model.User;
import de.serbroda.ragbag.security.JwtService;
import de.serbroda.ragbag.security.UserPrincipal;
import de.serbroda.ragbag.service.UserService;
import jakarta.servlet.http.HttpServletResponse;
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
@RequestMapping("/api/auth")
public class AuthController {

    private final AuthenticationManager authenticationManager;
    private final JwtService jwtService;
    private final JwtDecoder jwtDecoder;
    private final UserService userService;

    @PostMapping("/login")
    public ResponseEntity<LoginResponse> login(@RequestBody LoginRequest request, HttpServletResponse response) {
        Authentication auth = authenticationManager.authenticate(
                new UsernamePasswordAuthenticationToken(request.username(), request.password()));

        UserPrincipal principal = (UserPrincipal) auth.getPrincipal();

        String accessToken = jwtService.generateAccessToken(principal.getUserId(), principal.getUsername());

        String refreshToken = jwtService.generateRefreshToken(principal.getUserId());

        ResponseCookie refreshCookie = ResponseCookie.from("refresh_token", refreshToken)
                .httpOnly(true)
                .secure(false) // prod: true
                .sameSite("Lax") // oder Strict
                .path("/api/auth/refresh")
                .maxAge(Duration.ofDays(14))
                .build();

        response.addHeader(HttpHeaders.SET_COOKIE, refreshCookie.toString());

        return ResponseEntity.ok(new LoginResponse(accessToken));
    }

    @PostMapping("/refresh")
    public ResponseEntity<LoginResponse> refresh(
            @CookieValue(name = "refresh_token", required = false) String refreshToken) {
        if (refreshToken == null) {
            return ResponseEntity.status(HttpStatus.UNAUTHORIZED).build();
        }

        Jwt jwt = jwtDecoder.decode(refreshToken);

        if (!"REFRESH".equals(jwt.getClaimAsString("type"))) {
            throw new BadCredentialsException("Invalid refresh token");
        }

        String userId = jwt.getSubject();

        User user = userService.findUserById(userId).orElseThrow(() -> new DisabledException("User blocked"));

        String newAccessToken = jwtService.generateAccessToken(user.getId(), user.getUsername());

        return ResponseEntity.ok(new LoginResponse(newAccessToken));
    }

    @PostMapping("/logout")
    public ResponseEntity<Void> logout(HttpServletResponse response) {

        ResponseCookie deleteCookie = ResponseCookie.from("refresh_token", "")
                .path("/api/auth/refresh")
                .maxAge(0)
                .httpOnly(true)
                .build();

        response.addHeader(HttpHeaders.SET_COOKIE, deleteCookie.toString());

        return ResponseEntity.noContent().build();
    }

    public record LoginRequest(
            @JsonProperty("username") String username,
            @JsonProperty("password") String password) {}

    public record LoginResponse(
            @JsonProperty("access_token") String accessToken) {}
}
