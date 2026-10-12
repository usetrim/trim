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
3. Places the binary under `vendor/`
4. `bin/trim.js` execs that binary (self-heals if `npm install --ignore-scripts` was used)

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

Prefer the product uninstall first (stops proxy + OS logon, then you can remove the package):

```bash
trim uninstall
npm uninstall -g @usetrim/trim
```

On Windows, a running `trim.exe` (often `daemon run` from Always-on) locks the file. If `npm uninstall` prints `EPERM … unlink … vendor\trim.exe`, stop Trim then retry:

```powershell
# If trim is still on PATH:
trim daemon uninstall
trim stop

# Then retry package remove; clear npm rename leftovers if any:
npm uninstall -g @usetrim/trim
Remove-Item -Recurse -Force "$env:LOCALAPPDATA\nvm\*\node_modules\@usetrim" -ErrorAction SilentlyContinue
```

`preuninstall` best-effort runs `daemon uninstall` + `stop` (and clears a Windows lock on this package’s `vendor\trim.exe`) so normal upgrades/removals do not hit EPERM.

## License

MIT - see the Trim repository.
