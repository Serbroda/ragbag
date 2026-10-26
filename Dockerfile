# syntax=docker/dockerfile:1

# --- Build: OpenAPI SDKs, Svelte frontend and Spring Boot jar ---------------------------------
FROM docker.io/library/eclipse-temurin:25-jdk AS build
WORKDIR /build
# Leave memory for the Node/Vite frontend build on small hosts (default heap is 1/4 of RAM per JVM)
ENV MAVEN_OPTS="-Xmx768m"

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
FROM docker.io/library/eclipse-temurin:25-jre-alpine

# Fixed ids so a bind-mounted data directory can also be prepared on the host (chown 1000:1000 data)
RUN apk add --no-cache su-exec \
    && addgroup -S -g 1000 ragbag && adduser -S -u 1000 -G ragbag ragbag \
    && mkdir /data && chown ragbag:ragbag /data
WORKDIR /app
COPY --from=build /ragbag.jar /app/ragbag.jar
COPY --chmod=755 docker/entrypoint.sh /entrypoint.sh

# SQLite in /data by default; set SPRING_DATASOURCE_URL to use PostgreSQL or MariaDB instead.
# foreign_keys enables cascades, WAL lets reads run alongside writes, busy_timeout waits for locks.
ENV SPRING_PROFILES_ACTIVE=prod \
    SPRING_DATASOURCE_URL="jdbc:sqlite:/data/ragbag.db?foreign_keys=on&journal_mode=WAL&busy_timeout=5000&date_class=TEXT" \
    JAVA_TOOL_OPTIONS="-XX:MaxRAMPercentage=75"
VOLUME /data
EXPOSE 8080

# Public endpoint, so the check works without the actuator API key
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/api/app/config/features || exit 1

# Starts as root only to fix the ownership of /data, then runs the app as the ragbag user
ENTRYPOINT ["/entrypoint.sh"]
CMD ["java", "-jar", "/app/ragbag.jar"]
