"use strict";

/**
 * Best-effort stop before npm removes vendor/trim.exe.
 * On Windows, a running trim.exe (often `daemon run` from OS logon / autostart)
 * locks the file and npm uninstall fails with EPERM unlink.
 *
 * Prefer: trim uninstall  (or trim daemon uninstall && trim stop) before npm uninstall.
 * This hook soft-fails so package removal still proceeds when the binary is missing.
 */

const { execFileSync, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { resolvePlatform } = require("./platform");

const ROOT = path.join(__dirname, "..");

function binaryPath() {
  const { binaryName } = resolvePlatform();
  return path.join(ROOT, "vendor", binaryName);
}

function runTrim(args) {
  const bin = binaryPath();
  if (!fs.existsSync(bin)) return;
  try {
    execFileSync(bin, args, {
      stdio: "ignore",
      windowsHide: true,
      timeout: 15000,
      env: process.env,
    });
  } catch {
    // Soft-fail: never block npm uninstall.
  }
}

function stopWindowsTrimProcesses() {
  if (process.platform !== "win32") return;
  const bin = binaryPath();
  if (!fs.existsSync(bin)) return;
  // Last resort if daemon uninstall / stop left a lock on this vendor binary.
  try {
    spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-Command",
        `$target = '${bin.replace(/'/g, "''")}';` +
          `Get-CimInstance Win32_Process -Filter "Name = 'trim.exe'" |` +
          ` Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $target) } |` +
          ` ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`,
      ],
      { stdio: "ignore", windowsHide: true, timeout: 15000 },
    );
  } catch {
    // Soft-fail.
  }
}

function main() {
  // Remove OS logon hook first so login does not respawn during uninstall.
  runTrim(["daemon", "uninstall"]);
  runTrim(["stop"]);
  stopWindowsTrimProcesses();
}

main();
