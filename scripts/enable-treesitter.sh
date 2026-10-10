#!/usr/bin/env sh
# Root shim: Tree-sitter enable lives under server/ (module root for go get).
exec "$(cd "$(dirname "$0")/.." && pwd)/server/scripts/enable-treesitter.sh" "$@"
