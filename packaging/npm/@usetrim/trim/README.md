# `@usetrim/trim`

Official npm installer for the **Trim** CLI. On install it downloads the matching
binary from [GitHub Releases](https://github.com/usetrim/trim/releases), verifies
**SHA-256** against `checksums.txt`, and exposes the `trim` command.

```bash
npm install -g @usetrim/trim
trim version
```

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

## License

MIT - see the Trim repository.
