# Homebrew tap ops (`usetrim/homebrew-tap`)

Public tap. GoReleaser pushes `Formula/trim.rb` when `HOMEBREW_TAP_TOKEN` is set on **`usetrim/trim`**.

Canonical user-facing README (copy into the tap repo): `packaging/taps/homebrew-tap/README.md`.

## One-time setup (done for launch)

1. Public repo **`usetrim/homebrew-tap`**
2. Fine-grained PAT with **Contents: Read and write** on that repo only
3. Actions secret **`HOMEBREW_TAP_TOKEN`** on **`usetrim/trim`**
4. Tag `v*` so Release CLI uploads the formula

## After each release

- Push a new `v*` tag on `usetrim/trim`
- Confirm Actions **Release CLI** is green
- Confirm `Formula/trim.rb` version matches the tag in the tap

## User install

```bash
brew install usetrim/tap/trim
trim version
```

## Checklist

- [x] `usetrim/homebrew-tap` exists and is public
- [x] `HOMEBREW_TAP_TOKEN` set on `usetrim/trim`
- [x] `v0.1.3` Release published formula
- [x] `Formula/trim.rb` present (version 0.1.3)
- [x] Live tap `README.md` matches canonical `packaging/taps/homebrew-tap/README.md`
- [ ] Smoke: `brew install usetrim/tap/trim` on macOS or Linux

## Keep README in sync

Canonical copy lives in **`usetrim/trim`**: `packaging/taps/homebrew-tap/README.md`.  
When you change it, update https://github.com/usetrim/homebrew-tap/blob/main/README.md to match (web edit or clone the tap and PR). Do **not** hand-edit `Formula/*.rb` in the tap.
