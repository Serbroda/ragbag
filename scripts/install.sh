#!/usr/bin/env sh
# Installs or updates the server as a systemd service, removes it with --uninstall.
#
#   sudo ./scripts/install.sh                        # bin/ragbag, port 8080
#   sudo ./scripts/install.sh --binary ./ragbag-linux-amd64 --port 9000
#   sudo ./scripts/install.sh --uninstall [--purge]  # --purge also deletes data and configuration
#
# Running it again updates the binary and restarts the service; the configuration is kept.
#   Configuration: /etc/<app>/<app>.env   Data: /var/lib/<app>   Logs: journalctl -u <app> -f
set -eu

# Reusable for other single-binary apps (e.g. jucosuma): adjust these lines
APP=ragbag
DESCRIPTION="Ragbag bookmark manager"
DEFAULT_BINARY="bin/$APP"
DEFAULT_PORT=8080

BIN_PATH="/usr/local/bin/$APP"
CONFIG_DIR="/etc/$APP"
ENV_FILE="$CONFIG_DIR/$APP.env"
DATA_DIR="/var/lib/$APP"
UNIT_FILE="/etc/systemd/system/$APP.service"

binary="$DEFAULT_BINARY"
port="$DEFAULT_PORT"
action=install
purge=false

usage() {
	sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'
	exit "${1:-0}"
}

while [ $# -gt 0 ]; do
	case "$1" in
		--binary) binary="$2"; shift 2 ;;
		--port) port="$2"; shift 2 ;;
		--uninstall) action=uninstall; shift ;;
		--purge) purge=true; shift ;;
		-h | --help) usage ;;
		*) echo "Unknown option: $1" >&2; usage 1 ;;
	esac
done

[ "$(id -u)" -eq 0 ] || { echo "Please run as root (sudo $0 $*)" >&2; exit 1; }
command -v systemctl >/dev/null || { echo "systemd is required" >&2; exit 1; }

uninstall() {
	systemctl disable --now "$APP" 2>/dev/null || true
	rm -f "$UNIT_FILE" "$BIN_PATH"
	systemctl daemon-reload
	if [ "$purge" = true ]; then
		# DynamicUser keeps the state below /var/lib/private
		rm -rf "$CONFIG_DIR" "$DATA_DIR" "/var/lib/private/$APP"
		echo "$APP removed, including data and configuration."
	else
		echo "$APP removed. Data ($DATA_DIR) and configuration ($CONFIG_DIR) were kept; --purge deletes them."
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
	umask 077
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

install_app() {
	[ -f "$binary" ] || { echo "Binary not found: $binary (build it with: make build)" >&2; exit 1; }
	updating=false
	[ -f "$BIN_PATH" ] && updating=true

	install -m 0755 "$binary" "$BIN_PATH"
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
	echo "  Config:  $ENV_FILE (then: sudo systemctl restart $APP)"
	echo "  Logs:    journalctl -u $APP -f"
}

case "$action" in
	install) install_app ;;
	uninstall) uninstall ;;
esac
