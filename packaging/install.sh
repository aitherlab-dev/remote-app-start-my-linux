#!/usr/bin/env bash
# RemoteLauncher installer — installs the server binary and systemd
# user unit under $HOME. Re-running the script is safe: it overwrites
# the binary/unit in place and re-enables the service without touching
# user data under ~/.config/remotelauncher/.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVER_DIR="${REPO_ROOT}/server"
BINARY_SRC="${SERVER_DIR}/bin/remotelauncher"
UNIT_SRC="${SCRIPT_DIR}/remotelauncher.service"

BIN_DIR="${HOME}/.local/bin"
UNIT_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/systemd/user"
BINARY_DST="${BIN_DIR}/remotelauncher"
UNIT_DST="${UNIT_DIR}/remotelauncher.service"

echo ">>> RemoteLauncher installer"

if [[ ! -f "${BINARY_SRC}" ]]; then
	echo ">>> binary not found at ${BINARY_SRC} — building"
	make -C "${SERVER_DIR}" build
fi

if [[ ! -f "${UNIT_SRC}" ]]; then
	echo "!!! unit template missing: ${UNIT_SRC}" >&2
	exit 1
fi

echo ">>> installing binary to ${BINARY_DST}"
mkdir -p "${BIN_DIR}"
install -m 0755 "${BINARY_SRC}" "${BINARY_DST}"

echo ">>> installing unit to ${UNIT_DST}"
mkdir -p "${UNIT_DIR}"
install -m 0644 "${UNIT_SRC}" "${UNIT_DST}"

echo ">>> reloading user systemd"
systemctl --user daemon-reload

echo ">>> enabling remotelauncher.service"
systemctl --user enable --now remotelauncher.service

echo ">>> current status:"
systemctl --user --no-pager status remotelauncher.service || true

echo ">>> done. Logs: journalctl --user -u remotelauncher -f"
