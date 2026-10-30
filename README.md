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
- **Invite links** with role, expiry and maximum uses – send them via any messenger.
- **Public spaces** show up under *Discover spaces* for everybody on your instance, who can join with one click
  as viewer or contributor.
- Avatars for users and icons for spaces: upload an image or load it from an address.

**Collections & links**
- Nested collections with drag & drop and breadcrumbs.
- **Private collections** are only visible to selected members – not even to space owners and admins. Everything
  inside them (sub-collections, links) is private too, which makes them perfect for gift ideas.
- Links with title, description, tags, favicon and preview image. **Fetch details** loads them from the page.
- Full-text search over titles, addresses, descriptions and tags.
- Three ways to show links – cards with preview images, compact cards or a dense list – chosen per user and device.

**Personal**
- **Favorites** and a **dashboard** per user and space: arrange sections of hand-picked links, favorites and recently
  added links by drag & drop. Great as a homelab start page.

**Data**
- **Save from anywhere**: install Ragbag as an app and share links to it (Android), or use the bookmarklet.
- **Trash**: deleted links and collections can be restored for 30 days.
- **Export and import** collections as versioned JSON files, with a preview of what an import will do.

**Accounts**
- Users change their password themselves. Without email, **admins create one-time reset links** for people who
  forgot theirs.
- Registration can be switched off once everybody has an account.
- Light and dark mode, works on phones.

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

Open Ragbag and **register** – every account gets its own first space. Put your username into `ADMIN_USERS` to
manage all users (profile menu → *Manage users*), and set `REGISTRATION_ENABLED=false` once everybody has an
account if your instance is reachable from the internet.

Database migrations run automatically on start-up.

---

## ⚙️ Configuration

Everything is configured with environment variables. For local runs the server also reads `.env.local` and
`.env` from its working directory; variables that are set in the environment always win.

| Variable                | Description                                                          | Default                 |
|-------------------------|----------------------------------------------------------------------|-------------------------|
| `JWT_SECRET`            | Signs the login tokens, at least 32 characters. **Required**         | –                       |
| `DATABASE_URL`          | `sqlite:<file>` or `postgres://user:password@host:5432/db?sslmode=disable` | `sqlite:data/ragbag.db` (Docker: `/data/ragbag.db`) |
| `RAGBAG_ADDR`           | HTTP address                                                         | `:8080`                 |
| `ADMIN_USERS`           | Comma-separated usernames that may manage all users                  | –                       |
| `REGISTRATION_ENABLED`  | Whether new accounts can be registered                               | `true`                  |
| `ALLOW_PRIVATE_HOSTS`   | Let *Fetch details* and image addresses reach your own network (192.168.x.x); loopback stays blocked | `false` |
| `SECURE_COOKIE`         | Force the `Secure` flag on the login cookie (it is set automatically for HTTPS) | `false`    |
| `CORS_ALLOWED_ORIGINS`  | Comma-separated origins of frontends served elsewhere (dev server)   | –                       |
| `ACCESS_TOKEN_MINUTES`  | Lifetime of access tokens                                            | `15`                    |
| `REFRESH_TOKEN_MINUTES` | How long you stay signed in                                          | `20160` (14 days)       |
| `LOG_LEVEL`             | `debug`, `info`, `warn` or `error`; `debug` also logs the frontend's requests | `info`     |
| `LOG_FILE`              | Log into this file instead of stdout, e.g. `/var/log/ragbag/ragbag.log` (rotated) | – (stdout) |
| `LOG_FORMAT`            | `text` or `json` (one object per line, for log shippers)             | `text`                  |
| `LOG_MAX_SIZE_MB`, `LOG_MAX_FILES` | Rotation of `LOG_FILE`: size per file, compressed old files kept | `10`, `5`       |
| `RAGBAG_DEV`            | Development mode: demo account `admin`/`test123`, no secret required. Never in production | `false` |

### Reverse proxy

Ragbag works behind nginx proxy manager, Traefik, Caddy & co. Forward everything to port 8080; the app lives under
`/app`, the API under `/api`. When the proxy terminates HTTPS and sends `X-Forwarded-Proto: https`, the login cookie
is marked `Secure` automatically. Link previews in chat apps (WhatsApp, Signal, Discord) use the `Host` and
`X-Forwarded-Proto` headers the proxy passes on to build the image address.

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
- **Copying invite links does nothing:** browsers only allow clipboard access over HTTPS or on `localhost`. Ragbag
  falls back to selecting the link for `Ctrl/⌘ C`; a reverse proxy with HTTPS fixes it for good.

### Backups

- **SQLite:** stop Ragbag and copy the database file (with the `-wal` file if present), or copy it while running
  with `sqlite3 ragbag.db ".backup backup.db"`.
- **PostgreSQL:** `docker compose -f docker-compose.postgres.yaml exec db pg_dump -U ragbag ragbag > backup.sql`
- Collections can additionally be exported as JSON (collection menu → *Export as JSON*).

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

Sign in with the demo account `admin` / `test123`. `make help` lists all targets, the most important:

```bash
make test           # Go tests against SQLite
make test-postgres  # the same tests against a throwaway PostgreSQL container
make check          # go vet, gofmt and the frontend type check
make generate       # after changing openapi/openapi.yaml: regenerate server interfaces and TypeScript client
make build          # bin/ragbag with the frontend embedded
make docker         # container image
```

The code layout and the rules behind it (permissions, private collections, migrations) are described in
[CLAUDE.md](CLAUDE.md).

---

## 🤝 Contributing

Contributions are welcome:

1. Open an issue – feature request or bug
2. Fork the repository & create a branch
3. Submit a pull request with a clear description

---

## 📄 License

Ragbag is released under the **MIT License**. See [LICENSE](LICENSE) for details.
