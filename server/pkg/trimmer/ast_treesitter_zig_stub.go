//go:build treesitter && cgo && !zigts

package trimmer

import sitter "github.com/smacker/go-tree-sitter"

// zigTreeSitterLanguage is nil unless built with -tags treesitter,zigts.
// Without zigts, Zig uses the braced heuristic in skeletonizeZig.
func zigTreeSitterLanguage() *sitter.Language {
	return nil
}
