//go:build !treesitter || !cgo

package trimmer

// TreeSitterAvailable is false for default CGO-free builds.
func TreeSitterAvailable() bool { return false }

// tryTreeSitterSkeletonize is a no-op when the treesitter CGO build tag is off.
// Default builds stay CGO-free (Go stdlib AST + heuristics).
// customQueries require a treesitter build; callers fail closed when queries are set.
func tryTreeSitterSkeletonize(lang, source string, customQueries []string) (string, bool) {
	_ = lang
	_ = source
	_ = customQueries
	return "", false
}
