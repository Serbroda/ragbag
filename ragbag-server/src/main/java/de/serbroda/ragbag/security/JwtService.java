package de.serbroda.ragbag.security;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.Map;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.oauth2.jose.jws.MacAlgorithm;
import org.springframework.security.oauth2.jwt.JwsHeader;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtEncoder;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.stereotype.Service;

@Service
public class JwtService {

    public enum TokenType {
        ACCESS,
        REFRESH
    }

    private final JwtEncoder encoder;
    private final String issuer;
    private final int expirationAccessTokenMinutes;
    private final int expirationRefreshTokenMinutes;

    public JwtService(
            JwtEncoder encoder,
            @Value("${security.jwt.issuer:ragbag}") String issuer,
            @Value("${security.jwt.expiration.accessToken:15}") int expirationAccessTokenMinutes,
            @Value("${security.jwt.expiration.refreshToken:20160}") int expirationRefreshTokenMinutes) {
        this.encoder = encoder;
        this.issuer = issuer;
        this.expirationAccessTokenMinutes = expirationAccessTokenMinutes;
        this.expirationRefreshTokenMinutes = expirationRefreshTokenMinutes;
    }

    public String generateToken(String subject, Map<String, Object> claims, Instant expiresAt) {
        Instant now = Instant.now();
        final String jti = UUID.randomUUID().toString();

        JwtClaimsSet claimsSet = JwtClaimsSet.builder()
                .issuer(issuer)
                .issuedAt(now)
                .expiresAt(expiresAt)
                .subject(subject)
                .id(jti)
                .claims((c) -> c.putAll(claims))
                .build();

        return encode(claimsSet);
    }

    // spotless:off
    public String generateAccessToken(String userId, String username, long tokenVersion) {
        return generateToken(
                userId,
                Map.of(
                        "username", username,
                        "type", TokenType.REFRESH.name(),
                        "token_version", tokenVersion,
                        "roles", new String[] {"USER"}
                ),
                Instant.now().plus(expirationAccessTokenMinutes, ChronoUnit.MINUTES));
    }

    public String generateRefreshToken(String userId) {
        return generateToken(
                userId,
                Map.of(
                        "type", TokenType.REFRESH.name(),
                        "roles", new String[] {"USER"}
                ),
                Instant.now().plus(expirationRefreshTokenMinutes, ChronoUnit.MINUTES));
    }

    // spotless:on
    private String encode(JwtClaimsSet claims) {
        JwsHeader header = JwsHeader.with(MacAlgorithm.HS256).type("JWT").build();
        return encoder.encode(JwtEncoderParameters.from(header, claims)).getTokenValue();
    }
}
