"use strict";

/**
 * Best-effort stop before npm removes vendor/trim.exe.
 *
 * On Windows a running trim.exe (usually `daemon run`) locks the file. npm may
 * rename the package dir to `.trim-*` and then fail with EPERM unlink if any
 * process still holds the binary - including after a rename.
 *
 * Prefer: trim uninstall (or trim daemon uninstall && trim stop) before npm uninstall.
 * This hook soft-fails so package removal still proceeds.
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
      timeout: 12000,
      env: process.env,
    });
  } catch {
    // Soft-fail: never block npm uninstall.
  }
}

/**
 * Stop every trim.exe owned by this npm package tree (@usetrim/…), including
 * npm's temporary `.trim-*` rename folders. Exact path match is not enough:
 * after rename, ExecutablePath / CommandLine can disagree with vendor/.
 */
function stopWindowsTrimProcesses() {
  if (process.platform !== "win32") return;
  const bin = binaryPath().replace(/'/g, "''");
  const ps = [
    "$ErrorActionPreference = 'SilentlyContinue'",
    `$exact = '${bin}'`,
    "Get-CimInstance Win32_Process -Filter \"Name = 'trim.exe'\" | ForEach-Object {",
    "  $path = [string]$_.ExecutablePath",
    "  $cmd  = [string]$_.CommandLine",
    "  $hit = $false",
    "  if ($exact -and $path -and ($path -ieq $exact)) { $hit = $true }",
    "  if ($path -and ($path -match '[\\\\/]node_modules[\\\\/]@usetrim[\\\\/]')) { $hit = $true }",
    "  if ($cmd  -and ($cmd  -match '[\\\\/]node_modules[\\\\/]@usetrim[\\\\/]')) { $hit = $true }",
    "  if ($hit) { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }",
    "}",
    "Start-Sleep -Milliseconds 800",
  ].join("; ");

  try {
    spawnSync(
      "powershell.exe",
      ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", ps],
      { stdio: "ignore", windowsHide: true, timeout: 20000 },
    );
  } catch {
    // Soft-fail.
  }
}

function main() {
  // Kill first so daemon uninstall / stop are not racing a locked binary.
  stopWindowsTrimProcesses();
  runTrim(["daemon", "uninstall"]);
  runTrim(["stop"]);
  stopWindowsTrimProcesses();
}

main();
