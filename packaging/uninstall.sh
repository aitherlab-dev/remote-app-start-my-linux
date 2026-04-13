#!/usr/bin/env bash
# RemoteLauncher uninstaller — removes the binary and systemd user
# unit. User data under ~/.config/remotelauncher/ (certs, tokens,
# config.toml) is intentionally NOT touched — the user removes it
# manually if they want a clean slate.
set -euo pipefail

BIN_DIR="${HOME}/.local/bin"
UNIT_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/systemd/user"
BINARY_DST="${BIN_DIR}/remotelauncher"
UNIT_DST="${UNIT_DIR}/remotelauncher.service"

echo ">>> RemoteLauncher uninstaller"

if systemctl --user list-unit-files remotelauncher.service >/dev/null 2>&1; then
	echo ">>> stopping and disabling remotelauncher.service"
	systemctl --user disable --now remotelauncher.service 2>/dev/null || true
fi

if [[ -f "${UNIT_DST}" ]]; then
	echo ">>> removing ${UNIT_DST}"
	rm -f "${UNIT_DST}"
else
	echo ">>> unit file already absent"
fi

if [[ -f "${BINARY_DST}" ]]; then
	echo ">>> removing ${BINARY_DST}"
	rm -f "${BINARY_DST}"
else
	echo ">>> binary already absent"
fi

echo ">>> reloading user systemd"
systemctl --user daemon-reload

echo ">>> done. User data under ~/.config/remotelauncher/ was NOT removed."
