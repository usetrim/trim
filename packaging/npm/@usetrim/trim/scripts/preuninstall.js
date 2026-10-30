"use strict";

/**
 * Best-effort cleanup before npm removes vendor/trim.exe.
 *
 * Current Trim registers Windows OS logon against
 * %LOCALAPPDATA%\Trim\bin\trim.exe (stable copy), so a healthy Always-on
 * daemon does not lock this package’s vendor binary.
 *
 * This hook:
 *   1) always stops trim.exe that lock this npm package tree (so upgrades can replace vendor/)
 *   2) on real `npm uninstall` only: marker + daemon uninstall + stop, then unlock
 *      vendor (delete or rename + reboot-pending delete) so a locked trim.exe cannot
 *      survive npm’s tree removal. Skipped on upgrade/install/update.
 *
 * Prefer: trim uninstall (schedules npm uninstall -g after exit on Windows).
 * Soft-fails so package removal still proceeds.
 */

const { execFileSync, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { resolvePlatform } = require("./platform");

const ROOT = path.join(__dirname, "..");
const MARKER = path.join(ROOT, ".trim-uninstalling");

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

function isNpmUninstallCommand() {
  const cmd = String(process.env.npm_command || "").toLowerCase();
  return cmd === "uninstall" || cmd === "remove";
}

/**
 * Stop Trim processes that lock this package or the stable LocalAppData copy.
 * On upgrade we still unlock vendor/; LocalAppData matches are only force-stopped
 * when this is a real uninstall (Preserve Always-on during npm upgrade).
 */
function stopWindowsTrimProcesses(includeStable) {
  if (process.platform !== "win32") return;
  const bin = binaryPath().replace(/'/g, "''");
  const includeStableFlag = includeStable ? "1" : "0";
  const ps = [
    "$ErrorActionPreference = 'SilentlyContinue'",
    `$exact = '${bin}'`,
    `$includeStable = ${includeStableFlag}`,
    "Get-CimInstance Win32_Process -Filter \"Name = 'trim.exe'\" | ForEach-Object {",
    "  $path = [string]$_.ExecutablePath",
    "  $cmd  = [string]$_.CommandLine",
    "  $hit = $false",
    "  if ($exact -and $path -and ($path -ieq $exact)) { $hit = $true }",
    "  if ($path -and ($path -match '[\\\\/]node_modules[\\\\/]@usetrim[\\\\/]')) { $hit = $true }",
    "  if ($cmd  -and ($cmd  -match '[\\\\/]node_modules[\\\\/]@usetrim[\\\\/]')) { $hit = $true }",
    "  if ($includeStable -eq 1) {",
    "    if ($path -and ($path -match '[\\\\/]Trim[\\\\/]bin[\\\\/]trim\\.exe$')) { $hit = $true }",
    "    if ($cmd  -and ($cmd  -match '[\\\\/]Trim[\\\\/]bin[\\\\/]trim\\.exe')) { $hit = $true }",
    "  }",
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

/** MOVEFILE_DELAY_UNTIL_REBOOT (4) for a path that may still be mapped. */
function scheduleWindowsRebootDelete(filePath) {
  if (process.platform !== "win32") return;
  const p = String(filePath || "").replace(/'/g, "''");
  if (!p) return;
  try {
    spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-Command",
        `$p='${p}'; if (Test-Path -LiteralPath $p) { Add-Type -Namespace Win32 -Name Native -MemberDefinition '[DllImport("kernel32.dll",CharSet=CharSet.Unicode,SetLastError=true)] public static extern bool MoveFileEx(string lpExistingFileName, string lpNewFileName, int dwFlags);'; [Win32.Native]::MoveFileEx($p, $null, 4) | Out-Null }`,
      ],
      { stdio: "ignore", windowsHide: true, timeout: 8000 },
    );
  } catch {
    // Soft-fail.
  }
}

/** Best-effort unlock of vendor binary for real uninstall (Windows file lock). */
function unlockVendorBinaryForUninstall() {
  const bin = binaryPath();
  if (!fs.existsSync(bin)) return;
  try {
    fs.unlinkSync(bin);
    return;
  } catch {
    // Mapped running exe - rename aside (Windows allows rename of locked image).
  }
  const aside = bin + ".uninstalling";
  try {
    if (fs.existsSync(aside)) {
      try {
        fs.unlinkSync(aside);
      } catch {
        // ignore
      }
    }
    fs.renameSync(bin, aside);
    scheduleWindowsRebootDelete(aside);
  } catch {
    // Rename failed - still register reboot delete on the original locked path.
    scheduleWindowsRebootDelete(bin);
  }
}

function main() {
  const fullRemove = isNpmUninstallCommand();
  // Always unlock this package tree so npm can replace/remove vendor/.
  stopWindowsTrimProcesses(fullRemove);
  if (!fullRemove) {
    return;
  }
  try {
    fs.writeFileSync(MARKER, "1", { mode: 0o600 });
  } catch {
    // Soft-fail.
  }
  runTrim(["daemon", "uninstall"]);
  runTrim(["stop"]);
  stopWindowsTrimProcesses(true);
  unlockVendorBinaryForUninstall();
}

main();
