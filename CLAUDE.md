# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Ragbag is a self-hosted, Discord-like bookmark manager: a Go server (`cmd/ragbag`) that serves a REST API under
`/api` and the embedded Svelte frontend (`ui/v1`) under `/app`. It runs on SQLite (default) or PostgreSQL.

## Commands

`make help` lists everything. The important ones:

```bash
make dev            # server on :8080 with the settings from .env.local (demo account admin/test123)
make dev-ui         # Vite dev server on :5173 against the server on :8080
make build          # frontend → ui/v1/dist, then bin/ragbag with the frontend embedded
make test           # go test ./... (SQLite)
make test-postgres  # the same tests against a throwaway PostgreSQL container
make check          # go vet, gofmt and the frontend type check (svelte-check)
make generate       # regenerate Go server interfaces + TypeScript client from openapi/openapi.yaml
make docker         # container image ragbag:latest
make install        # build and install/update as a systemd service (Linux, sudo)
make build-linux    # bin/ragbag-linux-{amd64,arm64} to copy to a server
```

### systemd
`scripts/install.sh` installs or updates the binary as a hardened systemd service (`DynamicUser`, state in
`/var/lib/ragbag`, configuration in `/etc/ragbag/ragbag.env` with a generated `JWT_SECRET`, kept on updates);
`--dir /opt/ragbag` (or `make install DIR=/opt/ragbag`) chooses the binary location, which later updates keep;
`--uninstall [--purge]` removes it. Only the `APP`/`DESCRIPTION`/defaults lines at the top are app specific.

A single test: `go test ./internal/server -run TestPrivateCollections`.

### Docker
```bash
docker compose up -d --build                                   # single container with SQLite in a volume
docker compose -f docker-compose.postgres.yaml up -d --build   # app + PostgreSQL 18
```
Configuration is via environment variables (`internal/config`, documented in `.env.example`; for local
development the server also reads `.env.local` and `.env` from the working directory, real variables win): `DATABASE_URL`
(`sqlite:<path>` or `postgres://…`), `JWT_SECRET` (required unless `RAGBAG_DEV=true`), `ADMIN_USERS`,
`REGISTRATION_ENABLED`, `ALLOW_PRIVATE_HOSTS`, `SECURE_COOKIE`, `CORS_ALLOWED_ORIGINS`. Values that still look like
`${…}` (podman-compose not interpolating an empty default) are ignored with a warning. `docker/entrypoint.sh` starts
as root only to chown a bind-mounted `/data` to the `ragbag` user (uid 1000), then drops privileges with su-exec.

## Architecture

### API-first
`openapi/openapi.yaml` is the single source of truth. `make generate` produces:
- `internal/api/api.gen.go` (oapi-codegen, `go tool` dependency): models and the Echo `ServerInterface`, which
  `internal/server.Server` implements; CI fails if it is out of date
- `ui/v1/src/lib/sdk/` (openapi-generator typescript-fetch via a container), imported as
  `ragbag-frontend-sdk` through a Vite/tsconfig alias

When changing the API: edit the spec, `make generate`, implement the new method in `internal/server`.

### Packages
- `cmd/ragbag` — main: config, database, demo seed (dev), HTTP server with graceful shutdown
- `internal/config` — environment configuration
- `internal/db` — opens SQLite (pure Go `modernc.org/sqlite`, `CGO_ENABLED=0`) or PostgreSQL (pgx) and runs goose
  migrations from `migrations/sqlite` and `migrations/postgres`. Refuses databases of the former Java version
  (Liquibase table present) with a clear error
- `internal/store` — all SQL, via sqlx. Queries are written once with `?` placeholders (rebound for PostgreSQL) and
  must stay portable: no dialect specific functions, `ON CONFLICT DO NOTHING`, `LIKE … ESCAPE '\'`. `InTx` runs a
  transaction, `DryRun` one that is always rolled back (import preview)
- `internal/server` — HTTP handlers and business rules, one file per domain (auth, users, spaces, collections,
  bookmarks, dashboard, invites, exchange). Errors are RFC 9457 problem details (`badRequest`, `forbidden`,
  `notFound`, `conflict`); the frontend shows `detail`
- `internal/auth` — JWTs (HS256) and BCrypt passwords (8 characters to 72 bytes)
- `internal/fetch` — SSRF-guarded fetching of user supplied URLs (every redirect and the dialed IP are checked;
  loopback/link-local always refused, private networks unless `ALLOW_PRIVATE_HOSTS`) and the page metadata parser
- `internal/images` — avatars and space icons: content sniffing, decompression bomb check, center crop, 256 px PNG
- `ui` — embeds the frontend build `ui/v1/dist` (only `.gitkeep` is committed, so the package compiles before
  the first build). `v1` leaves room to develop a new UI generation next to the current one

### Domain rules (keep them in sync with the tests in `internal/server/*_test.go`)
- **Roles** OWNER > ADMIN > CONTRIBUTOR > VIEWER map to permissions in `permissions.go` (single source of truth,
  also sent to the frontend). Every handler taking a resource id starts with `requireSpace`, `requireCollection`
  or `requireBookmark`.
- **Private collections** (`restricted`) are visible only to their members, whatever the space role, and hide
  everything below them (`hiddenCollections`). Hidden collections answer 404, and every list or query of bookmarks
  must exclude them. Parents with hidden children cannot be deleted; leaving a space deletes private collections
  nobody can see anymore.
- **Discoverable spaces** (`discoverable`, `join_role` VIEWER or CONTRIBUTOR) are listed by `GET /spaces/discover`
  for every account and joined via `POST /spaces/{id}/join`; other spaces answer 404 there. Joining never changes
  an existing membership, and changing the setting needs `MANAGE_MEMBERS`.
- **Auth**: access tokens (`type=ACCESS`) in the `Authorization` header, refresh tokens (`type=REFRESH`) in an
  httpOnly cookie scoped to `/api/auth/refresh`, `Secure` automatically for HTTPS incl. `X-Forwarded-Proto`. Both
  carry `token_version`; logout everywhere, password changes and resets increment it.
- **Instance admins** are configured by username (`ADMIN_USERS`), never stored, and may list users and create
  one-time password reset links (only SHA-256 hashes stored, 24 h, one per user).
- **Export/import** (`exchange.go`): versioned JSON (`format: ragbag-collection`, `format_version`). Raise the
  version on incompatible changes and convert older files in `parseExchange`. The format is compatible with
  exports of the former Java version, which is the migration path to this one.

### Database migrations
Fresh schema per dialect in `internal/db/migrations/{sqlite,postgres}`; add a new numbered file to both for every
change, never edit an applied one. Times are stored in UTC, truncated to microseconds; ids are UUID strings.

### Frontend Structure (`ui/v1/src/lib`)
- `pages/` — one component per route (routes in `router.ts`), `layouts/AuthLayout.svelte` is the Discord-like shell (space bar, sidebar, quick switcher)
- `components/` — shared UI (`BookmarkCard`, modals, `QuickSwitcher`, collection tree)
- `stores/spaces.svelte.ts` — shared reactive session state (spaces, current user, last visited space)
- `stores/preferences.svelte.ts` — personal display settings (link view: cards / compact / list), stored in localStorage per user and device
- `utils/format.ts` — formatting helpers; always render bookmark URLs through `safeUrl()`; show users and spaces with `Avatar.svelte`
- Links: build every `href` with `p()` from `router.ts`. Hardcoded `/space/...` strings miss the `/app` base and turn into full page loads when served by the backend
- `api/client.ts` — auth handling, `errorMessage()` to surface problem detail texts
- `sdk/` — generated, do not edit (see API-first)
