#!/usr/bin/env sh
# Trim uninstall helper (Unix). Prefer: trim uninstall
# Host optional mirror: https://use-trim.com/uninstall.sh
# Full purge is CLI/chrome-driven (no invent path lists in this script).
set -eu

if ! command -v trim >/dev/null 2>&1; then
  echo "trim binary not found on PATH. Install Trim or run the binary path: trim uninstall"
  exit 1
fi
exec trim uninstall
