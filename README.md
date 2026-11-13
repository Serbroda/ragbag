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
- Choose the space Ragbag **opens on start** (settings, or right-click a space); otherwise it reopens the last one.
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
- Full-text search over titles, addresses, descriptions and tags: links have to match every word, `#tag` finds
  exactly that tag, `"two words"` a phrase.
- **Discord**: a space can post every newly saved link to a Discord channel (space settings → Integrations, with
  a webhook of that channel), with a link to its collection in Ragbag (set `PUBLIC_URL` behind a reverse proxy).
  Links in private collections and imports are never posted.
- **Smart collections** show every link of the space matching a search (e.g. `#recipe vegan`), wherever it is
  saved, and for every member only what that member may see.
- Three ways to show links – cards with preview images, compact cards or a dense list – chosen per user and device.
- **Headlines**: cards of news sites, patch notes or blogs can show the latest headlines of their RSS/Atom feed
  (found on the page), optionally only those containing certain words. Loaded when shown, at most once an hour.
  A **Headlines** section on the dashboard merges them into one list, newest first.

**Personal**
- **Favorites** and a **dashboard** per user and space: arrange sections of hand-picked links, favorites and recently
  added links by drag & drop, and add links right from there. Great as a homelab start page.

**Data**
- **Save from anywhere**: install Ragbag as an app and share links to it (Android), or use the bookmarklet.
- **Trash**: deleted links and collections can be restored for 30 days.
- **Export and import** collections as versioned JSON files, with a preview of what an import will do. The
  import also takes the bookmarks file (.html) of Chrome, Firefox, Safari, Edge, Pocket, Raindrop & co.: folders
  become sub-collections, tags, descriptions and dates are kept.

**Accounts**
- Users change their password themselves. Without email, **admins create one-time reset links** for people who
  forgot theirs.
- **Two-factor authentication** with any authenticator app (TOTP), turned on and off by each user in their
  settings; ten recovery codes for a lost phone. Password reset links never get around it. Who lost app and codes
  is helped by an admin (*Reset 2FA* in the accounts list), an admin by `ragbag reset-2fa <username>` on the server.
- **API tokens** for scripts, shortcuts and apps, read only or read & write, created and revoked in the settings
  (see [API](#api)).
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
cp .env.example .env        # set JWT_SECRET (openssl rand -base64 32), ADMIN_USERS and PUBLIC_URL
docker compose up -d --build
docker compose exec app cat /data/initial-admin-password
```

Open <http://localhost:8080> and sign in with that password (see *First start*). The database lives in the
`ragbag-data` volume. Every setting in `.env` is passed to the container, the same list in both compose files.

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

Everybody else **registers** – every account gets its own first space. Admins manage all users in the account menu
(→ *Administration*). Set `REGISTRATION_ENABLED=false` once everybody has an account if your instance is reachable
from the internet. Invite links of admins still let new people register (and join the space right away); invite
links of everybody else then only work for existing accounts.

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
| `JWT_SECRET`            | Signs the login tokens and encrypts two-factor secrets, at least 32 characters; with another one, accounts with two-factor authentication need a recovery code. **Required** | –                       |
| `DATABASE_URL`          | `sqlite:<file>` or `postgres://user:password@host:5432/db?sslmode=disable` | `sqlite:data/ragbag.db` (Docker: `/data/ragbag.db`) |
| `RAGBAG_ADDR`           | HTTP address                                                         | `:8080`                 |
| `ADMIN_USERS`           | Comma-separated usernames that may manage all users and download backups; the first start creates the first of them | – (the account created on the first start) |
| `DATA_DIR`              | Where the first start puts `initial-admin-password`; legal pages in `legal/` | the SQLite file's directory, systemd's state directory, else `data` (Docker: `/data`) |
| `REGISTRATION_ENABLED`  | Whether anybody can register; when off, only with an admin's invite  | `true`                  |
| `RATE_LIMIT_ENABLED`    | Rate limits against password guessing, bot sign-ups and floods (see below) | `true`       |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS` | Sending email: account confirmation and password reset (see below) | – (no email), port `587` |
| `IMPRINT_URL`, `PRIVACY_URL`, `TERMS_URL` | Legal pages elsewhere instead of files in `DATA_DIR/legal` (see *Legal pages*) | – |
| `PUBLIC_URL`            | Address of the instance for links in emails and chat previews, e.g. `https://ragbag.example.com`; required with `SMTP_HOST` | – |
| `QUOTA_LINKS_PER_USER`, `QUOTA_SPACES_PER_USER`, `QUOTA_COLLECTIONS_PER_SPACE`, `QUOTA_UPLOAD_MB_PER_USER` | Quotas for public instances (see below) | unlimited |
| `PLANS_ENABLED`, `QUOTA_PRO_*`, `PLANS_FREE_API_TOKENS` | Free and Pro plans, Pro granted by admins (see *Plans*) | off |
| `SUPPORT_URL`           | Donation page (Ko-fi & co.) linked in the app, https only               | –                       |
| `ALLOW_PRIVATE_HOSTS`   | Let *Fetch details* and image addresses reach your own network (192.168.x.x); loopback stays blocked | `false` |
| `SECURE_COOKIE`         | Force the `Secure` flag on the login cookie (it is set automatically for HTTPS) | `false`    |
| `CORS_ALLOWED_ORIGINS`  | Comma-separated origins of frontends served elsewhere (dev server)   | –                       |
| `ACCESS_TOKEN_MINUTES`  | Lifetime of access tokens                                            | `15`                    |
| `REFRESH_TOKEN_MINUTES` | How long you stay signed in                                          | `20160` (14 days)       |
| `LOG_LEVEL`             | `debug`, `info`, `warn` or `error`; `debug` also logs the frontend's requests | `info`     |
| `LOG_FILE`              | Log into this file instead of stdout, e.g. `/var/log/ragbag/ragbag.log` (rotated) | – (stdout) |
| `LOG_FORMAT`            | `text` or `json` (one object per line, for log shippers)             | `text`                  |
| `LOG_MAX_SIZE_MB`, `LOG_MAX_FILES` | Rotation of `LOG_FILE`: size per file, compressed old files kept | `10`, `5`       |
| `LOG_MAX_AGE_DAYS`      | Old files of `LOG_FILE` are deleted after that many days (a new file starts daily) | `7`  |
| `LOG_ANONYMIZE_IP`      | Log IPv4 addresses without the last byte, IPv6 by their /48 (rate limits use the full address) | `false` |
| `BACKUP_KEEP`           | The server backs itself up and keeps the newest that many; `0` is off (see *Backups*) | `0` (templates: `7`) |
| `BACKUP_INTERVAL_HOURS` | Hours between those backups                                          | `24`                    |
| `BACKUP_DIR`            | Where they go                                                        | `DATA_DIR/backups`      |

### API

Everything the app does goes through the REST API under `/api/v1`, described in
[`openapi/openapi.yaml`](openapi/openapi.yaml). Scripts and apps use it with a **personal API token** from the
settings (*Apps & API*), sent as `Authorization: Bearer rb_…`:

```bash
TOKEN=rb_…; URL=https://ragbag.example.org/api/v1
curl -H "Authorization: Bearer $TOKEN" $URL/spaces                         # your spaces
curl -H "Authorization: Bearer $TOKEN" $URL/spaces/<space>/collections     # the collection tree
curl -H "Authorization: Bearer $TOKEN" $URL/collections/<collection>/bookmarks
curl -H "Authorization: Bearer $TOKEN" "$URL/spaces/<space>/bookmarks/search?q=%23recipe"
# Save a link (title, description and preview are loaded from the page); needs a "read & write" token
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"url":"https://example.org","tags":["later"]}' $URL/collections/<collection>/bookmarks
# Import a browser's bookmarks file
curl -H "Authorization: Bearer $TOKEN" -H "Content-Type: text/html" \
  --data-binary @bookmarks.html "$URL/collections/<collection>/import?preview=true"
```

A token acts as you, with your role in each space, but only on links and collections: the account (password,
two-factor authentication, tokens), spaces' settings, members, invites, public links and who sees a collection
are out of its reach, and collections you locked stay hidden. *Read only* tokens answer 403 to every change.
Only a hash is stored; tokens survive a password change and end when revoked, expired or the account is blocked.
With [plans](#plans), API tokens are part of Pro (`PLANS_FREE_API_TOKENS=true` gives them to everybody): Free
accounts cannot create any, and the tokens of an account that is Free again answer 403 until it is Pro again.

### Reverse proxy

Ragbag works behind nginx proxy manager, Traefik, Caddy & co. Forward everything to port 8080; the app lives under
`/app`, the API under `/api`. When the proxy terminates HTTPS and sends `X-Forwarded-Proto: https`, the login cookie
is marked `Secure` automatically. Link previews in chat apps (WhatsApp, Signal, Discord) need the public address
for their image: set `PUBLIC_URL`, otherwise Ragbag builds it from the `Host` and `X-Forwarded-Proto` headers the
proxy passes on.

### Security headers

Ragbag sets them itself, the proxy needs no extra configuration (and should not add a second
`Content-Security-Policy`, which would make the browser apply both):

| Header | Value |
|--------|-------|
| `Content-Security-Policy` | Scripts and styles only from Ragbag itself, images from anywhere (link previews and favicons, also by plain HTTP), never inside a frame. API answers: `default-src 'none'` |
| `Strict-Transport-Security` | `max-age=31536000`, only when reached via HTTPS (also `X-Forwarded-Proto: https`), so HTTP in a homelab keeps working; without `includeSubDomains` |
| `X-Frame-Options` | `DENY` |
| `Referrer-Policy` | `no-referrer`: opening a saved link tells the site nothing about your Ragbag |
| `X-Content-Type-Options` | `nosniff` |
| `Cross-Origin-Opener-Policy` | `same-origin` (not for the bookmarklet's window, which closes itself) |
| `Permissions-Policy` | camera, microphone, location, payment, USB and topics off |
| `Cache-Control` | `no-store` for API answers; images and assets are cached |

Turning on HSTS in the proxy as well (nginx proxy manager: *HSTS Enabled*) does no harm.

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
| Loading pages, images and feeds     | 60, then one every 2 seconds            | user (public links: IP) |
| Reports of public links             | 5, then one every 5 minutes             | IP               |

IPv6 addresses count by their /64. When the page limit is reached, new links are still saved, only without
loading title and preview; feeds kept for an hour do not count. The client IP comes from `X-Forwarded-For` only when the proxy is in your own network
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

### Plans

Off by default: a self-hosted instance treats everybody the same, with the quotas above or none. A public
instance can split accounts into **Free** and **Pro** with `PLANS_ENABLED=true`: the `QUOTA_*` variables are then
the limits of Free, `QUOTA_PRO_*` (same names with `PRO_`, unlimited unless set) those of Pro.

```bash
PLANS_ENABLED=true
QUOTA_SPACES_PER_USER=2          # Free: two own spaces (the first of a new account counts)
QUOTA_UPLOAD_MB_PER_USER=50      # Free: 50 MB of images
QUOTA_PRO_UPLOAD_MB_PER_USER=1000
```

Pro are the instance admins and accounts an admin made Pro (*Administration → Accounts → Make Pro*). Joining the spaces of others is free; a space has the Pro limits when one of its owners
is Pro. Going back to Free keeps everything, only adding beyond the Free limits is refused. Everybody sees their
plan, usage and limits in the account menu under *Plan*. [API tokens](#api) are part of Pro as well, unless
`PLANS_FREE_API_TOKENS=true`.

Pro is never sold. To let people chip in for the server, set `SUPPORT_URL` to a donation page (e.g.
`https://ko-fi.com/serbroda`): a Ko-fi button next to the search (each person can close it) and the account
menu then link it, with or without plans. Donations unlock nothing, and the plan page does not mention them.

### Moderation

What an instance open to the public needs, for its users and for notice and action under the EU Digital Services
Act (DSA):

- **Deleting the own account** (*Settings → Account*, with the password): spaces only that person is a member of
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

### Legal pages

A public instance needs an imprint and a privacy policy (in Germany § 5 DDG and Art. 13 GDPR), terms of use are
recommended. Put them as Markdown into the data directory – `legal/imprint.md`, `legal/privacy.md`,
`legal/terms.md` (systemd: `/var/lib/ragbag/legal/`, Docker: `/data/legal/`) – or link pages elsewhere with
`IMPRINT_URL`, `PRIVACY_URL` and `TERMS_URL`. Ragbag shows them at `/app/legal/…` and links them on the sign-in
page, on public links and in the account menu; with terms, registering needs accepting them (the time is
stored). Changes to the files apply without a restart, raw HTML in them is not shown. The admin overview says
which are missing.

**Templates in German** for what Ragbag actually processes are in [`docs/legal`](docs/legal): copy them, fill in
the `[…]` parts and drop what does not apply. They are drafts, not legal advice.

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

**How long logs are kept** belongs into your privacy policy, as they hold IP addresses:

- `LOG_FILE`: a new file every day (and at `LOG_MAX_SIZE_MB`), older ones are deleted after `LOG_MAX_AGE_DAYS`
  (default 7).
- journald (the systemd service without `LOG_FILE`): as long as journald keeps anything, for all services; set e.g.
  `MaxRetentionSec=7day` in `/etc/systemd/journald.conf`.
- Docker: the compose files limit the container log to 5 × 10 MB; it is gone with the container.

`LOG_ANONYMIZE_IP=true` logs shortened addresses (`203.0.113.0`); fail2ban & co. then cannot block single
addresses anymore, the built-in rate limits still can.

### Monitoring

`GET /api/health` answers `200 {"status":"ok","version":"…"}`, or `503` when the database does not, without a
token – for Uptime Kuma, Gatus & co. and the container's health check. Successful checks are only logged at
`debug`.

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

- **Automatically:** with `BACKUP_KEEP=7` (set in the templates of `install.sh` and `.env.example`) the server
  writes a backup every `BACKUP_INTERVAL_HOURS` (24) into `BACKUP_DIR` (`DATA_DIR/backups`: systemd
  `/var/lib/ragbag/backups`, Docker `/data/backups`) and keeps the newest 7. The schedule follows the newest file,
  so restarts neither add nor skip one. They are on the same disk as the database: copy them elsewhere now and
  then (rsync, restic, …).
- **In the app:** account menu → *Administration* → *Backup* (instance admins): download one now, the stored
  ones, or *Back up now*
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
   own. The preview also says who will be an admin afterwards (the accounts in `ADMIN_USERS`) and
   warns when nobody would be.
4. Sign in with your old account. Sessions do not move along (the new instance has its own `JWT_SECRET`), share
   links and bookmarklets keep working. Set `ADMIN_USERS` (and `PUBLIC_URL` for emails) like before.
   Accounts with two-factor authentication: take the old `JWT_SECRET` along, their authenticator apps only keep
   working with it (otherwise they sign in with a recovery code and set it up again).

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
make dev-mail   # Mailpit catches every email (SMTP_* in .env.local.example), read them at http://localhost:8025
```

In GoLand/IntelliJ the shared run configuration *Mailpit* does the same as `make dev-mail`; add it to a compound
configuration to start it with the server.

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
