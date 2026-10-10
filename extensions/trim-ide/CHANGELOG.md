# Changelog

All notable changes to the Trim IDE extension are documented in this file.

## [1.0.2] - 2026-10-09

### Changed
- Replace placeholder Marketplace screenshots with live VS Code Trim IDE captures (`screenshot-setup.png`, `screenshot-status.png`)
- Add capture pipeline: `media/SCREENSHOTS.md` + `scripts/_process_ext_screenshots.py` (fit+pad to 1280×800)
- Clarify Marketplace README / SUPPORT for end users (less internal publish jargon)

## [1.0.1] - 2026-10-09

### Fixed
- Marketplace / Open VSX Details images: use absolute GitHub raw URLs (and vsce `--baseImagesUrl` for monorepo path `extensions/trim-ide`). Relative `media/*.png` was rewritten to `…/raw/HEAD/media/…` (404) because vsce ignores `repository.directory`.

## [1.0.0] - 2026-10-06

### Added
- Marketplace packaging readiness: gallery banner, screenshots, Getting Started walkthrough
- Privacy / Security / Support docs; privacy URL `https://use-trim.com/privacy`
- Localized `package.nls` for es, de, fr, ja, zh-cn, pt-br (EN remains source of truth for site_messages sync)
- Unit tests for URL/header/counter helpers (`npm test`)
- CI workflow `.github/workflows/publish-trim-ide.yml` (package on every change; publish on release / manual `publish=true` - Marketplace optional `VSCE_PAT`; Open VSX PAT bootstrap once then Trusted Publishing OIDC)
- Brand media generated from shared Trim masters (`media/icon.png` + light/dark marks)

### Changed
- Version **1.0.0** for public store signaling (Marketplace + Open VSX live; Open VSX Trusted Publishing after verified namespace)
- Repository metadata → `https://github.com/usetrim/trim` (directory `extensions/trim-ide`)
- Categories: Machine Learning, Visualization, Other
- README install path prefers Marketplace / Open VSX; VSIX remains supported for sideload

## [0.1.0] - 2026-10-01

### Added
- Always-on IDE proxy autostart (`trim start`) with cloud preference follow / local override
- Telemetry flush to `POST /api/v1/me/events` (tab shown/accepted, optional AI LOC)
- Secret Storage for API keys; `X-Hardware-UUID` + `Trim: Copy Hardware ID`
- Fail-closed UI chrome from `GET /api/v1/public/ide-chrome` with last-known-good cache
- Pending counter persistence across reload/crash until a successful flush
- `User-Agent: TrimIDE/<version>` for AntiFraud Protect compatibility
- Quota exhausted upgrade toast via `/me/quota` and 402 payloads
- Fail-closed config/copy from site_messages only (no invented flush default, no invented backoff)

### Notes
- IDE hardware IDs are **not** the same as CLI fingerprints - register this IDE device under Dashboard → Settings → API keys
- LOC tracking requires `trim.trackDocumentEdits: true` **and** `trim.minLinesForAiHeuristic >= 1`
- `trim.autoFlushSeconds` package default is `0` (off); operators set an enabled value inside cloud `IDE_AUTO_FLUSH_*` bounds
