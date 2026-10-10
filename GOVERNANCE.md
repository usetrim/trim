# Governance

## Maintainers

Maintainers review pull requests, triage issues, cut releases, and own production infrastructure for the hosted Trim control plane.

## Decision making

- Day-to-day changes land via pull request review.
- Breaking API or billing policy changes require maintainer consensus.
- Security issues follow `SECURITY.md` and are not discussed in public issues until a fix is available.

## Release process

1. Merge to the default branch through CI.
2. Tag `vX.Y.Z` to trigger GoReleaser for CLI binaries.
3. Deploy API and web from the same commit when possible.

## Code of conduct

Community participation follows `CODE_OF_CONDUCT.md`.
