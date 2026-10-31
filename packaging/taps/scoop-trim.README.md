# usetrim/scoop-trim (create this repo on GitHub)

Public Scoop bucket. GoReleaser pushes the scoop manifest when `SCOOP_BUCKET_TOKEN` is set.

1. Create public repo `usetrim/scoop-trim`.
2. PAT with `contents: write` on that repo.
3. Add Actions secret `SCOOP_BUCKET_TOKEN` on `usetrim/trim`.
4. Push a `v*` tag.
5. Users:

```text
scoop bucket add trim https://github.com/usetrim/scoop-trim
scoop install trim
trim version
# One-time Deep deps (local llmlingua - Scoop does not pip-install it)
trim compress --bootstrap
# Expect a green success banner when ready
```

Deep Mode runtime: https://use-trim.com/docs/installation#deep-runtime

Until the token exists, release still succeeds (`skip_upload` when unset).
