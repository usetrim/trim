# Distribution launch checklist (npm + Homebrew + Go)

Channels **1 (npm)**, **2 (Homebrew)**, **4 (Go clone install)** plus curl/irm installers.

## 1) npm (`@usetrim/trim`) - Trusted Publishing

Code: `packaging/npm/@usetrim/trim/`  
CI: `.github/workflows/publish-npm.yml` (OIDC; no long-lived publish token)

### Live status

- [x] Org **`usetrim`**, account **2FA**
- [x] Package public: https://www.npmjs.com/package/@usetrim/trim (`latest` **0.1.3+**)
- [x] Trusted Publisher **Valid** → `usetrim/trim` / `publish-npm.yml` / allow `npm publish`
- [x] Publishing access: require 2FA and **disallow bypass 2FA tokens**
- [x] Bootstrap token `trim-github-actions` revoked; remove GitHub `NPM_TOKEN` if still present
- [x] Accidental `0.0.0-stage` unpublished

### Verify

```bash
npm install -g @usetrim/trim
trim version
```

## 2) Homebrew (`usetrim/tap/trim`)

- [x] Public repo https://github.com/usetrim/homebrew-tap
- [x] PAT → Actions secret `HOMEBREW_TAP_TOKEN` on `usetrim/trim`
- [x] GoReleaser uploaded `Formula/trim.rb` (confirmed on `v0.1.3`)
- [x] Sync tap README from `packaging/taps/homebrew-tap/README.md` (matches live `usetrim/homebrew-tap` `main`)
- [ ] Smoke on Mac/Linux: `brew install usetrim/tap/trim && trim version`

### Verify

```bash
brew install usetrim/tap/trim
trim version
```

## 4) Go (from clone)

```bash
git clone https://github.com/usetrim/trim.git
cd trim/cli && go install ./cmd/trim
```

Remote `go install github.com/usetrim/trim/cli/cmd/trim@v…` stays unsupported while `replace ../server` exists in `cli/go.mod`.

## Already live (keep advertising)

```bash
curl -fsSL https://use-trim.com/install.sh | sh
irm https://use-trim.com/install.ps1 | iex
```

## IDE extension (`usetrim.trim-ide`)

| Channel | Status |
| --- | --- |
| VS Code Marketplace | Live - search **Trim IDE** (publisher `usetrim`); CI `VSCE_PAT` optional (manual upload OK) |
| Open VSX | Live **1.0.0** - installable; namespace verification + Trusted Publishing OIDC still pending Eclipse claim |

Operator runbook: `extensions/trim-ide/PUBLISH.md`  
Workflow: `.github/workflows/publish-trim-ide.yml` (GitHub Environment `open-vsx`)
