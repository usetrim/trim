#!/bin/sh
set -e

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

REPO="${TRIM_GITHUB_REPO:-usetrim/trim}"
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1)
if [ -z "$TAG" ]; then
  echo "Could not resolve latest release tag from ${REPO}"
  exit 1
fi

EXT="tar.gz"
BINARY_NAME="trim"
if [ "$OS" = "windows" ] || [ "$OS" = "mingw64_nt"* ]; then
  EXT="zip"
  BINARY_NAME="trim.exe"
fi

# Prefer Tree-sitter CGO build when TRIM_WITH_TREESITTER=1 (or true/yes).
# Asset names match .github/workflows/release.yml treesitter-* jobs.
# TRIM_WITH_ZIG_TREESITTER=1 selects treesitter+Zig grammar (linux amd64/arm64, darwin amd64/arm64, windows amd64).
WITH_TS="$(printf '%s' "${TRIM_WITH_TREESITTER:-}" | tr '[:upper:]' '[:lower:]')"
WITH_ZIG="$(printf '%s' "${TRIM_WITH_ZIG_TREESITTER:-}" | tr '[:upper:]' '[:lower:]')"
WITH_DEEP="$(printf '%s' "${TRIM_WITH_DEEP:-}" | tr '[:upper:]' '[:lower:]')"
ASSET="trim_${OS}_${ARCH}.${EXT}"
if [ "$WITH_ZIG" = "1" ] || [ "$WITH_ZIG" = "true" ] || [ "$WITH_ZIG" = "yes" ]; then
  case "${OS}_${ARCH}" in
    linux_amd64|linux_arm64)
      ASSET="trim_linux_${ARCH}_treesitter_zig.tar.gz"
      EXT="tar.gz"
      WITH_TS="1"
      ;;
    darwin_amd64|darwin_arm64)
      ASSET="trim_darwin_${ARCH}_treesitter_zig.tar.gz"
      EXT="tar.gz"
      BINARY_NAME="trim"
      WITH_TS="1"
      ;;
    windows_amd64)
      ASSET="trim_windows_amd64_treesitter_zig.zip"
      EXT="zip"
      BINARY_NAME="trim.exe"
      WITH_TS="1"
      ;;
    *)
      echo "No treesitter+Zig release asset for ${OS}/${ARCH}."
      echo "Supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64."
      echo "Unset TRIM_WITH_ZIG_TREESITTER or use TRIM_WITH_TREESITTER=1."
      exit 1
      ;;
  esac
elif [ "$WITH_TS" = "1" ] || [ "$WITH_TS" = "true" ] || [ "$WITH_TS" = "yes" ]; then
  case "${OS}_${ARCH}" in
    darwin_amd64|darwin_arm64)
      ASSET="trim_darwin_${ARCH}_treesitter.tar.gz"
      EXT="tar.gz"
      BINARY_NAME="trim"
      ;;
    linux_arm64)
      ASSET="trim_linux_arm64_treesitter.tar.gz"
      EXT="tar.gz"
      ;;
    linux_amd64)
      ASSET="trim_linux_amd64_treesitter.tar.gz"
      EXT="tar.gz"
      ;;
    windows_amd64)
      ASSET="trim_windows_amd64_treesitter.zip"
      EXT="zip"
      BINARY_NAME="trim.exe"
      ;;
    *)
      echo "No treesitter release asset for ${OS}/${ARCH}."
      echo "Unset TRIM_WITH_TREESITTER to install the default CGO-free archive."
      exit 1
      ;;
  esac
fi

BASE="https://github.com/${REPO}/releases/download/${TAG}"
URL="${BASE}/${ASSET}"
CHECKSUMS_URL="${BASE}/checksums.txt"

echo "Installing Trim ${TAG} for ${OS}/${ARCH} (asset: ${ASSET})..."

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# If treesitter asset 404s, fail closed when explicitly requested; do not silently
# fall back to CGO-free (would hide missing release artifacts).
if ! curl -fsSL "$URL" -o "trim_archive.${EXT}"; then
  echo "Failed to download ${URL}"
  if [ "$WITH_TS" = "1" ] || [ "$WITH_TS" = "true" ] || [ "$WITH_TS" = "yes" ]; then
    echo "TRIM_WITH_TREESITTER was set; treesitter asset must exist on the release."
    echo "Unset TRIM_WITH_TREESITTER to install the default CGO-free archive."
  fi
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi

# Verify SHA256 against release checksums.txt (required; no silent skip).
if ! curl -fsSL "$CHECKSUMS_URL" -o checksums.txt; then
  echo "Failed to download checksums.txt from ${CHECKSUMS_URL}"
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi

EXPECTED=$(grep -E "[[:space:]]${ASSET}$" checksums.txt | awk '{print $1}' | head -n1)
if [ -z "$EXPECTED" ]; then
  echo "No checksum entry for ${ASSET} in checksums.txt"
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "trim_archive.${EXT}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "trim_archive.${EXT}" | awk '{print $1}')
else
  echo "Need sha256sum or shasum to verify the release archive"
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi

if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "Checksum mismatch for ${ASSET}"
  echo "  expected: ${EXPECTED}"
  echo "  actual:   ${ACTUAL}"
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi
echo "Checksum OK (${ACTUAL})"

# Optional Cosign verify: if a .sig is published for this asset, verification is required.
if curl -fsSL "${BASE}/${ASSET}.sig" -o "trim_archive.${EXT}.sig" 2>/dev/null; then
  if ! command -v cosign >/dev/null 2>&1; then
    echo "Release includes ${ASSET}.sig but cosign is not installed."
    echo "Install cosign (https://docs.sigstore.dev/cosign/installation/) and re-run."
    cd -
    rm -rf "$TMP_DIR"
    exit 1
  fi
  if ! cosign verify-blob --signature "trim_archive.${EXT}.sig" "trim_archive.${EXT}"; then
    echo "Cosign signature verification failed for ${ASSET}"
    cd -
    rm -rf "$TMP_DIR"
    exit 1
  fi
  echo "Cosign signature verified"
fi

if [ "$EXT" = "zip" ]; then
  unzip -q "trim_archive.${EXT}"
else
  tar -xzf "trim_archive.${EXT}"
fi

# Treesitter archives may nest the binary under a named folder.
if [ ! -f "$BINARY_NAME" ]; then
  FOUND=$(find . -type f -name "$BINARY_NAME" 2>/dev/null | head -n1)
  if [ -n "$FOUND" ]; then
    mv "$FOUND" "./$BINARY_NAME"
  fi
fi
if [ ! -f "$BINARY_NAME" ]; then
  echo "Archive extracted but ${BINARY_NAME} was not found"
  cd -
  rm -rf "$TMP_DIR"
  exit 1
fi

INSTALL_DIR="${TRIM_INSTALL_DIR:-/usr/local/bin}"
install_file() {
  src="$1"
  dest="$2"
  if [ ! -f "$src" ]; then
    return 0
  fi
  if [ -w "$INSTALL_DIR" ]; then
    mv "$src" "$dest"
  else
    sudo mv "$src" "$dest"
  fi
}

if [ ! -w "$INSTALL_DIR" ]; then
  echo "Requires elevated privileges to install into ${INSTALL_DIR}"
fi

install_file "$BINARY_NAME" "$INSTALL_DIR/trim"
install_file optimizer.py "$INSTALL_DIR/optimizer.py"
install_file requirements-deep.txt "$INSTALL_DIR/requirements-deep.txt"
install_file requirements.txt "$INSTALL_DIR/requirements.txt"

# Optional frozen Deep Mode binary (PyInstaller). Fail closed when TRIM_WITH_DEEP=1 and missing.
if [ "$WITH_DEEP" = "1" ] || [ "$WITH_DEEP" = "true" ] || [ "$WITH_DEEP" = "yes" ]; then
  DEEP_ASSET="trim-deep-${OS}-${ARCH}"
  if [ "$OS" = "windows" ] || [ "$OS" = "mingw64_nt"* ]; then
    DEEP_ASSET="trim-deep-windows-amd64.exe"
  fi
  DEEP_URL="${BASE}/${DEEP_ASSET}"
  echo "Fetching frozen Deep Mode binary ${DEEP_ASSET}..."
  if curl -fsSL "$DEEP_URL" -o "$TMP_DIR/${DEEP_ASSET}"; then
    chmod +x "$TMP_DIR/${DEEP_ASSET}" 2>/dev/null || true
    install_file "$TMP_DIR/${DEEP_ASSET}" "$INSTALL_DIR/trim-deep"
    echo "trim-deep installed beside trim (on-machine Deep Mode binary; no pip bootstrap)."
  else
    echo "Failed to download ${DEEP_URL}"
    echo "TRIM_WITH_DEEP was set; freeze trim-deep via Actions → Build trim-deep (upload_to_release=${TAG})."
    echo "Or unset TRIM_WITH_DEEP and use: trim compress file.txt --deep (local dependency bootstrap)."
    cd -
    rm -rf "$TMP_DIR"
    exit 1
  fi
fi

cd -
rm -rf "$TMP_DIR"

# Privacy-light install beacon (country from CDN CF-IPCountry only on the server).
# Opt out: DO_NOT_TRACK=1 or TRIM_TELEMETRY_DISABLED=1. Never fails the install.
trim_install_beacon() {
  if [ "${DO_NOT_TRACK:-}" = "1" ] || [ "${TRIM_TELEMETRY_DISABLED:-}" = "1" ]; then
    return 0
  fi
  _api="${TRIM_API_BASE_URL:-https://api.use-trim.com}"
  _api=$(printf '%s' "$_api" | sed 's:/*$::')
  _path="${1:-/install.sh}"
  # Fail soft: install succeeded even if beacon is unreachable.
  curl -fsS -X POST -m 3 \
    "${_api}/api/v1/public/install-hit?path=${_path}" \
    >/dev/null 2>&1 || true
}
trim_install_beacon "/install.sh"

echo "Trim installed. Run: trim start"
echo "Then: trim login"
if [ "$WITH_ZIG" = "1" ] || [ "$WITH_ZIG" = "true" ] || [ "$WITH_ZIG" = "yes" ]; then
  echo "Enhanced grammar build installed (includes Zig)."
elif [ "$WITH_TS" = "1" ] || [ "$WITH_TS" = "true" ] || [ "$WITH_TS" = "yes" ]; then
  echo "Enhanced grammar build installed."
else
  echo "Default build installed. For the enhanced grammar build: TRIM_WITH_TREESITTER=1 curl -fsSL ... | sh"
  echo "For Zig grammar: TRIM_WITH_ZIG_TREESITTER=1 curl -fsSL ... | sh"
fi
echo "Deep Mode (on-machine file compress): trim compress file.txt --deep"
echo "  First Deep run may install local dependencies on this machine."
echo "  Optional frozen binary: TRIM_WITH_DEEP=1 (requires trim-deep-* on the release)."
