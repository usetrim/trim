# usetrim/homebrew-tap

Official [Homebrew](https://brew.sh) tap for **[Trim](https://use-trim.com)** - a local AI context optimization proxy (OpenAI-compatible).

Formulae in this repository are published automatically by [GoReleaser](https://goreleaser.com) from [`usetrim/trim`](https://github.com/usetrim/trim) on each `v*` release. Do not edit `Formula/*.rb` by hand.

## Install

```bash
brew install usetrim/tap/trim
```

Equivalent explicit form:

```bash
brew tap usetrim/tap
brew install trim
```

## Upgrade

```bash
brew update
brew upgrade trim
```

## Uninstall

```bash
brew uninstall trim
```

To remove the tap as well:

```bash
brew untap usetrim/tap
```

## Verify

```bash
trim version
```

## Requirements

- [Homebrew](https://brew.sh) on macOS or Linux
- Supported arches: `darwin` / `linux` · `amd64` / `arm64`

## Other installers

| Channel | Command |
|---------|---------|
| npm | `npm install -g @usetrim/trim` |
| curl (macOS/Linux) | `curl -fsSL https://use-trim.com/install.sh \| sh` |
| PowerShell (Windows) | `irm https://use-trim.com/install.ps1 \| iex` |

## Links

- Website & docs: https://use-trim.com
- Source: https://github.com/usetrim/trim
- Releases: https://github.com/usetrim/trim/releases
- Issues: https://github.com/usetrim/trim/issues
- npm: https://www.npmjs.com/package/@usetrim/trim

## License

Trim is [MIT](https://github.com/usetrim/trim/blob/main/LICENSE)-licensed.
