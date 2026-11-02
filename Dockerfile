# syntax=docker/dockerfile:1

# --- Frontend ---------------------------------------------------------------------------------
FROM docker.io/library/node:26-alpine AS ui
WORKDIR /build/ui/v1
COPY ui/v1/package.json ui/v1/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY ui/v1 ./
RUN VITE_APP_BASE=/app npm run build

# --- Version: the build argument, else git describe of the checkout (docker compose passes none) ---
FROM docker.io/library/alpine:3.24 AS version
RUN apk add --no-cache git
ARG VERSION=
# The context is only mounted for this step, .git never ends up in an image; without it the version is "dev"
RUN --mount=type=bind,target=/src \
    echo "${VERSION:-$(git -c safe.directory='*' -C /src describe --tags --always 2>/dev/null || echo dev)}" > /version

# --- Server: a static binary with the frontend embedded ---------------------------------------
FROM docker.io/library/golang:1.27-alpine AS server
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd cmd
COPY internal internal
COPY ui/ui.go ui/
COPY --from=ui /build/ui/v1/dist ui/v1/dist
COPY --from=version /version /version
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(cat /version)" -o /ragbag ./cmd/ragbag

# --- Binary only: builds bin/ragbag without Go, Node or make on the host -------------------------
#   podman build --target binary --output bin .     (docker: docker build --target binary --output bin .)
FROM scratch AS binary
COPY --from=server /ragbag /ragbag

# --- Runtime ----------------------------------------------------------------------------------
FROM docker.io/library/alpine:3.24

# Fixed ids so a bind-mounted data directory can also be prepared on the host (chown 1000:1000 data)
RUN apk add --no-cache su-exec ca-certificates tzdata \
    && addgroup -S -g 1000 ragbag && adduser -S -u 1000 -G ragbag ragbag \
    && mkdir /data && chown ragbag:ragbag /data
COPY --from=server /ragbag /usr/local/bin/ragbag
COPY --chmod=755 docker/entrypoint.sh /entrypoint.sh

# SQLite in /data by default; set DATABASE_URL=postgres://... to use PostgreSQL instead
# (the first start of an empty instance puts the admin's initial password into /data as well)
ENV DATABASE_URL=sqlite:/data/ragbag.db \
    DATA_DIR=/data \
    RAGBAG_ADDR=:8080
VOLUME /data
EXPOSE 8080

# Public, checks the database too (also for monitoring: GET /api/health)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/api/health || exit 1

# Starts as root only to fix the ownership of /data, then runs the server as the ragbag user
ENTRYPOINT ["/entrypoint.sh"]
CMD ["ragbag"]
