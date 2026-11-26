#!/bin/sh
set -eu

umask 027

# Runtime config lives on the data volume so install / auto salt survive restarts.
DATA_CFG="/app/site/data/gboot-config.json"

prepare_config() {
	if [ -n "${GBOOT_CONFIG:-}" ] && [ "$GBOOT_CONFIG" != "/app/config.json" ] && [ "$GBOOT_CONFIG" != "$DATA_CFG" ]; then
		return 0
	fi
	if [ -f "$DATA_CFG" ]; then
		export GBOOT_CONFIG="$DATA_CFG"
		return 0
	fi
	if [ -f /app/site/data/install.lock ] && [ -f /app/config.json ]; then
		cp /app/config.json "$DATA_CFG"
		chmod 600 "$DATA_CFG" 2>/dev/null || true
	fi
	export GBOOT_CONFIG="$DATA_CFG"
}

if [ "$(id -u)" = "0" ]; then
	for d in /app/site/data /app/site/static/upload /app/site/static/backup; do
		mkdir -p "$d"
		# no-new-privileges may make chown fail even with CAP_CHOWN; volumes are often already gboot-owned
		chown -R gboot:gboot "$d" 2>/dev/null || true
		chmod 750 "$d" 2>/dev/null || true
	done
	prepare_config
	if [ -f "$GBOOT_CONFIG" ]; then
		chown gboot:gboot "$GBOOT_CONFIG" 2>/dev/null || true
		chmod 600 "$GBOOT_CONFIG" 2>/dev/null || true
	fi
	exec su-exec gboot:gboot env GBOOT_CONFIG="${GBOOT_CONFIG:-$DATA_CFG}" /app/pbootcms-go
fi

prepare_config
exec /app/pbootcms-go
