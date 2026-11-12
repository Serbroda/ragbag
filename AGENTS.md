# AGENTS.md

This file provides guidance to coding agents (Claude Code, Codex & co) when working with code in this repository.

Ragbag is a self-hosted, Discord-like bookmark manager: a Go server (`cmd/ragbag`) that serves a REST API under
`/api` and the embedded Svelte frontend (`ui/v1`) under `/app`. It runs on SQLite (default) or PostgreSQL.

## Commands

`make help` lists everything. The important ones:

```bash
make dev            # server on :8080 with the settings from .env.local (an empty database starts with the setup, like in production)
make dev-ui         # Vite dev server on :5173 against the server on :8080
make build          # frontend → ui/v1/dist, then bin/ragbag with the frontend embedded
make test           # go test ./... (SQLite)
make test-postgres  # the same tests against a throwaway PostgreSQL container
make check          # go vet, gofmt and the frontend type check (svelte-check)
make generate       # regenerate Go server interfaces + TypeScript client from openapi/openapi.yaml
make docker         # container image ragbag:latest
make install        # build and install/update as a systemd service (Linux, sudo)
make build-linux    # bin/ragbag-linux-{amd64,arm64} to copy to a server
make release-patch  # tag the next version on main (also release-minor/-major); push with git push origin main vX.Y.Z
```

### Versions
Git tags (`vX.Y.Z`) are the only place the version lives: builds take `git describe --tags --always --dirty`
(e.g. `v1.2.0-3-gabc1234` after a release) into `main.version`, the server reports it in
`GET /api/app/config/features` and the frontend shows it in the account menu. Container builds take the `VERSION`
build argument (`make docker`, `install.sh --build`), else run `git describe` on the mounted context themselves
(`docker compose up --build`), so `.git` is not in `.dockerignore`. The frontend offers a reload when the server
reports another version than the one it was loaded with. `scripts/release.sh` (behind the
`release-*` targets) only tags clean `main`; pushing stays a separate step.

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
(`sqlite:<path>` or `postgres://…`), `JWT_SECRET` (required, also locally), `ADMIN_USERS`,
`REGISTRATION_ENABLED`, `RATE_LIMIT_ENABLED`, `QUOTA_*`, `PLANS_ENABLED`, `PLANS_FREE_API_TOKENS`, `SUPPORT_URL`, `SMTP_*`, `PUBLIC_URL`, `DATA_DIR`, `IMPRINT_URL`/`PRIVACY_URL`/`TERMS_URL`, `BACKUP_*`, `LOG_MAX_AGE_DAYS`, `LOG_ANONYMIZE_IP`, `ALLOW_PRIVATE_HOSTS`, `SECURE_COOKIE`, `CORS_ALLOWED_ORIGINS`, `LOG_LEVEL`, `LOG_FILE`
(rotating, `internal/logging`), `LOG_FORMAT`. Values that still look like
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
- `cmd/ragbag` — main: config, database, setup admin, HTTP server with graceful shutdown;
  `ragbag backup|restore|reset-2fa` commands (`commands.go`, only need `DATABASE_URL`)
- `internal/backup` — backup and restore of the whole instance, independent of the database
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
- `internal/logging` — slog setup: stdout or a rotating file (lumberjack), text or JSON. `server/logging.go` logs
  one line per request (secrets in paths masked, no query strings) and `audit()` the events worth troubleshooting
  (logins, failed logins as WARN, memberships, deletions); never log tokens, passwords or PINs
- `internal/auth` — JWTs (HS256) and BCrypt passwords (8 characters to 72 bytes)
- `internal/fetch` — SSRF-guarded fetching of user supplied URLs (every redirect and the dialed IP are checked;
  loopback/link-local always refused, private networks unless `ALLOW_PRIVATE_HOSTS`) and the page metadata parser
- `internal/images` — avatars and space icons (`Normalize`: center crop, 256 px PNG) and link previews
  (`NormalizePreview`): content sniffing, decompression bomb check, small GIF/WebP kept as they are.
  `ImagePicker.svelte` applies a change right away when given `onupload`/`onremove` (avatar), otherwise it keeps it
  in `bind:change` until the surrounding form saves (space settings, creating a space).
  Random space icons are DiceBear's "Planets" (CC0 1.0, `utils/planet.ts`), generated in the browser (loaded on
  demand, never the DiceBear API) and uploaded as PNG like any other icon; new spaces get one unless removed
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
- **Registration with invites** (`auth.go`): `POST /auth/register` takes an optional `inviteToken` (a space
  invite); the account joins that space with its first space, with email only once confirmed
  (`email_verifications.invite_id`, kept when a new link is sent; a revoked invite joins nothing). The use is
  counted at registration. While `REGISTRATION_ENABLED` is off, only invites of instance admins register
  (`inviteRegisters`). Signed-out visitors land on the login page with the remembered invite, which previews it
  via the public `GET /auth/invites/{token}` (`registration`).
- **Welcome** (`spaces.go`, `WelcomeModal.svelte`): `space_members.welcomed_at` is null until the member first
  opens the space (`POST /spaces/{id}/welcomed`, the layout calls it when showing the welcome). The owner of a new
  account's first space gets an introduction and a random planet as icon (the server cannot draw them); spaces
  created by hand count as welcomed. Other members see the space's `welcome_message` (Markdown by members with
  `EDIT_SPACE`, rendered by goldmark without raw HTML or dangerous links into `welcomeHtml`), later via the space menu.
- **Display names** (`users.display_name`, `PUT /users/me/display-name`, optional at registration): 1–50
  characters, any case, not unique; the username stays the login and identity (logs, audit, admin dialogs). Fields
  that only name somebody (link author, reactions, news, inviter, reports, trash) carry the display name, else the
  username (`shownName` in SQL, `User.Name()`, `shownName()` in the frontend); lists of people (members, admin)
  carry both and show `@username` next to a display name. Deleting an account clears it.
- **Discoverable spaces** (`discoverable`, `join_role` VIEWER or CONTRIBUTOR) are listed by `GET /spaces/discover`
  for every account and joined via `POST /spaces/{id}/join`; other spaces answer 404 there. Joining never changes
  an existing membership, and changing the setting needs `MANAGE_MEMBERS`.
- **Collection order** (`position`, per parent) is shared by all members; `POST /collections/{id}/move` takes the
  new parent and optionally `beforeId` (a collection below that parent), otherwise the collection goes to the end.
  In the sidebar the upper/lower quarter of a row drops in front of/behind it, the middle into it.
- **Links** move between collections of the same space only (`collectionId` in the update, needs
  `CREATE_BOOKMARKS` on the target). A search matches every term (`store.SearchTerms`: words, `"phrases"`, at
  most 8); a `#tag` term matches that tag exactly.
- **Smart collections** (`smart.go`, `collections.query`, set when creating, only the search changes later): no
  links of their own, `GET /collections/{id}/bookmarks` answers with the search over the whole space instead
  (so `hiddenCollections` applies per member). Everything that would put something into one answers 400
  (`holdsLinks`: creating or moving links and collections, imports); they cannot be shared and are left out of
  public links of a collection above them. Exports carry the `query`. Private, locked, trash and order work
  like for any collection. In the frontend pick targets with `flattenTargets()`, never `flattenTree()`.
- **New links** (`activity.go`) are links *others* added after the user last looked at the collection
  (`collection_visits`, before the first visit the time they joined the space). They feed `newCount` in the space
  list and the collection tree, `isNew` when listing a collection and `GET /spaces/{id}/news` (the bell in the header); hidden collections
  never count. Listing does not mark anything as seen, `POST /collections/{id}/seen` does (the frontend calls it
  after showing new links).
- **Reactions** come from a fixed set of emojis (`reactionEmojis`, mirrored in `Reactions.svelte`); everybody who
  can read a link may react. There are deliberately no comments (nothing to moderate).
- **Space bar** order and folders are personal (`space_bars`, JSON of `SpaceBarDto`); the server drops spaces the
  user is no longer a member of. Leaving a space removes the user's reactions and visits there.
- **Public links** (`shares.go`, one per collection) show a collection and its sub-collections to anybody, without
  an account, at `/app/share/{token}` (`GET /api/v1/shares/{token}`, a public path). They never show the space,
  members, authors, reactions or private collections; private collections cannot be shared, and a shared one that
  becomes private (or moves below one) answers 404 until it is public again. Every failure is the same 404.
  Managing the link needs `EDIT_COLLECTIONS`; only those see the token.
- **Trash** (`trash.go`): deleting only sets `deleted_at` on a link or collection. Trashed collections are part of
  `hiddenCollections` (with everything below), trashed links are filtered by `bookmarkJoins`, so every query leaves
  them out without extra code; queries that do not use those joins must filter `deleted_at IS NULL` themselves.
  The trash lists what the user could see (`hiddenCollectionsOf(…, withTrash=false)`), restoring brings back
  favorites, reactions and pins, `PurgeOldTrash` (hourly from main) deletes after 30 days.
- **Preview images**: shown is an upload (`image_id`, `images.NormalizePreview`: scaled to 1200 px, aspect kept,
  JPEG unless transparent) over an own address (`image_url`) over the page's `og_image`, none if `image_hidden`
  (`previewImage()` in the frontend, the same order for public shares). Addresses are loaded by the browser,
  never cached. Links that point straight to an image become their own preview (`fetch.isImage`, by content
  type or first bytes). Unused uploads are deleted by the hourly job (`DeleteOrphanImages`).
- **Quick saving**: `/app/add?url=&title=&text=` (route outside a space) is the target of the web app manifest's
  `share_target` (installed app on Android) and of the bookmarklet in the settings (*Apps & API*); signed-out visitors
  return there after the login (`rememberPendingAdd`).
- **Offline** (`ui/v1/public/sw.js`): the service worker only answers failed page loads with a notice built into
  it. It deliberately caches nothing (no private links on the device, no stale versions) and leaves `/api` alone.
- **Headlines** (`headlines.go`, `fetch/feed.go`, `Headlines.svelte`): a link may have a news feed (`feed_url`,
  RSS or Atom) and a filter (`feed_filter`, comma separated, any word in a headline or its categories). Saving
  takes a feed or a page linking to one (`FindFeed`, 400 if none); an unchanged address is not loaded again, a left
  out one stays. The cards load `GET /bookmarks/{id}/headlines` (public links: `/shares/{token}/bookmarks/{id}/headlines`,
  same checks as `sharedCollection`) when shown; the server keeps each feed in memory for an hour (failures 10
  minutes), shared by all users, and only a load counts against the fetch limit. Nothing of a feed is stored.
  The dashboard section `HEADLINES` merges the feeds of picked links (none picked: every link of the space with a
  feed, `allFeeds`) via `GET /spaces/{id}/headlines?bookmarkId=…`, newest first, each article once.
- **Link previews** (`meta.go`): the server writes title, description and Open Graph tags into `index.html` per
  page (chat apps do not run JavaScript). Shared collections go through the same `sharedCollection` as the API,
  so private, revoked or unknown links get the generic preview; pages behind a login never reveal anything.
  Usable invites name inviter and space (whoever has the link sees that on the invite page too). Addresses use
  `PUBLIC_URL`, else the request's host. The image is `ui/v1/public/og-image.png` (1200×630).
- **Locked collections** (`locks.go`) are personal: a user hides a collection (and everything below) from their
  own view; nobody else is affected. Locking needs a PIN (4–8 digits, BCrypt hash, set with the password; 5 wrong
  PINs pause unlocking for 5 minutes). Until unlocked they are part of `hiddenCollections`, so every list, search,
  counter and dashboard leaves them out. `POST /users/me/unlock` returns a 10 minute token (`type=UNLOCK`, carries
  `token_version`) for the `X-Ragbag-Unlock` header; the frontend (`stores/lock.svelte.ts`) renews it on activity,
  keeps it in sessionStorage and drops it to lock (account menu, ⌘⇧L, the "Unlocked" pill).
- **Rate limits** (`ratelimit.go`, `RATE_LIMIT_ENABLED`, off in tests unless `withRateLimits`): token buckets in
  memory per IP (all of `/api`, login, register, password reset), per account (failed logins, by user id so
  username and email count together) and per user (password change, loading user supplied URLs; creating or
  editing a link then skips the details instead of failing). The limits are listed in the README.
- **Quotas** (`quotas.go`, `config.Quotas`, 0 = unlimited, the default): links per user (`created_by`, trash
  included), spaces owned per user (not the first one at registration), collections per space, uploaded image
  bytes per user (`images.created_by`). Checked before creating, or at the end of the transaction with
  `adding = 0` (imports, replaced images); 403 "Limit reached".
- **Plans** (`plans.go`, `PLANS_ENABLED`, off by default for self-hosting): Free gets `config.Quotas`, Pro
  `Plans.Pro` (`quotasOf`, `userQuotas`; collections by the space's owners, `spaceQuotas`). Pro (`proSource`) are
  instance admins and `users.pro_granted_at` (admins, `PUT/DELETE /admin/users/{id}/pro`); nothing is sold.
  `PlanPage.svelte` at `/app/plan` (account menu). `SUPPORT_URL` (feature flag `support_url`) links a donation
  page next to the search (`SupportButton.svelte`, closing it is kept in localStorage) and at the bottom of the
  account menu, deliberately not on the plan page: donations unlock nothing.
- **Email** (`internal/mail`, `email.go`, only with `SMTP_HOST`; tests use `UseMailer` with a fake): new
  accounts get `email_verified_at` only by `POST /auth/verify-email/{token}` (48 h, hash in
  `email_verifications`), which also creates their first space and signs them in; until then login answers 403
  and resends the link. Unconfirmed accounts are deleted after 7 days (hourly job). `POST /auth/password-reset`
  mails a 1 h link in `password_resets` and always answers 204. Mails go out in the background, at most 3 per
  account per hour (`Server.mails`, also without rate limits); links use `PUBLIC_URL`, never the request's host. Without SMTP accounts are
  confirmed at once, waiting ones on their next login.
- **Operations** (`operations.go`): `ScheduledBackups` (from main) writes `backup.Dir` files every
  `BACKUP_INTERVAL_HOURS` when `BACKUP_KEEP` > 0, next due an interval after the newest file, and prunes to
  `BACKUP_KEEP`; admins list, download and trigger them (`/admin/backups`). `GET /api/health` pings the database.
  `LOG_ANONYMIZE_IP` shortens the IP in request and audit lines (`loggedIP`), main rotates `LOG_FILE` daily so
  `LOG_MAX_AGE_DAYS` holds.
- **Legal pages** (`legal.go`): `DATA_DIR/legal/{imprint,privacy,terms}.md` rendered by goldmark without raw
  HTML (`GET /app/legal/{page}`, public), or `IMPRINT_URL` & co.; the feature flags `legal_*` say where they are
  (`utils/legal.ts`, `LegalLinks.svelte`). With terms, `POST /auth/register` needs `acceptTerms`
  (`users.terms_accepted_at`). German templates in `docs/legal`.
- **Changed terms** (`terms.go`, `terms_changes`): instance admins record a change on `/admin` (`POST /admin/terms`,
  today or announced for a later day, `DELETE /admin/terms/next` withdraws it). From `effective_at`, accounts whose
  `terms_accepted_at` is older (or null) get 403 with `X-Ragbag-Terms` on everything but `allowedPendingTerms`
  (own account, `POST /users/me/terms`, deleting the account), API tokens included; the frontend opens `/terms`
  (`TermsPage.svelte`). Announced changes show `TermsNotice.svelte` (`UserDto.termsChangeAt`).
- **Security headers** (`headers.go`): a strict CSP for the app (scripts/styles from `'self'`, images from
  anywhere, no frames) and `default-src 'none'` plus `no-store` for `/api`; HSTS only over HTTPS. No inline
  scripts in `index.html` or `meta.go` (the CSP would block them); handlers may override (images: sandbox).
- **Auth**: access tokens (`type=ACCESS`) in the `Authorization` header, refresh tokens (`type=REFRESH`) in an
  httpOnly cookie scoped to `/api/auth/refresh`, `Secure` automatically for HTTPS incl. `X-Forwarded-Proto`. Both
  carry `token_version`; logout everywhere, password changes and resets increment it.
- **Two-factor authentication** (`twofactor.go`, `auth/totp.go`, `TwoFactorSettings.svelte`): TOTP, off unless
  the user turns it on (`POST /users/me/two-factor/setup` with the password → secret and `otpauth://` address,
  `…/enable` with the first code → 10 recovery codes shown once, `…/disable` with password and a code). `POST
  /auth/login` then takes `code` (app or recovery code): without or with a wrong one it answers 401 with
  `X-Ragbag-Two-Factor: required`, only after the password was right. A code works once (`totp_last_step`), 5
  wrong ones pause checking for 5 minutes (like PINs, also without rate limits). The secret is stored encrypted
  with a key derived from `JWT_SECRET` (`auth.SecretBox`), recovery codes as SHA-256 hashes; with another
  `JWT_SECRET` only recovery codes work. Reset links sign nobody in, so they never bypass it; admins reset it
  (`DELETE /admin/users/{id}/two-factor`), `ragbag reset-2fa <username>` is the way back in for an admin. The QR
  code is drawn in the browser (`QrCode.svelte`, `qrcode-generator` loaded on demand).
- **Discord webhook** (`webhooks.go`, `spaces.discord_webhook_url`, `DiscordWebhookSettings.svelte`): one per
  space, set by members with `EDIT_SPACE` (`/spaces/{id}/webhook`, `…/test`); `CreateBookmark` calls
  `announceLink`, which posts an embed in the background (its "Collection" field links to the collection in
  Ragbag: `PUBLIC_URL`, else the request's host; embed footers cannot hold links). Never posted: links in or below private collections
  (`privateInPath`) and imports (they do not go through `CreateBookmark`). Only `https://discord.com/api/webhooks/…`
  addresses are accepted and redirects are not followed, so it is no way to make the server call other hosts.
  The address is a secret: not in `SpaceDto`, not for API tokens (`tokenNever`), never logged (not even errors
  of the HTTP client). Text of users is escaped and `allowed_mentions` is empty. A 404/401 from Discord sets
  `discord_webhook_failed_at`, which stops posting until a new address or a working test. Tests use
  `UseWebhookSender` with a fake
- **API tokens** (`tokens.go`, `api_tokens`, `ApiTokenSettings.svelte`): personal tokens `rb_…` (SHA-256 hash
  stored, shown once, created with the password at `/users/me/tokens`, at most 20, optional expiry) are accepted
  by `authenticate` like access tokens, but only for what `tokenMay` allows: scope READ = GET, WRITE = everything,
  both only below `/v1/spaces`, `/v1/collections`, `/v1/bookmarks` (plus `GET /users/me`) and without the paths
  that manage a space or decide who sees what (`tokenNever`, `tokenNoWrite`). New endpoints elsewhere are closed
  to tokens by default; add management endpoints below those prefixes to the lists. Tokens never unlock locked
  collections and are not revoked by password changes (blocked/deleted accounts lose them). With plans they are
  part of Pro unless `PLANS_FREE_API_TOKENS` (`apiTokensFor`, sent as `UserDto.apiTokens`): Free accounts get 403
  when creating one, and existing tokens of an account that is Free again answer 403 until it is Pro again.
- **Instance admins** are configured by username (`ADMIN_USERS`), never stored, and may list users, create
  one-time password reset links (only SHA-256 hashes stored, 24 h, one per user) and download backups. The one
  exception: while `ADMIN_USERS` is empty, the setup admin (`users.setup_admin`) is the admin. Everything they do lives on
  `/admin/:tab` (`AdminPage.svelte`: overview from `GET /admin/overview` with counts and the effective
  configuration, accounts, reports, backup), outside the space layout.
- **Moderation** (`moderation.go`): deleting an account (`POST /users/me/delete`, password; admins
  `DELETE /admin/users/{id}`) deletes the spaces it alone is a member of, leaves the others (`RemoveMember`) and
  anonymizes the row (`AnonymizeUser`: `deleted-…`, no email, password or avatar, `deleted_at`), so `created_by`
  stays valid. Only owners of spaces with others answer 409; admins hand those over (`NextOwner`). Blocked
  accounts (`blocked_at`, reason shown at login) get 401 on every token and their public links answer 404, like
  links disabled after a report (`collection_shares.disabled_at`, kept when revoked so the space cannot share
  again). Reports (`abuse_reports`, public `POST /shares/{token}/reports`, rate limited) are resolved by instance
  admins (disable, dismiss, enable); with email both sides get the decision. Resolved reports go after 180 days.
- **Setup** (`setup.go`): `Bootstrap` (from main) creates the setup admin when there are no accounts, named after
  the first of `ADMIN_USERS` (else `admin`), with a random password in `DATA_DIR/initial-admin-password` (0600,
  never logged). Until `POST /auth/setup` (or a password change) replaces it, `must_change_password` makes every
  other request answer 403 with `X-Ragbag-Setup` (the frontend opens `/setup`), except what a restore needs
  (`allowedDuringSetup`): the wizard asks first, and only starting fresh sets a password. Then the file is deleted.
- **Backups** (`internal/backup`, `GET /admin/backup`, `POST /admin/backup/restore`, `ragbag backup|restore`):
  every table as gzip JSON (format `ragbag-instance`) with the ids, values normalized (RFC 3339 times, base64
  blobs, booleans converted for the target), so SQLite and PostgreSQL restore each other. Tables are listed in
  foreign key order in `backup.tables` (a test fails for tables neither there nor in `Skipped`; self references
  and cycles via `parent`/`deferred`). Restores replace everything in one transaction and only on an empty
  instance (no spaces, no accounts besides the setup admin); older backups go through a temporary SQLite
  database at their `schema_version` and are migrated there first. A new migration needs no backup code unless
  it adds a table or a reference cycle.
- **Export/import** (`exchange.go`): versioned JSON (`format: ragbag-collection`, `format_version`). Raise the
  version on incompatible changes and convert older files in `parseExchange`. The format is compatible with
  exports of the former Java version, which is the migration path to this one. Sent as `text/html` the import
  takes a browser's bookmarks file instead (`exchange_html.go`, Netscape format read token by token): it is
  converted to the same `exchangeFile`, so limits, merging, quotas and the preview are shared.

### Database migrations
Fresh schema per dialect in `internal/db/migrations/{sqlite,postgres}`; add a new numbered file to both for every
change, never edit an applied one. Times are stored in UTC, truncated to microseconds; ids are UUID strings.

### Frontend Structure (`ui/v1/src/lib`)
- `pages/` — one component per route (routes in `router.ts`), `layouts/AuthLayout.svelte` is the Discord-like shell (space bar, sidebar, quick switcher)
- `components/` — shared UI (`BookmarkCard`, modals, `QuickSwitcher`, collection tree)
- `stores/spaces.svelte.ts` — shared reactive session state (spaces, current user). The start page opens
  `startSpaceId()`: the chosen start space (`users.start_space_id`, unset by leaving or deleting the space), else
  the last visited one (localStorage per user), else the first in the user's space bar
- `pages/SettingsPage.svelte` — personal settings at `/settings/:tab` (profile, security, appearance, apps & API,
  account), outside the space layout like `/admin` and `/plan`. Every setting is a `SettingsItem.svelte`: a row
  with its current state and one button; its form, and the password some need, only shows after that button, so
  there is never more than one form open per setting and no page full of password fields. New personal settings
  go there, not into dialogs
- `stores/preferences.svelte.ts` — personal display settings (link view: cards / compact / list), stored in localStorage per user and device
- `utils/format.ts` — formatting helpers; always render bookmark URLs through `safeUrl()`; show users and spaces with `Avatar.svelte`
- Wording: sentence case for buttons, titles and labels ("Add link", "New collection"), "…" on buttons that ask
  for more before acting. Fields are darker than their surface (`app.css`), lists of rows get one main button and
  a "⋯" menu with destructive entries in red (`AdminUsers.svelte`), and cards without a preview image get a
  stand-in only next to cards that have one (`imagePlaceholder`). Cards name who added a link only in spaces
  with several members (`SpaceDto.memberCount`, context `solo-space`). In the sidebar a collection keeps its
  icon (folder, or filter for smart ones); locked, private and shared show as a small mark next to the name
  (`TreeNode.mark`); link counts are deliberately not shown there. Space settings are tabs as well (`?tab=members`)
- Unknown addresses end at `NotFoundPage.svelte` (the `*notfound` route; signed-out visitors at the login). `p()`
  with an empty `search: {}` builds a broken link ("…undefined"): leave `search` out instead
- Links: build every `href` with `p()` from `router.ts`. Hardcoded `/space/...` strings miss the `/app` base and turn into full page loads when served by the backend
- `api/client.ts` — auth handling, `errorMessage()` to surface problem detail texts
- `sdk/` — generated, do not edit (see API-first)
