#!/usr/bin/env sh
# Fetch optional Tree-sitter CGO deps and print the build command.
# Requires Go + a C toolchain (gcc, clang, or MSVC).
set -e
cd "$(dirname "$0")/.."
go get github.com/smacker/go-tree-sitter@v0.0.0-20240827094217-dd81d9e9be82
if [ "${ENABLE_ZIG_TREESITTER:-0}" = "1" ]; then
  go get github.com/tree-sitter-grammars/tree-sitter-zig@latest
  go mod tidy
  echo "Build with: CGO_ENABLED=1 go build -tags \"treesitter,zigts\" -o trim-api ./cmd/api"
  echo "Docker:     docker build --build-arg ENABLE_TREESITTER=1 --build-arg ENABLE_ZIG_TREESITTER=1 -t trim-api ."
else
  go mod tidy
  echo "Build with: CGO_ENABLED=1 go build -tags treesitter -o trim-api ./cmd/api"
  echo "Docker:     docker build --build-arg ENABLE_TREESITTER=1 -t trim-api ."
  echo "Optional Zig grammar: ENABLE_ZIG_TREESITTER=1 $0"
fi
