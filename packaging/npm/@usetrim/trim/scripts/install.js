"use strict";

/**
 * Download + SHA-256 verify the matching GitHub Release binary into vendor/.
 * Used by postinstall and by bin/trim.js self-heal when --ignore-scripts was used.
 *
 * Env:
 *   TRIM_GITHUB_REPO   default usetrim/trim
 *   TRIM_NPM_TAG       override release tag (e.g. v0.1.0); default v${package.version}
 *   TRIM_NPM_SKIP_DOWNLOAD=1  skip (CI packaging only)
 */

const crypto = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const { resolvePlatform } = require("./platform");

const ROOT = path.join(__dirname, "..");
const VENDOR = path.join(ROOT, "vendor");
const PKG = require("../package.json");

function repo() {
  return process.env.TRIM_GITHUB_REPO || "usetrim/trim";
}

function tag() {
  if (process.env.TRIM_NPM_TAG) return process.env.TRIM_NPM_TAG;
  return `v${PKG.version}`;
}

async function fetchBuffer(url) {
  const res = await fetch(url, {
    headers: { "User-Agent": "trim-npm-install/@usetrim/trim" },
    redirect: "follow",
  });
  if (!res.ok) {
    throw new Error(`GET ${url} failed: HTTP ${res.status}`);
  }
  return Buffer.from(await res.arrayBuffer());
}

function sha256(buf) {
  return crypto.createHash("sha256").update(buf).digest("hex");
}

function expectedChecksum(checksumsTxt, assetName) {
  const lines = checksumsTxt.split(/\r?\n/);
  for (const line of lines) {
    const m = line.trim().match(/^([a-fA-F0-9]{64})\s+(\S+)$/);
    if (!m) continue;
    if (m[2] === assetName || m[2].endsWith(`/${assetName}`)) {
      return m[1].toLowerCase();
    }
  }
  throw new Error(`Asset ${assetName} not found in checksums.txt`);
}

function extractArchive(archivePath, destDir, ext) {
  fs.mkdirSync(destDir, { recursive: true });
  if (ext === "zip") {
    // Windows 10+ and modern macOS/Linux ship bsdtar as `tar`.
    execFileSync("tar", ["-xf", archivePath, "-C", destDir], { stdio: "inherit" });
  } else {
    execFileSync("tar", ["-xzf", archivePath, "-C", destDir], { stdio: "inherit" });
  }
}

function findBinary(extractRoot, binaryName) {
  const direct = path.join(extractRoot, binaryName);
  if (fs.existsSync(direct) && fs.statSync(direct).isFile()) return direct;

  const entries = fs.readdirSync(extractRoot, { withFileTypes: true });
  for (const ent of entries) {
    if (!ent.isDirectory()) continue;
    const nested = path.join(extractRoot, ent.name, binaryName);
    if (fs.existsSync(nested) && fs.statSync(nested).isFile()) return nested;
  }
  throw new Error(`Extracted archive but did not find ${binaryName}`);
}

async function install() {
  if (process.env.TRIM_NPM_SKIP_DOWNLOAD === "1") {
    console.log("@usetrim/trim: TRIM_NPM_SKIP_DOWNLOAD=1 - skipping binary download");
    return;
  }

  const { asset, ext, binaryName } = resolvePlatform();
  const releaseTag = tag();
  const base = `https://github.com/${repo()}/releases/download/${releaseTag}`;
  const assetUrl = `${base}/${asset}`;
  const checksumsUrl = `${base}/checksums.txt`;

  console.log(`@usetrim/trim: installing ${releaseTag} (${asset})…`);

  const [archiveBuf, checksumsBuf] = await Promise.all([
    fetchBuffer(assetUrl),
    fetchBuffer(checksumsUrl),
  ]);

  const want = expectedChecksum(checksumsBuf.toString("utf8"), asset);
  const got = sha256(archiveBuf);
  if (got !== want) {
    throw new Error(`SHA-256 mismatch for ${asset}: expected ${want}, got ${got}`);
  }

  const tmp = fs.mkdtempSync(path.join(require("node:os").tmpdir(), "trim-npm-"));
  try {
    const archivePath = path.join(tmp, asset);
    fs.writeFileSync(archivePath, archiveBuf);
    const extractDir = path.join(tmp, "out");
    extractArchive(archivePath, extractDir, ext);
    const srcBin = findBinary(extractDir, binaryName);

    fs.mkdirSync(VENDOR, { recursive: true });
    const destBin = path.join(VENDOR, binaryName);
    fs.copyFileSync(srcBin, destBin);
    if (process.platform !== "win32") {
      fs.chmodSync(destBin, 0o755);
    }

    // Optional sidecar files from the archive (Deep Mode helpers).
    for (const name of ["optimizer.py", "requirements-deep.txt", "requirements.txt"]) {
      try {
        const src = findBinary(extractDir, name);
        fs.copyFileSync(src, path.join(VENDOR, name));
      } catch {
        // optional
      }
    }

    console.log(`@usetrim/trim: binary ready at ${destBin}`);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

if (require.main === module) {
  install().catch((err) => {
    console.error(`@usetrim/trim: install failed: ${err.message}`);
    console.error("Fallback: curl -fsSL https://use-trim.com/install.sh | sh");
    process.exit(1);
  });
}

module.exports = { install, tag, repo };
