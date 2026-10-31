# `@usetrim/trim`

Official npm installer for the **Trim** CLI. On install it downloads the matching
binary from [GitHub Releases](https://github.com/usetrim/trim/releases), verifies
**SHA-256** against `checksums.txt`, and exposes the `trim` command.

```bash
npm install -g @usetrim/trim
trim version
trim login
trim start
# everyday Always-on (preference + best-effort OS logon; no admin on normal Windows)
trim autostart enable
# one-time Deep deps on this PC (Microsoft LLMLingua via pip - not Trim cloud)
trim compress --bootstrap
# success prints a green banner; then optional:
# trim compress README.md --deep
trim help compress
trim help autostart
trim help daemon
```

No environment variables required for Trim Cloud. Self-host overrides: see the Trim repo `cli/.env.example`.

Always-on docs: [Always-on with IDE](https://use-trim.com/docs/concepts/always-on) · [`trim autostart`](https://use-trim.com/docs/cli/autostart) · [Installation](https://use-trim.com/docs/installation).

Or without a global install:

```bash
npx @usetrim/trim version
```

## How it works

1. `postinstall` fetches `trim_<os>_<arch>.(tar.gz|zip)` for package version `vX.Y.Z`
2. Verifies checksums from the same release
3. Places `trim` / `trim.exe` **and** Deep helpers (`optimizer.py`, `requirements-deep.txt`, `requirements.txt`) under `vendor/`
4. `bin/trim.js` execs that binary (self-heals binary **and** Deep sidecars if `npm install --ignore-scripts` was used)
5. On Windows Always-on (`trim autostart enable`), Trim copies those Deep helpers into `%LOCALAPPDATA%\Trim\bin` beside the stable `trim.exe`
6. First Deep use: `trim compress --bootstrap` (local `pip install` of Microsoft LLMLingua; **not** done at npm install time). Success prints a green banner like `trim start`. Same step after brew / curl / Scoop / winget.

Same assets as `https://use-trim.com/install.sh` (default CGO-free build).

## Environment

| Variable | Meaning |
|----------|---------|
| `TRIM_GITHUB_REPO` | Override repo (`owner/name`), default `usetrim/trim` |
| `TRIM_NPM_TAG` | Override release tag (default `v` + `package.json` version) |
| `TRIM_NPM_SKIP_DOWNLOAD=1` | Skip download (packaging/CI only) |

## Prefer other installers?

- **curl:** `curl -fsSL https://use-trim.com/install.sh \| sh`
- **Homebrew (when tap is live):** `brew install usetrim/tap/trim`
- **Windows:** `irm https://use-trim.com/install.ps1 \| iex`

## Uninstall

One shot (preferred):

```bash
trim uninstall
```

That stops Trim processes, deletes local data/Deep caches, renames the locked `vendor/trim.exe` on Windows, and a detached post-exit helper deletes it then runs `npm uninstall -g @usetrim/trim`. A package-local `.trim-uninstalling` marker blocks this shim from re-downloading the binary mid-uninstall.

Fallback only if `trim` still resolves afterward:

```bash
npm uninstall -g @usetrim/trim
```

On Windows, Always-on registers **Startup Run → `%LOCALAPPDATA%\Trim\bin\trim.exe`** (a stable copy outside this package). `trim uninstall` also removes that tree.

`preuninstall` always unlocks this package’s `vendor\trim.exe`. On a real `npm uninstall` (not upgrade) it writes `.trim-uninstalling`, best-effort runs `daemon uninstall` + `stop`, then deletes or renames a locked vendor binary (reboot-pending delete on Windows). Prefer `trim uninstall` for a full machine purge (data dirs, Deep caches, ARP, scheduled binary delete); plain `npm uninstall -g` is not the same as `trim uninstall`.

## License

MIT - see the Trim repository.
