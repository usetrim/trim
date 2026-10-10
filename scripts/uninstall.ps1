#Requires -Version 5.1
# Trim Windows uninstall helper. Prefer: trim uninstall
# Usage: irm https://use-trim.com/uninstall.ps1 | iex
# Full purge is CLI/chrome-driven (no invent path lists in this script).
$ErrorActionPreference = "Stop"

$trim = Get-Command trim -ErrorAction SilentlyContinue
if (-not $trim) {
  Write-Host "trim.exe not found on PATH. Install Trim or run the binary path: trim uninstall"
  exit 1
}
& trim uninstall
exit $LASTEXITCODE
