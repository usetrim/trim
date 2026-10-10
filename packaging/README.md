# Package manager publish (npm / Homebrew / Scoop / winget)

GoReleaser already emits brew / scoop / winget manifests from `.goreleaser.yaml`.
Upload to public taps is **ops**, not automatic, until tokens and remote repos exist.
Every `v*` tag also uploads those manifests as GitHub Release assets (no token required)
so you can copy them into remotes before secrets are wired.

**npm** ships as `@usetrim/trim` under `packaging/npm/` (downloads the same Release binaries
with checksum verification). See `packaging/npm/README.md`.

## Prerequisites (you create once)

Copy-paste runbooks live under `packaging/taps/`.
Example formula / scoop / winget shapes (placeholders only) live under `packaging/taps/examples/`.

| Target | Create | Secret on GitHub Actions | Runbook |
|--------|--------|---------------------------|---------|
| **npm** | npmjs.org org `usetrim` + Trusted Publisher on `@usetrim/trim` (live) | OIDC via `publish-npm.yml` (no long-lived token) | `npm/README.md` + `DISTRIBUTION-LAUNCH.md` |
| Homebrew | Public tap `usetrim/homebrew-tap` + `Formula/trim.rb` (live) | `HOMEBREW_TAP_TOKEN` (PAT Contents R/W on tap only) | `taps/homebrew-tap.README.md` + user README in `taps/homebrew-tap/` |
| Scoop | Public bucket `usetrim/scoop-trim` | `SCOOP_BUCKET_TOKEN` | `taps/scoop-trim.README.md` |
| winget | Fork of `microsoft/winget-pkgs` (or PR flow) | `WINGET_GITHUB_TOKEN` | `taps/winget-pkgs.README.md` |

Leave secrets unset until those remotes exist. `.goreleaser.yaml` sets
`skip_upload: '{{ not (isEnvSet "…_TOKEN") }}'` for brew / scoop / winget, so
empty secrets never fail the release; manifests still appear under `dist/` and on the Release page.

## After each `v*` tag

1. Tag push runs `.github/workflows/release.yml` (binaries + Cosign + SBOM + treesitter/Zig).
2. Package-manager manifests are uploaded onto the Release (manual publish path without tokens).
3. If tap tokens are set, Homebrew / Scoop formulas update automatically.
4. winget uploads when `WINGET_GITHUB_TOKEN` is set (fork ready). Otherwise use Release assets or `dist/` for a manual PR.
5. Apple / Windows signing jobs soft-skip until cert secrets are set (see below).
6. Frozen Deep Mode: **default on** every tag via `deep-attach` → `build-trim-deep.yml` (`continue-on-error` if torch OOM). Opt out: repository variable `TRIM_SKIP_DEEP_ON_RELEASE=true`. Manual fallback: Actions → **Build trim-deep** with `upload_to_release=<tag>`.

## Local verify (no publish)

```bash
cd cli
goreleaser release --snapshot --clean --skip=publish
ls dist/
```

## End-user install (after taps / npm are live)

```bash
# npm (JS / AI / Node developers)
npm install -g @usetrim/trim

# Homebrew (macOS / Linux) - requires usetrim/homebrew-tap + HOMEBREW_TAP_TOKEN
brew install usetrim/tap/trim

# Scoop / winget (optional; Scoop skipped in the current launch set)
scoop bucket add trim https://github.com/usetrim/scoop-trim
scoop install trim
winget install Trim.CLI
```

### Go developers (from a clone)

`go install github.com/usetrim/trim/cli/cmd/trim@…` is **not** supported yet:
`cli/go.mod` uses a local `replace` for `../server`, which `go install …@version` rejects.

```bash
git clone https://github.com/usetrim/trim.git
cd trim/cli
go install ./cmd/trim
```

Prefer Release binaries, Homebrew, npm, or the curl/irm installers for end users.

## End-user uninstall

Worldwide one-shot (full purge by default - data dirs, Deep caches from site_messages, binary/PATH, Windows Apps entry):

```bash
trim uninstall
# Opt out of deleting local data / Deep caches / binary:
trim uninstall --keep-data
```

Helpers (delegate to CLI only):

```bash
curl -fsSL https://use-trim.com/uninstall.sh | sh
# Windows PowerShell:
irm https://use-trim.com/uninstall.ps1 | iex
```

On Windows, Settings → Apps → Trim also runs `trim uninstall` (same full purge of `%USERPROFILE%\.trim` and Deep model weights).

Package managers (after `trim uninstall` so local caches are cleared):

```bash
brew uninstall trim
scoop uninstall trim
winget uninstall Trim.CLI
```

Product docs: `/docs/uninstall`.

Until taps are live, use:

```bash
curl -fsSL https://use-trim.com/install.sh | sh
# Tree-sitter: TRIM_WITH_TREESITTER=1 ...
# Zig grammar (linux/darwin/windows amd64 + darwin arm64): TRIM_WITH_ZIG_TREESITTER=1 ...
# Frozen deep: TRIM_WITH_DEEP=1 ...
```

## Signing (paid; CI scaffolding in release.yml)

Jobs `sign-darwin` and `sign-windows` soft-skip when secrets are empty.

| Platform | Secrets |
|----------|---------|
| Apple notarization | `APPLE_DEVELOPER_ID_APPLICATION`, `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_SPECIFIC_PASSWORD` |
| Windows EV | `WINDOWS_CERT_PFX_BASE64`, `WINDOWS_CERT_PASSWORD` |

Package managers do not replace code signing. Unsigned binaries still work via curl install; Gatekeeper / SmartScreen need certs.

## Enterprise HA

See `packaging/enterprise-ha.md` and `docker-compose.prod.yml`.
