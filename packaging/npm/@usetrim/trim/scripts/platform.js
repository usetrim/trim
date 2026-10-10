"use strict";

/**
 * Map Node platform/arch to GoReleaser asset names (default CGO-free build).
 * Must stay aligned with scripts/install.sh and .goreleaser.yaml archives.
 */

function resolvePlatform(nodePlatform = process.platform, nodeArch = process.arch) {
  let os;
  switch (nodePlatform) {
    case "darwin":
      os = "darwin";
      break;
    case "linux":
      os = "linux";
      break;
    case "win32":
      os = "windows";
      break;
    default:
      throw new Error(
        `Unsupported platform "${nodePlatform}". Supported: darwin, linux, win32.`,
      );
  }

  let arch;
  switch (nodeArch) {
    case "x64":
      arch = "amd64";
      break;
    case "arm64":
      arch = "arm64";
      break;
    default:
      throw new Error(
        `Unsupported architecture "${nodeArch}". Supported: x64, arm64.`,
      );
  }

  const ext = os === "windows" ? "zip" : "tar.gz";
  const binaryName = os === "windows" ? "trim.exe" : "trim";
  const asset = `trim_${os}_${arch}.${ext}`;

  return { os, arch, ext, binaryName, asset };
}

module.exports = { resolvePlatform };
