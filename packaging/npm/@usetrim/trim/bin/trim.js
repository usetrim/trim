#!/usr/bin/env node
"use strict";

/**
 * Thin shim: exec the Go binary under vendor/. Self-heals if postinstall was skipped.
 */

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const os = require("node:os");
const { install } = require("../scripts/install");
const { resolvePlatform } = require("../scripts/platform");

async function main() {
  const { binaryName } = resolvePlatform();
  const pkgRoot = path.join(__dirname, "..");
  const marker = path.join(pkgRoot, ".trim-uninstalling");
  // Always heal sticky TEMP leftovers from older builds that wrote
  // %TEMP%/trim-uninstalling and never cleared it (even when refusing below).
  try {
    fs.unlinkSync(path.join(os.tmpdir(), "trim-uninstalling"));
  } catch {
    // ignore missing / permission
  }
  // Package-local marker only blocks self-heal mid-uninstall.
  if (fs.existsSync(marker)) {
    console.error("@usetrim/trim: uninstall in progress (or incomplete). Refusing to run or re-download.");
    console.error("If Trim should be gone: npm uninstall -g @usetrim/trim");
    process.exit(1);
  }

  const binPath = path.join(pkgRoot, "vendor", binaryName);
  const asidePath = binPath + ".uninstalling";
  // If Windows renamed the running binary mid-uninstall, never self-heal over it.
  if (fs.existsSync(asidePath)) {
    console.error("@usetrim/trim: uninstall in progress (renamed binary present). Refusing to re-download.");
    console.error("If Trim should be gone: npm uninstall -g @usetrim/trim");
    process.exit(1);
  }

  if (!fs.existsSync(binPath)) {
    console.error("@usetrim/trim: binary missing - downloading from GitHub Releases…");
    await install();
  }

  if (!fs.existsSync(binPath)) {
    console.error("@usetrim/trim: binary still missing after install.");
    console.error("Try: curl -fsSL https://use-trim.com/install.sh | sh");
    process.exit(1);
  }

  if (process.platform !== "win32") {
    try {
      fs.chmodSync(binPath, 0o755);
    } catch {
      // ignore
    }
  }

  const result = spawnSync(binPath, process.argv.slice(2), {
    stdio: "inherit",
    env: process.env,
    windowsHide: true,
  });

  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }
  process.exit(result.status === null ? 1 : result.status);
}

main().catch((err) => {
  console.error(err.message || err);
  process.exit(1);
});
