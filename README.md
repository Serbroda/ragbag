# Ragbag – Links for you and your people

Ragbag is a self-hosted place to collect and share links. Like servers in Discord, **spaces** bring people
together: each space has members with roles, a tree of **collections** and the links inside them. Use it as a
shared bookmark list for your family or team, a reading list, a gift list – or as the start page of your homelab.

It is a single Go binary with an embedded **Svelte** frontend, running on **SQLite** or **PostgreSQL**.

---

## 📝 Features

**Spaces & people**
- Spaces with members and roles: **owner**, **admin**, **contributor** (adds and edits links and collections)
  and **viewer**. A space bar like in Discord switches between them, `Ctrl/⌘ K` jumps anywhere.
- Arrange the space bar your way: drag spaces into your own order and group them in **folders**.
- Choose the space Ragbag **opens on start** (profile, or right-click a space); otherwise it reopens the last one.
- **Invite links** with role, expiry and maximum uses – send them via any messenger.
- **Public spaces** show up under *Discover spaces* for everybody on your instance, who can join with one click
  as viewer or contributor.
- Avatars for users and logos for spaces. New spaces start with a **random planet** – roll another one or upload
  your own image.
- **What's new**: a dot on spaces and collections with links others added since your last visit, and a bell with
  the latest additions. React to links with a small set of **emojis** (deliberately no comments).

**Collections & links**
- Nested collections with drag & drop and breadcrumbs; sort them by dragging them in the sidebar.
- **Private collections** are only visible to selected members – not even to space owners and admins. Everything
  inside them (sub-collections, links) is private too, which makes them perfect for gift ideas.
- **Lock collections** for yourself with a PIN: they disappear from your view, search and dashboard until you
  unlock them – handy when somebody looks over your shoulder. Nobody else is affected.
- **Public links** show a collection and its sub-collections to anybody, without an account – read-only, without
  members, authors or private collections. Chat apps like WhatsApp and Signal show a proper preview.
- Links with title, description, tags, favicon and preview image. **Fetch details** loads them from the page,
  suggests tags already used in the space and warns about duplicates. Links that point straight to an image become
  their own preview; set your **own preview image** (upload or address) or hide it.
- Select several links (long press on phones) to move, tag or delete them at once; copy a link from its menu.
- Sort links (newest, oldest, name, domain) and narrow them down with tag chips.
- Full-text search over titles, addresses, descriptions and tags; `#tag` finds exactly that tag.
- Three ways to show links – cards with preview images, compact cards or a dense list – chosen per user and device.

**Personal**
- **Favorites** and a **dashboard** per user and space: arrange sections of hand-picked links, favorites and recently
  added links by drag & drop, and add links right from there. Great as a homelab start page.

**Data**
- **Save from anywhere**: install Ragbag as an app and share links to it (Android), or use the bookmarklet.
- **Trash**: deleted links and collections can be restored for 30 days.
- **Export and import** collections as versioned JSON files, with a preview of what an import will do.

**Accounts**
- Users change their password themselves. Without email, **admins create one-time reset links** for people who
  forgot theirs.
- Registration can be switched off once everybody has an account.
- Users **delete their account** themselves; admins **block or delete accounts** and decide on **reports of public
  links** (see *Moderation*).
- Light and dark mode, works on phones and can be installed as an app.
- The version is shown in the account menu.

---

## 🔧 Installation

### Docker (recommended)

```bash
git clone https://github.com/Serbroda/ragbag.git
cd ragbag
cp .env.example .env        # set JWT_SECRET (openssl rand -base64 32) and ADMIN_USERS
docker compose up -d --build
```

Open <http://localhost:8080>. The database lives in the `ragbag-data` volume.

- **PostgreSQL** instead of SQLite: `docker compose -f docker-compose.postgres.yaml up -d --build`
  (set `DB_PASSWORD` in `.env` as well).
- **Keep the data next to the compose file:** use `./data:/data` instead of the volume (see the comment in
  `docker-compose.yaml`). The container fixes the permissions on start; on SELinux hosts (Fedora, RHEL) append `:Z`.
- **Without compose:**
  ```bash
  docker build -t ragbag .
  docker run -d --name ragbag -p 8080:8080 -v ragbag-data:/data -e JWT_SECRET=... ragbag
  ```

The image is about 30 MB and needs roughly 20 MB of memory.

### As a systemd service (without containers)

`scripts/install.sh` installs the binary as a systemd service: it starts on boot, restarts on failure and runs
as an unprivileged, sandboxed user. Running it again updates the binary and keeps your configuration.

```bash
make build                                   # bin/ragbag with the frontend embedded
sudo ./scripts/install.sh                    # binary → /usr/local/bin/ragbag
sudo ./scripts/install.sh --dir /opt/ragbag  # or → /opt/ragbag/ragbag
```

| What          | Where                                                                  |
|---------------|------------------------------------------------------------------------|
| Configuration | `/etc/ragbag/ragbag.env` (created with a random `JWT_SECRET`)          |
| Data          | `/var/lib/ragbag`                                                      |
| Logs          | `journalctl -u ragbag -f`                                              |

After changing the configuration run `sudo systemctl restart ragbag`. Further options:

```bash
sudo ./scripts/install.sh --binary ./ragbag-linux-amd64 --port 9000   # a prebuilt binary, another port
sudo ./scripts/install.sh --uninstall           # remove the service, keep data and configuration
sudo ./scripts/install.sh --uninstall --purge   # remove everything
make install DIR=/opt/ragbag                    # build and install in one step
```

Updates keep the location chosen with `--dir`.

**No Go, Node or make on the server?** Let the script build the binary in a container – only podman or docker is
needed:

```bash
sudo ./scripts/install.sh --build --dir /opt/ragbag   # after a git pull, the same command updates
```

Alternatively build on your machine and copy the binary: `make build-linux` creates `bin/ragbag-linux-amd64` and
`bin/ragbag-linux-arm64`, or `docker build --target binary --output bin .` builds `bin/ragbag` for your platform.

### From source

Prerequisites: Go 1.27+, Node.js 24 (LTS) and npm.

```bash
make build                   # frontend + bin/ragbag
JWT_SECRET=$(openssl rand -base64 32) ./bin/ragbag
```

### First start

The first start of an empty instance creates an **admin account** – named after the first entry of `ADMIN_USERS`,
`admin` when that is empty – with a random password in a file next to the data (never in the log):

```bash
sudo cat /var/lib/ragbag/initial-admin-password              # systemd (install.sh prints it, too)
docker compose exec app cat /data/initial-admin-password     # Docker
cat data/initial-admin-password                              # from source (or $DATA_DIR)
```

Sign in with it and choose: **start fresh** (then you set your own password and email address) or **restore a
backup** of another Ragbag (see *Moving to another server*), which replaces this account with those of the backup.
Either way the file is deleted. While `ADMIN_USERS` is empty this account is the instance admin; otherwise the
ones listed there are.

Everybody else **registers** – every account gets its own first space. Admins manage all users in the profile menu
(→ *Administration*). Set `REGISTRATION_ENABLED=false` once everybody has an account if your instance is reachable
from the internet.

Database migrations run automatically on start-up.

### Updating

Releases are git tags (`vX.Y.Z`); the account menu shows which one is running. Pull and rebuild the same way you
installed:

```bash
git pull && docker compose up -d --build                  # Docker
git pull && sudo ./scripts/install.sh --build --dir /opt/ragbag   # systemd, built in a container
```

Migrations run on start-up, configuration and data are kept. Back up first (see *Backups*) when an update says it
changes the database.

---

## ⚙️ Configuration

Everything is configured with environment variables. For local runs the server also reads `.env.local` and
`.env` from its working directory; variables that are set in the environment always win.

| Variable                | Description                                                          | Default                 |
|-------------------------|----------------------------------------------------------------------|-------------------------|
| `JWT_SECRET`            | Signs the login tokens, at least 32 characters. **Required**         | –                       |
| `DATABASE_URL`          | `sqlite:<file>` or `postgres://user:password@host:5432/db?sslmode=disable` | `sqlite:data/ragbag.db` (Docker: `/data/ragbag.db`) |
| `RAGBAG_ADDR`           | HTTP address                                                         | `:8080`                 |
| `ADMIN_USERS`           | Comma-separated usernames that may manage all users and download backups; the first start creates the first of them | – (the account created on the first start) |
| `DATA_DIR`              | Where the first start puts `initial-admin-password`                  | the SQLite file's directory, systemd's state directory, else `data` (Docker: `/data`) |
| `REGISTRATION_ENABLED`  | Whether new accounts can be registered                               | `true`                  |
| `RATE_LIMIT_ENABLED`    | Rate limits against password guessing, bot sign-ups and floods (see below) | `true`       |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS` | Sending email: account confirmation and password reset (see below) | – (no email), port `587` |
| `PUBLIC_URL`            | Address of the instance for links in emails, e.g. `https://ragbag.example.com`; required with `SMTP_HOST` | – |
| `QUOTA_LINKS_PER_USER`, `QUOTA_SPACES_PER_USER`, `QUOTA_COLLECTIONS_PER_SPACE`, `QUOTA_UPLOAD_MB_PER_USER` | Quotas for public instances (see below) | unlimited |
| `ALLOW_PRIVATE_HOSTS`   | Let *Fetch details* and image addresses reach your own network (192.168.x.x); loopback stays blocked | `false` |
| `SECURE_COOKIE`         | Force the `Secure` flag on the login cookie (it is set automatically for HTTPS) | `false`    |
| `CORS_ALLOWED_ORIGINS`  | Comma-separated origins of frontends served elsewhere (dev server)   | –                       |
| `ACCESS_TOKEN_MINUTES`  | Lifetime of access tokens                                            | `15`                    |
| `REFRESH_TOKEN_MINUTES` | How long you stay signed in                                          | `20160` (14 days)       |
| `LOG_LEVEL`             | `debug`, `info`, `warn` or `error`; `debug` also logs the frontend's requests | `info`     |
| `LOG_FILE`              | Log into this file instead of stdout, e.g. `/var/log/ragbag/ragbag.log` (rotated) | – (stdout) |
| `LOG_FORMAT`            | `text` or `json` (one object per line, for log shippers)             | `text`                  |
| `LOG_MAX_SIZE_MB`, `LOG_MAX_FILES` | Rotation of `LOG_FILE`: size per file, compressed old files kept | `10`, `5`       |

### Reverse proxy

Ragbag works behind nginx proxy manager, Traefik, Caddy & co. Forward everything to port 8080; the app lives under
`/app`, the API under `/api`. When the proxy terminates HTTPS and sends `X-Forwarded-Proto: https`, the login cookie
is marked `Secure` automatically. Link previews in chat apps (WhatsApp, Signal, Discord) use the `Host` and
`X-Forwarded-Proto` headers the proxy passes on to build the image address.

### Rate limits

On by default (`RATE_LIMIT_ENABLED`), kept in memory and reset by a restart. Refused requests answer `429` with a
`Retry-After` header and show up in the log with the IP.

| What                                | Limit                                   | Per              |
|-------------------------------------|-----------------------------------------|------------------|
| All API requests                    | 600 at once, then 20 per second         | IP               |
| Logins                              | 10 at once, then one every 6 seconds    | IP               |
| Failed logins                       | 10, then one every 3 minutes            | account          |
| Registrations                       | 5, then one every 10 minutes            | IP               |
| Password reset links                | 10, then one per minute                 | IP               |
| Changing the password               | 5, then one per minute                  | user             |
| Loading pages and images by address | 60, then one every 2 seconds            | user             |
| Reports of public links             | 5, then one every 5 minutes             | IP               |

IPv6 addresses count by their /64. When the page limit is reached, new links are still saved, only without
loading title and preview. The client IP comes from `X-Forwarded-For` only when the proxy is in your own network
(loopback or private addresses, e.g. nginx proxy manager or a Cloudflare tunnel on the same host); otherwise every
visitor would share the proxy's address.

### Email

Optional. With `SMTP_HOST` (plus `SMTP_FROM` and `PUBLIC_URL`) Ragbag sends two kinds of mail:

- **Confirming new accounts:** after registering, an account can sign in only once it opened the link sent to its
  address (valid 48 hours; signing in sends a new one). Its first space is created then. Accounts that are not
  confirmed within 7 days are deleted.
- **Forgotten passwords:** *Forgot your password?* on the sign-in page mails a reset link (valid one hour, works
  once). The answer is the same whether the account exists or not.

At most three mails per account every hour or so, whoever asks. `SMTP_TLS` is `starttls` (default, port 587),
`tls` (default for port 465) or `none` (only for a relay on the same host). Without `SMTP_HOST` nothing changes:
accounts can sign in right away and instance admins hand out reset links. Accounts that existed before count as
confirmed, and so do those waiting when email is turned off again.

### Quotas

Unlimited unless set; on a public instance they keep single accounts from filling the disk. Reaching one answers
`403` with a message the app shows.

| Variable                      | Counts                                                                     | E.g.    |
|-------------------------------|----------------------------------------------------------------------------|---------|
| `QUOTA_LINKS_PER_USER`        | Links a user added in any space, trashed ones until the trash is emptied   | `10000` |
| `QUOTA_SPACES_PER_USER`       | Spaces a user owns (the first one of a new account is always created)      | `10`    |
| `QUOTA_COLLECTIONS_PER_SPACE` | Collections of a space, trashed ones included                              | `500`   |
| `QUOTA_UPLOAD_MB_PER_USER`    | Uploaded images: avatar, space icons and preview images (about 100–600 KB each) | `200` |

Imports count as well; one that would exceed a quota is refused as a whole (the preview already says so).
Images uploaded before Ragbag 0.15 count for nobody.

### Moderation

What an instance open to the public needs, for its users and for notice and action under the EU Digital Services
Act (DSA):

- **Deleting the own account** (*Profile & appearance*, with the password): spaces only that person is a member of
  are deleted with it; links they added to spaces shared with others stay there, shown as `deleted-…`. Name,
  email address, password, avatar, PIN, favorites, reactions and dashboards are gone. Being the only owner of a
  space with other members needs another owner first.
- **Blocking accounts** (*Administration → Accounts*, optionally with a reason): the account is signed out, sees the reason
  when signing in, and its public links stop working. Unblocking undoes it. Instance admins cannot be blocked.
- **Deleting accounts** as an admin works like the person deleting it themselves, except that spaces they were
  the only owner of go to the member with the highest role.
- **Reports:** every public link has *Report this collection* at the bottom, for anybody, without an account
  (category, details, optionally name and email, and a confirmation of good faith). Admins find them under
  *Administration → Reports* (the account menu shows how many are open) and **disable the link** (the space cannot
  share the collection again), **dismiss** the report or enable a disabled link again. With email, admins are
  told about new reports, and the reporter and the person who shared the link learn the decision and its reasons.
  Resolved reports are deleted after 180 days.

Reports about anything else (spaces are not public) reach you through the contact address in your legal notice.

### Logs

Ragbag logs one line per API request (method, path, status, duration, IP, user), logins and failed logins (`WARN`,
with the IP, usable for fail2ban), wrong PINs, changes to spaces and memberships, imports and deletions, and server
errors with their cause. Secrets never end up there: share, invite and reset links are masked (`/share/***`), search
terms and other query parameters are left out, passwords and PINs anyway.

By default the lines go to stdout, i.e. `docker logs ragbag` or `journalctl -u ragbag -f`. With `LOG_FILE` they go to
a file instead, which is rotated (`LOG_MAX_SIZE_MB`, `LOG_MAX_FILES`, old files gzipped). The systemd service may
only write `/var/log/ragbag/` (created for it), so use `LOG_FILE=/var/log/ragbag/ragbag.log` there; in Docker put
the file into the data volume, e.g. `/data/logs/ragbag.log`. `LOG_LEVEL=debug` adds the requests for the app's files
and images, `LOG_FORMAT=json` suits Loki, Graylog & co.

### Troubleshooting

- **"Fetch details" finds nothing in Docker/Podman:** the log says *cannot resolve host* when DNS inside the
  container does not work, often because a host firewall (e.g. UFW) blocks the container network. Allow the
  container subnets (Podman: `10.88.0.0/15`) or set public DNS servers (see `docker-compose.yaml`). The log says
  *refusing private address* for links into your own network: set `ALLOW_PRIVATE_HOSTS=true`.
- **Copying links does nothing:** browsers only allow clipboard access over HTTPS or on `localhost`. Ragbag
  falls back to the older copy command, or selects the link for `Ctrl/⌘ C`; a reverse proxy with HTTPS fixes it
  for good.
- **Disk filling up with `install.sh --build`:** every build used to leave its layers behind as untagged images.
  Since v0.12.0 the script removes them after the build; older leftovers go with `podman image prune -f`.

### Backups

Ragbag backs up the whole instance into one file, the same for SQLite and PostgreSQL: every account (with its
password hash and PIN), space, collection, link, reaction, public link and image, with their ids. Left out are
only pending reset and confirmation links.

- **In the app:** account menu → *Administration* → *Backup* (instance admins), which also shows the commands for automatic backups
- **On the command line**, with the same environment as the server (it may keep running):
  ```bash
  ragbag backup ragbag-backup.json.gz            # or without a file to stdout
  sudo sh -c 'set -a; . /etc/ragbag/ragbag.env; ragbag backup' > ragbag-backup.json.gz   # systemd, while it runs
  docker compose exec app ragbag backup > ragbag-backup.json.gz                           # Docker
  ```

The file holds password hashes: keep it as safe as the database. It is gzip compressed JSON (format
`ragbag-instance`); backups of older versions can be restored by newer ones, which migrate them on the way.

Copying the database itself works too: for SQLite stop Ragbag and copy the file (with the `-wal` file if present)
or use `sqlite3 ragbag.db ".backup backup.db"`, for PostgreSQL `pg_dump`. Single collections can additionally be
exported as JSON (collection menu → *Export as JSON*).

### Moving to another server

E.g. from SQLite in the homelab to PostgreSQL on a VPS:

1. On the old instance: *Download backup* (or `ragbag backup`).
2. Install the new one with an empty database (`DATABASE_URL=postgres://…`) and start it.
3. Sign in with the initial password (see *First start*) and pick *Restore a backup*. Ragbag checks the file and
   shows what it holds before replacing anything; the setup account is replaced, so it needs no password of its
   own.
4. Sign in with your old account. Sessions do not move along (the new instance has its own `JWT_SECRET`), share
   links and bookmarklets keep working. Set `ADMIN_USERS` (and `PUBLIC_URL` for emails) like before.

Instead of step 3, `ragbag restore ragbag-backup.json.gz` restores into the empty database at `DATABASE_URL`
(`-dry-run` only checks the file). A backup only ever goes into an instance without data: no spaces and no accounts
besides the setup admin.

### Coming from the Java version

Earlier versions of Ragbag were built with Spring Boot and used a different database schema. The Go version
refuses those databases with a hint instead of touching them. To move over, export your collections in the old
version (*Export as JSON*), start the new one with an empty database, create your spaces and import the files.

---

## 🛠️ Development

| Part     | Stack                                                                          |
|----------|--------------------------------------------------------------------------------|
| Backend  | Go, echo, sqlx, goose migrations, SQLite (pure Go) and PostgreSQL (pgx)        |
| Frontend | Svelte 5, Vite, Tailwind CSS 4, Flowbite, sv-router (`ui/v1`)                  |
| API      | `openapi/openapi.yaml` → Go server interfaces (oapi-codegen) and TypeScript client (openapi-generator) |

```bash
make dev        # API on :8080 with the settings from .env.local (created from .env.local.example)
make dev-ui     # frontend with hot reload on :5173, talks to :8080
```

The first start with an empty database goes through the setup like on a server: sign in as `admin` with the
password in `data/initial-admin-password`. `make help` lists all targets, the most important:

```bash
make test           # Go tests against SQLite
make test-postgres  # the same tests against a throwaway PostgreSQL container
make check          # go vet, gofmt and the frontend type check
make generate       # after changing openapi/openapi.yaml: regenerate server interfaces and TypeScript client
make build          # bin/ragbag with the frontend embedded
make docker         # container image
```

The code layout and the rules behind it (permissions, private collections, migrations) are described in
[AGENTS.md](AGENTS.md).

---

## 🤝 Contributing

Contributions are welcome:

1. Open an issue – feature request or bug
2. Fork the repository & create a branch
3. Submit a pull request with a clear description

---

## 📄 License

Ragbag is released under the **MIT License**. See [LICENSE](LICENSE) for details.

The random space logos are the [Planets](https://www.dicebear.com/styles/planets/) style by
[DiceBear](https://www.dicebear.com), released under [CC0 1.0](https://creativecommons.org/publicdomain/zero/1.0/);
they are generated in the browser, nothing is sent to DiceBear.
