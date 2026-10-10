# usetrim/winget-pkgs (fork of microsoft/winget-pkgs)

1. Fork `microsoft/winget-pkgs` to `usetrim/winget-pkgs` (or your org).
2. PAT with `contents: write` on the fork.
3. Add Actions secret `WINGET_GITHUB_TOKEN` on `usetrim/trim`.
4. Push a `v*` tag. GoReleaser uploads winget manifests when the token is set.
5. Open / merge the PR into upstream winget-pkgs per Microsoft's process.
6. Users: `winget install Trim.CLI` (exact id follows publisher in `.goreleaser.yaml`).

Until the token exists, manifests land under `dist/` only (`skip_upload` when unset).
