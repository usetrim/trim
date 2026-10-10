package trimmer

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// SkeletonizeGoAST uses the Go standard library parser (no CGO) to clear
// function and method bodies while keeping signatures, types, and exports.
// Falls back to braced heuristics when the source does not parse.
func SkeletonizeGoAST(source string) string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "input.go", source, parser.ParseComments)
	if err != nil {
		return skeletonizeBraced(source, []string{"func "})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body != nil {
				x.Body = &ast.BlockStmt{List: nil}
			}
		case *ast.FuncLit:
			if x.Body != nil {
				x.Body = &ast.BlockStmt{List: nil}
			}
		}
		return true
	})

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return skeletonizeBraced(source, []string{"func "})
	}
	out := buf.String()
	out = strings.ReplaceAll(out, "{\n}", "{ /* body trimmed by trim */ }")
	out = strings.ReplaceAll(out, "{}", "{ /* body trimmed by trim */ }")
	return out
}
