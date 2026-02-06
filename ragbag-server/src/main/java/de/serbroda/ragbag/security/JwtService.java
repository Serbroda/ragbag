package de.serbroda.ragbag.security;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
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
    private final long expirationSeconds;

    public JwtService(JwtEncoder encoder, @Value("${security.jwt.expiration}") long expirationSeconds) {
        this.encoder = encoder;
        this.expirationSeconds = expirationSeconds;
    }

    public String generateAccessToken(String userId, String username, long tokenVersion) {
        Instant now = Instant.now();

        JwtClaimsSet claims = JwtClaimsSet.builder()
                .issuer("ragbag")
                .issuedAt(now)
                .expiresAt(now.plus(15, ChronoUnit.MINUTES))
                .subject(userId)
                .claim("username", username)
                .claim("type", TokenType.ACCESS.name())
                .claim("token_version", tokenVersion)
                .claim("roles", new String[] {"USER"})
                .build();

        return encode(claims);
    }

    public String generateRefreshToken(String userId) {
        Instant now = Instant.now();

        JwtClaimsSet claims = JwtClaimsSet.builder()
                .issuer("ragbag")
                .issuedAt(now)
                .expiresAt(now.plus(14, ChronoUnit.DAYS))
                .subject(userId)
                .claim("type", JwtService.TokenType.REFRESH.name())
                .claim("roles", new String[] {"USER"})
                .build();

        return encode(claims);
    }

    private String encode(JwtClaimsSet claims) {
        JwsHeader header = JwsHeader.with(MacAlgorithm.HS256).type("JWT").build();

        return encoder.encode(JwtEncoderParameters.from(header, claims)).getTokenValue();
    }
}
