//go:build treesitter && cgo && zigts

package trimmer

import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
	tree_sitter_zig "github.com/tree-sitter-grammars/tree-sitter-zig/bindings/go"
)

// zigTreeSitterLanguage loads the official Zig grammar via tree-sitter-zig
// (not shipped in smacker). Enable with: go build -tags "treesitter,zigts"
// and: go get github.com/tree-sitter-grammars/tree-sitter-zig@latest
func zigTreeSitterLanguage() *sitter.Language {
	ptr := tree_sitter_zig.Language()
	if ptr == nil {
		return nil
	}
	return sitter.NewLanguage(unsafe.Pointer(ptr))
}
