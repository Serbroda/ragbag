#!/bin/sh
set -e

# Bind mounts (e.g. ./data:/data) keep the ownership of the host directory, which the unprivileged
# app user usually cannot write to. When started as root, hand /data to that user and drop privileges;
# when started with --user / `user:`, run as that user and leave the permissions alone.
if [ "$(id -u)" = "0" ]; then
	if ! find /data \! -user ragbag -exec chown ragbag:ragbag {} + 2>/dev/null; then
		echo "WARNING: cannot change the owner of /data. On SELinux hosts (Fedora, RHEL) mount it with :Z," >&2
		echo "         e.g. ./data:/data:Z, or run 'chown -R 1000:1000 ./data' on the host." >&2
	fi
	exec su-exec ragbag "$@"
fi
exec "$@"
