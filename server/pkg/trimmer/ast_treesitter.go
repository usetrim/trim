//go:build treesitter && cgo

package trimmer

import (
	"context"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/cue"
	"github.com/smacker/go-tree-sitter/dockerfile"
	"github.com/smacker/go-tree-sitter/elixir"
	"github.com/smacker/go-tree-sitter/elm"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/groovy"
	"github.com/smacker/go-tree-sitter/hcl"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/lua"
	tsmarkdown "github.com/smacker/go-tree-sitter/markdown/tree-sitter-markdown"
	"github.com/smacker/go-tree-sitter/ocaml"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/protobuf"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/scala"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/svelte"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/toml"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
	"github.com/smacker/go-tree-sitter/yaml"
)

// TreeSitterAvailable is true for CGO treesitter builds.
func TreeSitterAvailable() bool { return true }

// tryTreeSitterSkeletonize uses real Tree-sitter grammars (CGO) to stub function
// and method bodies while keeping signatures, types, and imports intact.
// When customQueries is non-empty (from .trimrc custom_query), those queries replace
// the built-in body query for the language grammar (bible Mode 4 custom).
// Build with: CGO_ENABLED=1 go build -tags treesitter
// Dependencies (run once when enabling the tag):
//
//	go get github.com/smacker/go-tree-sitter@latest
func tryTreeSitterSkeletonize(lang, source string, customQueries []string) (string, bool) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	grammar, defaultQuery := treeSitterGrammar(lang)
	if grammar == nil {
		return "", false
	}
	query := defaultQuery
	if len(customQueries) > 0 {
		var parts []string
		for _, q := range customQueries {
			q = strings.TrimSpace(q)
			if q != "" {
				parts = append(parts, q)
			}
		}
		if len(parts) == 0 {
			return "", false
		}
		query = strings.Join(parts, "\n")
	}
	if query == "" {
		return "", false
	}
	out, err := skeletonizeWithTreeSitter([]byte(source), grammar, query)
	if err != nil || out == "" {
		return "", false
	}
	return out, true
}

func treeSitterGrammar(lang string) (*sitter.Language, string) {
	switch lang {
	case "go", "golang":
		return golang.GetLanguage(), `
(function_declaration body: (block) @body)
(method_declaration body: (block) @body)
(func_literal body: (block) @body)
`
	case "js", "jsx", "javascript", "mjs", "cjs":
		return javascript.GetLanguage(), jsBodyQuery
	case "ts", "typescript":
		return typescript.GetLanguage(), jsBodyQuery
	case "tsx":
		return tsx.GetLanguage(), jsBodyQuery
	case "py", "python":
		return python.GetLanguage(), `
(function_definition body: (block) @body)
`
	case "rs", "rust":
		return rust.GetLanguage(), `
(function_item body: (block) @body)
(closure_expression body: (block) @body)
`
	case "java":
		return java.GetLanguage(), `
(method_declaration body: (block) @body)
(constructor_declaration body: (constructor_body) @body)
`
	case "kt", "kotlin":
		return kotlin.GetLanguage(), `
(function_declaration (function_body) @body)
`
	case "c", "h":
		return c.GetLanguage(), `
(function_definition body: (compound_statement) @body)
`
	case "cpp", "cc", "cxx", "hpp", "c++":
		return cpp.GetLanguage(), `
(function_definition body: (compound_statement) @body)
(lambda_expression body: (compound_statement) @body)
`
	case "cs", "csharp":
		return csharp.GetLanguage(), `
(method_declaration body: (block) @body)
(constructor_declaration body: (block) @body)
`
	case "php":
		return php.GetLanguage(), `
(function_definition body: (compound_statement) @body)
(method_declaration body: (compound_statement) @body)
`
	case "rb", "ruby":
		return ruby.GetLanguage(), `
(method body: (body_statement) @body)
(singleton_method body: (body_statement) @body)
`
	case "swift":
		return swift.GetLanguage(), `
(function_declaration body: (function_body) @body)
`
	case "sql":
		// DerekStride/tree-sitter-sql via smacker bindings: stub CREATE FUNCTION /
		// PROCEDURE bodies and keep DDL signatures.
		return sql.GetLanguage(), `
(create_function statement: (_) @body)
(create_procedure statement: (_) @body)
(function_body) @body
`
	case "scala":
		return scala.GetLanguage(), `
(function_definition body: (block) @body)
(function_definition body: (indented_block) @body)
(function_definition body: (expression) @body)
`
	case "bash", "sh", "shell", "zsh", "ksh":
		return bash.GetLanguage(), `
(function_definition body: (compound_statement) @body)
`
	case "lua":
		return lua.GetLanguage(), `
(function_definition body: (block) @body)
(function_declaration body: (block) @body)
`
	case "elixir", "ex", "exs":
		return elixir.GetLanguage(), `
(anonymous_function body: (do_block) @body)
(call target: (identifier) @_name (#match? @_name "^(def|defp|defmacro|defmacrop)$") (do_block) @body)
`
	case "dockerfile", "docker":
		return dockerfile.GetLanguage(), `
(run_instruction (shell_command) @body)
(run_instruction (json_string_array) @body)
(cmd_instruction (shell_command) @body)
(cmd_instruction (json_string_array) @body)
(entrypoint_instruction (shell_command) @body)
(entrypoint_instruction (json_string_array) @body)
`
	case "html", "htm":
		return html.GetLanguage(), `
(script_element (raw_text) @body)
(style_element (raw_text) @body)
`
	case "css":
		return css.GetLanguage(), `
(rule_set (block) @body)
(media_statement (block) @body)
(keyframes_statement (keyframes_block) @body)
`
	case "svelte":
		return svelte.GetLanguage(), `
(script_element (raw_text) @body)
(style_element (raw_text) @body)
`
	case "yaml", "yml":
		// Multiline scalars only: keep short keys/values intact for AI config context.
		return yaml.GetLanguage(), `
(block_scalar) @body
`
	case "toml":
		return toml.GetLanguage(), `
(string) @body
`
	case "hcl", "tf", "terraform":
		return hcl.GetLanguage(), `
(heredoc_template) @body
`
	case "proto", "protobuf":
		// Keep message field lists (signatures). Stub only rpc bodies when present.
		return protobuf.GetLanguage(), `
(rpc body: (block) @body)
(service body: (block) @body)
`
	case "cue":
		// Stub multiline field values; keep short labels and package clauses.
		return cue.GetLanguage(), `
(field (value (string)) @body)
(field (value (list_lit)) @body)
(field (value (struct_lit)) @body)
`
	case "elm":
		return elm.GetLanguage(), `
(value_declaration (function_declaration_left) (_) @body)
(anonymous_function_expr (_) @body)
`
	case "groovy", "gradle":
		return groovy.GetLanguage(), `
(function_definition body: (block) @body)
(method_declaration body: (block) @body)
(closure (block) @body)
`
	case "md", "markdown":
		// Block grammar only: stub fenced/indented code; keep prose for RAG context.
		return tsmarkdown.GetLanguage(), `
(fenced_code_block (code_fence_content) @body)
(indented_code_block) @body
`
	case "ml", "mli", "ocaml":
		return ocaml.GetLanguage(), `
(let_binding body: (_) @body)
(fun_expression) @body
`
	case "zig", "zon":
		if lang := zigTreeSitterLanguage(); lang != nil {
			return lang, `
(function_declaration body: (block) @body)
(test_declaration body: (block) @body)
(comptime_declaration body: (block) @body)
`
		}
		// Without -tags zigts: fall through to braced heuristic in Skeletonize.
		return nil, ""
	default:
		return nil, ""
	}
}

const jsBodyQuery = `
(function_declaration body: (statement_block) @body)
(generator_function_declaration body: (statement_block) @body)
(method_definition body: (statement_block) @body)
(arrow_function body: (statement_block) @body)
(function_expression body: (statement_block) @body)
(generator_function body: (statement_block) @body)
`

type byteRange struct {
	start, end int
}

func skeletonizeWithTreeSitter(source []byte, lang *sitter.Language, queryStr string) (string, error) {
	ctx := context.Background()
	root, err := sitter.ParseCtx(ctx, source, lang)
	if err != nil || root == nil {
		return "", err
	}

	q, err := sitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return "", err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	qc.Exec(q, root)

	var ranges []byteRange
	seen := map[byteRange]struct{}{}
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		m = qc.FilterPredicates(m, source)
		for _, c := range m.Captures {
			name := q.CaptureNameForId(c.Index)
			if name != "body" {
				continue
			}
			n := c.Node
			r := byteRange{start: int(n.StartByte()), end: int(n.EndByte())}
			if r.end <= r.start {
				continue
			}
			if _, dup := seen[r]; dup {
				continue
			}
			seen[r] = struct{}{}
			ranges = append(ranges, r)
		}
	}
	if len(ranges) == 0 {
		return "", nil
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].start > ranges[j].start
	})

	result := make([]byte, len(source))
	copy(result, source)
	for _, r := range ranges {
		replacement := stubForBody(result[r.start:r.end])
		result = append(result[:r.start], append(replacement, result[r.end:]...)...)
	}
	return string(result), nil
}

func stubForBody(body []byte) []byte {
	trimmed := strings.TrimSpace(string(body))
	switch {
	case strings.HasPrefix(trimmed, "{"):
		return []byte("{ /* body trimmed by trim */ }")
	case strings.HasPrefix(trimmed, ":"):
		// Python suite sometimes captured oddly; keep compact pass.
		return []byte(":\n    ...  # body trimmed by trim\n")
	case strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, ">"):
		// YAML block scalars (literal / folded).
		return []byte("|\n  # body trimmed by trim\n")
	default:
		if strings.Contains(trimmed, "\n") {
			return []byte("\n    ...  # body trimmed by trim\n")
		}
		return []byte("{ /* body trimmed by trim */ }")
	}
}
