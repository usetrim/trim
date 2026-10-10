#!/usr/bin/env node
"use strict";

/**
 * Thin shim: exec the Go binary under vendor/. Self-heals if postinstall was skipped.
 */

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const { install } = require("../scripts/install");
const { resolvePlatform } = require("../scripts/platform");

async function main() {
  const { binaryName } = resolvePlatform();
  const binPath = path.join(__dirname, "..", "vendor", binaryName);

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
