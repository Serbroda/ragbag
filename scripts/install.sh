#!/usr/bin/env sh
# Installs or updates the server as a systemd service, removes it with --uninstall.
#
#   sudo ./scripts/install.sh                        # bin/ragbag → /usr/local/bin/ragbag, port 8080
#   sudo ./scripts/install.sh --dir /opt/ragbag      # binary → /opt/ragbag/ragbag
#   sudo ./scripts/install.sh --build                # build bin/ragbag with podman/docker first (no Go/Node/make)
#   sudo ./scripts/install.sh --binary ./ragbag-linux-amd64 --port 9000
#   sudo ./scripts/install.sh --uninstall [--purge]  # --purge also deletes data and configuration
#
# Running it again updates the binary (where it was installed before) and restarts the service; the
# configuration is kept. Configuration: /etc/<app>/<app>.env  Data: /var/lib/<app>  Logs: journalctl -u <app> -f
set -eu

# Reusable for other single-binary apps (e.g. jucosuma): adjust these lines
APP=ragbag
DESCRIPTION="Ragbag bookmark manager"
DEFAULT_BINARY="bin/$APP"
DEFAULT_PORT=8080

DEFAULT_DIR=/usr/local/bin
CONFIG_DIR="/etc/$APP"
ENV_FILE="$CONFIG_DIR/$APP.env"
DATA_DIR="/var/lib/$APP"
UNIT_FILE="/etc/systemd/system/$APP.service"

binary="$DEFAULT_BINARY"
port="$DEFAULT_PORT"
dir=""
action=install
purge=false
build=false

usage() {
	sed -n '2,11p' "$0" | sed 's/^# \{0,1\}//'
	exit "${1:-0}"
}

while [ $# -gt 0 ]; do
	case "$1" in
		--binary) binary="$2"; shift 2 ;;
		--port) port="$2"; shift 2 ;;
		--dir) dir="${2%/}"; shift 2 ;;
		--uninstall) action=uninstall; shift ;;
		--purge) purge=true; shift ;;
		--build) build=true; shift ;;
		-h | --help) usage ;;
		*) echo "Unknown option: $1" >&2; usage 1 ;;
	esac
done

[ "$(id -u)" -eq 0 ] || { echo "Please run as root (sudo $0 $*)" >&2; exit 1; }
command -v systemctl >/dev/null || { echo "systemd is required" >&2; exit 1; }

# Without --dir, updates and uninstall use the location of the existing installation
installed_path=$(sed -n 's/^ExecStart=//p' "$UNIT_FILE" 2>/dev/null || true)
if [ -n "$dir" ]; then
	BIN_PATH="$dir/$APP"
elif [ -n "$installed_path" ]; then
	BIN_PATH="$installed_path"
else
	BIN_PATH="$DEFAULT_DIR/$APP"
fi

uninstall() {
	systemctl disable --now "$APP" 2>/dev/null || true
	rm -f "$UNIT_FILE"
	remove_binary "$BIN_PATH"
	systemctl daemon-reload
	if [ "$purge" = true ]; then
		# DynamicUser keeps the state below /var/lib/private
		rm -rf "$CONFIG_DIR" "$DATA_DIR" "/var/lib/private/$APP"
		echo "$APP removed, including data and configuration."
	else
		echo "$APP removed. Data ($DATA_DIR) and configuration ($CONFIG_DIR) were kept; --purge deletes them."
	fi
}

# Removes the binary and its directory if it is one of its own (e.g. /opt/ragbag) and empty now.
remove_binary() {
	rm -f "$1"
	bin_dir=$(dirname "$1")
	if [ "$(basename "$bin_dir")" = "$APP" ]; then
		rmdir "$bin_dir" 2>/dev/null || true
	fi
}

random_secret() {
	if command -v openssl >/dev/null; then
		openssl rand -base64 32
	else
		head -c 32 /dev/urandom | base64
	fi
}

write_env_file() {
	mkdir -p "$CONFIG_DIR"
	# The secret is in there, so only root reads it; systemd passes it to the service
	touch "$ENV_FILE"
	chmod 600 "$ENV_FILE"
	cat >"$ENV_FILE" <<ENV
# Configuration of $APP, see .env.example in the repository for all options.
# Apply changes with: sudo systemctl restart $APP

RAGBAG_ADDR=:$port
DATABASE_URL=sqlite:$DATA_DIR/$APP.db
JWT_SECRET=$(random_secret)

# Usernames that may manage all users and create password reset links (comma-separated)
ADMIN_USERS=

# Disable after creating your accounts if the instance is publicly reachable
REGISTRATION_ENABLED=true

# Let "Fetch details" load pages from your own network (192.168.x.x)
ALLOW_PRIVATE_HOSTS=false
ENV
	echo "Created $ENV_FILE with a random JWT_SECRET"
}

write_unit_file() {
	install -m 0644 /dev/null "$UNIT_FILE"
	cat >"$UNIT_FILE" <<UNIT
[Unit]
Description=$DESCRIPTION
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=$BIN_PATH
EnvironmentFile=$ENV_FILE
# systemd creates a throwaway unprivileged user and $DATA_DIR owned by it
DynamicUser=yes
StateDirectory=$APP
WorkingDirectory=$DATA_DIR
Restart=on-failure
RestartSec=5

# Hardening: the service can only write its state directory
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes
LockPersonality=yes
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX

[Install]
WantedBy=multi-user.target
UNIT
}

# Builds the binary in a container (Dockerfile stage "binary"), so the host needs neither Go nor Node nor make.
build_binary() {
	engine=$(command -v podman || command -v docker || true)
	[ -n "$engine" ] || { echo "--build needs podman or docker" >&2; exit 1; }
	repo=$(cd "$(dirname "$0")/.." && pwd)
	version=$(git -C "$repo" describe --tags --always --dirty 2>/dev/null || echo dev)
	echo "Building $APP $version with $(basename "$engine") (this takes a few minutes the first time)..."
	"$engine" build --target binary --build-arg VERSION="$version" --output "$repo/bin" "$repo"
	binary="$repo/bin/$APP"
}

install_app() {
	[ "$build" = true ] && build_binary
	[ -f "$binary" ] || { echo "Binary not found: $binary (build it with --build or make build)" >&2; exit 1; }
	updating=false
	[ -n "$installed_path" ] && updating=true

	if [ -n "$installed_path" ] && [ "$installed_path" != "$BIN_PATH" ]; then
		echo "Moving the binary from $installed_path to $BIN_PATH"
		remove_binary "$installed_path"
	fi
	install -d -m 0755 "$(dirname "$BIN_PATH")"
	install -m 0755 "$binary" "$BIN_PATH"
	# SELinux (Fedora, RHEL): give the copy the label of its new location so systemd may execute it
	command -v restorecon >/dev/null && restorecon "$BIN_PATH" 2>/dev/null || true
	[ -f "$ENV_FILE" ] || write_env_file
	write_unit_file
	systemctl daemon-reload
	systemctl enable "$APP" >/dev/null 2>&1
	systemctl restart "$APP"

	sleep 2
	if ! systemctl is-active --quiet "$APP"; then
		echo "$APP did not start, see: journalctl -u $APP -n 50" >&2
		journalctl -u "$APP" -n 20 --no-pager >&2 || true
		exit 1
	fi
	configured_port=$(sed -n 's/^RAGBAG_ADDR=.*:\([0-9]*\)$/\1/p' "$ENV_FILE")
	[ "$updating" = true ] && echo "$APP updated and restarted." || echo "$APP installed and started."
	echo "  Open:    http://$(hostname):${configured_port:-$port}"
	echo "  Binary:  $BIN_PATH"
	echo "  Config:  $ENV_FILE (then: sudo systemctl restart $APP)"
	echo "  Logs:    journalctl -u $APP -f"
}

case "$action" in
	install) install_app ;;
	uninstall) uninstall ;;
esac
