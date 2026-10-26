# syntax=docker/dockerfile:1

# --- Build: OpenAPI SDKs, Svelte frontend and Spring Boot jar ---------------------------------
FROM eclipse-temurin:25-jdk AS build
WORKDIR /build

COPY .mvn .mvn
COPY mvnw pom.xml ./
COPY openapi openapi
COPY ragbag-server-sdk ragbag-server-sdk
COPY ragbag-frontend-sdk ragbag-frontend-sdk
COPY ragbag-frontend ragbag-frontend
COPY ragbag-server ragbag-server

# Tests run in CI; the cache mounts keep Maven and npm downloads between builds
RUN --mount=type=cache,target=/root/.m2 \
    --mount=type=cache,target=/root/.npm \
    ./mvnw -B -ntp package -DskipTests \
    && cp ragbag-server/target/ragbag-server-*.jar /ragbag.jar

# --- Runtime ---------------------------------------------------------------------------------
FROM eclipse-temurin:25-jre-alpine

RUN addgroup -S ragbag && adduser -S -G ragbag ragbag
WORKDIR /app
COPY --from=build /ragbag.jar /app/ragbag.jar
USER ragbag

ENV SPRING_PROFILES_ACTIVE=prod \
    JAVA_TOOL_OPTIONS="-XX:MaxRAMPercentage=75"
EXPOSE 8080

# Public endpoint, so the check works without the actuator API key
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/api/app/config/features || exit 1

ENTRYPOINT ["java", "-jar", "/app/ragbag.jar"]
