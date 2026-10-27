# syntax=docker/dockerfile:1

# --- Frontend ---------------------------------------------------------------------------------
FROM docker.io/library/node:24-alpine AS ui
WORKDIR /build/ragbag-frontend
COPY ragbag-frontend/package.json ragbag-frontend/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY ragbag-frontend ./
RUN mkdir -p ../web/dist && VITE_APP_BASE=/app npm run build

# --- Server: a static binary with the frontend embedded ---------------------------------------
FROM docker.io/library/golang:1.27-alpine AS server
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd cmd
COPY internal internal
COPY web/embed.go web/
COPY --from=ui /build/web/dist web/dist
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /ragbag ./cmd/ragbag

# --- Runtime ----------------------------------------------------------------------------------
FROM docker.io/library/alpine:3.22

# Fixed ids so a bind-mounted data directory can also be prepared on the host (chown 1000:1000 data)
RUN apk add --no-cache su-exec ca-certificates tzdata \
    && addgroup -S -g 1000 ragbag && adduser -S -u 1000 -G ragbag ragbag \
    && mkdir /data && chown ragbag:ragbag /data
COPY --from=server /ragbag /usr/local/bin/ragbag
COPY --chmod=755 docker/entrypoint.sh /entrypoint.sh

# SQLite in /data by default; set DATABASE_URL=postgres://... to use PostgreSQL instead
ENV DATABASE_URL=sqlite:/data/ragbag.db \
    RAGBAG_ADDR=:8080
VOLUME /data
EXPOSE 8080

# Public endpoint, so the check works without a token
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/api/app/config/features || exit 1

# Starts as root only to fix the ownership of /data, then runs the server as the ragbag user
ENTRYPOINT ["/entrypoint.sh"]
CMD ["ragbag"]
