package trimmer

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	codeFenceRe  = regexp.MustCompile("(?s)```([^\\n]*)\\n(.*?)```")
	dupLineRe    = regexp.MustCompile(`(?m)^(.*)$`)
	pathInCodeRe = regexp.MustCompile(`(?i)(?:^|\n)\s*(?://|#)\s*(?:filepath|file|path)\s*[:=]\s*([^\s\n]+)`)
	// Selection / diff pinning from user prompts (no em dash ranges).
	lineRangeRe      = regexp.MustCompile(`(?i)\b(?:lines?|L)\s*(\d+)\s*[-:]\s*L?(\d+)\b`)
	pathLineRangeRe  = regexp.MustCompile(`(?i)([a-z0-9_./\\-]+\.[a-z0-9]+)\s*(?:lines?\s*|:)(\d+)\s*[-:]\s*(\d+)`)
	cursorCiteRe     = regexp.MustCompile("(?m)`{0,3}(\\d+):(\\d+):([^\\n`]+)`{0,3}")
	jsMethodSigRe    = regexp.MustCompile(`(?i)^\s*(public|private|protected|static|async|readonly|\*)*\s*[A-Za-z_$][\w$]*\s*\(`)
	activeFileAttrRe = regexp.MustCompile(`(?i)(?:currently[_\s-]?focused[_\s-]?file|active[_\s-]?file|open[_\s-]?file|focused[_\s-]?file)\s*[:=]\s*["']?([^\s"'<>]+)`)
	filePathAttrRe   = regexp.MustCompile(`(?i)<file[^>]*\b(?:path|filepath|name)\s*=\s*["']([^"']+)["']`)
	markdownFileRe   = regexp.MustCompile(`(?m)^(?:#{1,6}\s*)?(?:File|Path|Open file)\s*:\s*([^\s\n]+)`)
	cursorOpenFileRe = regexp.MustCompile(`(?i)(?:open_and_recently_viewed_files|currently focused file)[^\n]{0,200}?([A-Za-z]:[\\/][^\s"'<>]+|/[^\s"'<>]+\.[A-Za-z0-9]+)`)
)

// LineRange is a 1-based inclusive line span inside a source file.
type LineRange struct {
	Start int
	End   int
	Path  string
}

// CompressionMode values match .trimrc mode= (mild|balanced|aggressive|custom).
const (
	ModeMild       = "mild"
	ModeBalanced   = "balanced"
	ModeAggressive = "aggressive"
	ModeCustom     = "custom"
)

// NormalizeMode maps aliases to canonical compression modes.
// Empty or unknown input returns "" (fail closed: no silent invent of balanced).
func NormalizeMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mild", "1", "mode1", "mode_1":
		return ModeMild
	case "balanced", "2", "mode2", "mode_2":
		return ModeBalanced
	case "aggressive", "3", "mode3", "mode_3":
		return ModeAggressive
	case "custom", "4", "mode4", "mode_4":
		return ModeCustom
	default:
		return ""
	}
}

// Heuristic decides whether a code block is safe to skeletonize.
// neverTrimPath is true when .tokenignore / .trimrc marks the path as protected.
// activeFileProtection enables active/focus path immunity from IDE metadata and .trimrc.
// minLines is the small-file exemption (balanced=15, aggressive=5, custom from .trimrc).
func ShouldSkeletonize(filePath, code, userPrompt string, neverTrimPath bool, activeFileProtection bool, minLines int) bool {
	if neverTrimPath {
		return false
	}
	if minLines <= 0 {
		minLines = 15
	}
	lowerPath := strings.ToLower(filePath)
	if activeFileProtection && (strings.Contains(lowerPath, "active") || strings.HasPrefix(lowerPath, "focus:")) {
		return false
	}
	filename := extractFilename(filePath)
	if filename != "" && strings.Contains(userPrompt, filename) {
		return false
	}
	if activeFileProtection {
		for _, protected := range ExtractActiveFilePaths(userPrompt) {
			if pathsReferSame(filePath, protected) {
				return false
			}
		}
	}
	if strings.Count(code, "\n") < minLines {
		return false
	}
	return true
}

func pathsReferSame(a, b string) bool {
	a = strings.TrimSpace(strings.ReplaceAll(a, "\\", "/"))
	b = strings.TrimSpace(strings.ReplaceAll(b, "\\", "/"))
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(a, b) {
		return true
	}
	fa, fb := extractFilename(a), extractFilename(b)
	if fa != "" && strings.EqualFold(fa, fb) {
		return true
	}
	// Suffix match for absolute vs relative path pairs.
	la, lb := strings.ToLower(a), strings.ToLower(b)
	return strings.HasSuffix(la, "/"+lb) || strings.HasSuffix(lb, "/"+la)
}

// ExtractActiveFilePaths pulls IDE / Cursor / VS Code active-file hints from the prompt.
// Those paths stay uncompressed so the model keeps full edit context.
func ExtractActiveFilePaths(userPrompt string) []string {
	if userPrompt == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(strings.Trim(p, `"'`))
		p = strings.ReplaceAll(p, "\\", "/")
		if p == "" {
			return
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, re := range []*regexp.Regexp{activeFileAttrRe, filePathAttrRe, markdownFileRe, cursorOpenFileRe} {
		for _, m := range re.FindAllStringSubmatch(userPrompt, -1) {
			if len(m) > 1 {
				add(m[1])
			}
		}
	}
	return out
}

func extractFilename(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// parseFenceHeader splits fence info like "typescript:src/auth.ts" or `ts path=src/a.ts`.
func parseFenceHeader(info string) (lang, pathHint string) {
	info = strings.TrimSpace(info)
	if info == "" {
		return "", ""
	}
	lower := strings.ToLower(info)
	for _, prefix := range []string{"path=", "file=", "filepath="} {
		if idx := strings.Index(lower, prefix); idx >= 0 {
			rest := strings.TrimSpace(info[idx+len(prefix):])
			rest = strings.Trim(rest, `"'`)
			langPart := strings.TrimSpace(info[:idx])
			return firstToken(langPart), rest
		}
	}
	if i := strings.Index(info, ":"); i > 0 {
		maybeLang := info[:i]
		maybePath := strings.TrimSpace(info[i+1:])
		if strings.Contains(maybePath, "/") || strings.Contains(maybePath, ".") {
			return firstToken(maybeLang), strings.Trim(maybePath, `"'`)
		}
	}
	return firstToken(info), ""
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func pathFromCodeBody(code string) string {
	if m := pathInCodeRe.FindStringSubmatch(code); len(m) > 1 {
		return strings.Trim(m[1], `"'`)
	}
	return ""
}

// SkeletonizeSource replaces function/method bodies with stubs for supported languages.
// Default (CGO-free): Go uses the stdlib AST parser; other languages use structural heuristics
// that preserve signatures, types, interfaces, and imports while stubbing bodies.
// Optional: build with -tags treesitter and CGO_ENABLED=1 to prefer Tree-sitter grammars
// via tryTreeSitterSkeletonize when those bindings are wired.
func SkeletonizeSource(lang, source string) string {
	return SkeletonizeSourceOpts(lang, source, nil)
}

// SkeletonizeSourceOpts is SkeletonizeSource with optional .trimrc custom_query snippets.
// When customQueries is non-empty, only Tree-sitter applies them. If Tree-sitter is
// unavailable or the query fails, the source is returned unchanged (fail closed, no invent).
func SkeletonizeSourceOpts(lang, source string, customQueries []string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if out, ok := tryTreeSitterSkeletonize(lang, source, customQueries); ok {
		return out
	}
	if len(customQueries) > 0 {
		return source
	}
	switch lang {
	case "go", "golang":
		return SkeletonizeGoAST(source)
	case "ts", "tsx", "typescript", "js", "jsx", "javascript", "mjs", "cjs":
		return skeletonizeJavaScript(source)
	case "py", "python":
		return skeletonizePython(source)
	case "rs", "rust":
		return skeletonizeBraced(source, []string{"fn ", "impl ", "pub fn ", "async fn ", "pub async fn "})
	case "java", "kt", "kotlin", "scala", "groovy", "gradle":
		return skeletonizeBraced(source, []string{"class ", "interface ", "fun ", "def ", "public ", "private ", "protected ", "override ", "object ", "trait "})
	case "c", "h", "cpp", "cc", "cxx", "hpp", "c++":
		return skeletonizeBraced(source, []string{"class ", "struct ", "namespace ", "template "})
	case "cs", "csharp":
		return skeletonizeBraced(source, []string{"class ", "interface ", "struct ", "record ", "public ", "private ", "protected ", "internal "})
	case "php":
		return skeletonizeBraced(source, []string{"function ", "class ", "public function ", "private function ", "protected function "})
	case "rb", "ruby":
		return skeletonizeRuby(source)
	case "swift":
		return skeletonizeBraced(source, []string{"func ", "class ", "struct ", "protocol ", "extension "})
	case "zig":
		return skeletonizeZig(source)
	case "sql":
		return skeletonizeSQL(source)
	case "bash", "sh", "shell", "zsh", "ksh":
		return skeletonizeBash(source)
	case "lua":
		return skeletonizeBraced(source, []string{"function ", "local function "})
	case "elixir", "ex", "exs":
		return skeletonizeElixir(source)
	case "elm":
		return skeletonizeBraced(source, []string{"=", "let ", "in ", "case ", "of "})
	case "ocaml", "ml", "mli":
		return skeletonizeBraced(source, []string{"let ", "fun ", "module ", "struct ", "sig "})
	case "css":
		return skeletonizeBraced(source, []string{"{", "@media ", "@keyframes ", "@supports "})
	case "html", "htm", "svelte":
		return stripCommentsAndWhitespace(source)
	case "dockerfile", "docker":
		return stripCommentsAndWhitespace(source)
	case "yaml", "yml", "toml", "hcl", "tf", "terraform", "proto", "protobuf", "cue", "md", "markdown":
		return stripCommentsAndWhitespace(source)
	default:
		return stripCommentsAndWhitespace(source)
	}
}

// LangFromFilename maps a file path extension to a trimmer language id.
// Empty means unknown (caller should use log/heuristic-only paths).
func LangFromFilename(path string) string {
	base := strings.ToLower(filepath.Base(path))
	if base == "dockerfile" || strings.HasPrefix(base, "dockerfile.") {
		return "dockerfile"
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	switch ext {
	case "go":
		return "go"
	case "ts":
		return "typescript"
	case "tsx":
		return "tsx"
	case "js", "jsx", "mjs", "cjs":
		return "javascript"
	case "py", "pyw":
		return "python"
	case "rs":
		return "rust"
	case "java":
		return "java"
	case "kt", "kts":
		return "kotlin"
	case "c", "h":
		return "c"
	case "cpp", "cc", "cxx", "hpp", "hxx":
		return "cpp"
	case "cs":
		return "csharp"
	case "php":
		return "php"
	case "rb":
		return "ruby"
	case "swift":
		return "swift"
	case "zig", "zon":
		return "zig"
	case "sql":
		return "sql"
	case "scala":
		return "scala"
	case "sh", "bash", "zsh", "ksh":
		return "bash"
	case "lua":
		return "lua"
	case "ex", "exs":
		return "elixir"
	case "dockerfile":
		return "dockerfile"
	case "html", "htm":
		return "html"
	case "css":
		return "css"
	case "svelte":
		return "svelte"
	case "yml", "yaml":
		return "yaml"
	case "toml":
		return "toml"
	case "hcl", "tf", "tfvars":
		return "hcl"
	case "proto":
		return "protobuf"
	case "cue":
		return "cue"
	case "elm":
		return "elm"
	case "groovy", "gradle":
		return "groovy"
	case "md", "markdown":
		return "markdown"
	case "ml", "mli":
		return "ocaml"
	default:
		return ""
	}
}

// UnwrapSingleCodeFence returns the fence body when s is exactly one markdown fence.
func UnwrapSingleCodeFence(s string) string {
	trimmed := strings.TrimSpace(s)
	sub := codeFenceRe.FindStringSubmatch(trimmed)
	if len(sub) < 3 {
		return s
	}
	if strings.TrimSpace(codeFenceRe.ReplaceAllString(trimmed, "")) != "" {
		return s
	}
	return sub[2]
}

// skeletonizeJavaScript preserves imports, type/interface/declare lines, and
// stubs function / method / arrow / class constructor bodies with brace depth.
func skeletonizeJavaScript(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	depth := 0
	inBody := false
	pendingSig := false

	isTypeOnly := func(trimmed string) bool {
		return strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "export type ") ||
			strings.HasPrefix(trimmed, "export interface ") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "declare ") ||
			strings.HasPrefix(trimmed, "export declare ")
	}

	isCallableSig := func(line string) bool {
		trimmed := strings.TrimSpace(line)
		if isTypeOnly(trimmed) {
			return false
		}
		markers := []string{
			"function ", "async function ", "export function ", "export async function ",
			"export default function ", "constructor(", " get ", " set ",
			") => {", ")=>{", "=> {", "=>{",
		}
		for _, m := range markers {
			if strings.Contains(line, m) {
				return true
			}
		}
		// Class / object method style: name(...) {  or async name(...) {
		if strings.Contains(line, "(") && strings.Contains(line, ")") && strings.Contains(line, "{") {
			if strings.Contains(line, "class ") || strings.Contains(line, "enum ") {
				return false
			}
			if jsMethodSigRe.MatchString(line) {
				return true
			}
		}
		return false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBody {
			if isTypeOnly(trimmed) {
				out = append(out, line)
				continue
			}

			opens := strings.Count(line, "{")
			closes := strings.Count(line, "}")

			if isCallableSig(line) && opens > 0 {
				sig := strings.TrimRight(line, "{ \t")
				out = append(out, sig+" { /* body trimmed by trim */ }")
				depth += opens - closes
				if depth > 0 {
					inBody = true
				}
				pendingSig = false
				continue
			}

			// Multi-line signature: keep lines until opening brace.
			if isCallableSig(line) || pendingSig {
				pendingSig = true
				if opens > 0 {
					sig := strings.TrimRight(line, "{ \t")
					out = append(out, sig+" { /* body trimmed by trim */ }")
					depth += opens - closes
					if depth > 0 {
						inBody = true
					}
					pendingSig = false
					continue
				}
				out = append(out, line)
				continue
			}

			out = append(out, line)
			continue
		}

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			inBody = false
			depth = 0
			pendingSig = false
		}
	}
	return strings.Join(out, "\n")
}

// skeletonizeZig stubs fn / test / comptime blocks. Used when Tree-sitter Zig
// is off (default) or zigts build failed. Prefer -tags treesitter,zigts with
// github.com/tree-sitter-grammars/tree-sitter-zig for grammar-accurate stubs.
func skeletonizeZig(source string) string {
	markers := []string{
		"pub fn ", "export fn ", "inline fn ", "pub inline fn ", "export inline fn ",
		"pub extern fn ", "extern fn ",
		"fn ", "test ", "comptime ", "pub comptime ",
		"pub const ", "const ", "struct ", "union ", "enum ", "opaque ",
		"pub threadlocal ", "threadlocal ",
	}
	return skeletonizeBraced(source, markers)
}

func skeletonizeSQL(source string) string {
	// Heuristic path when Tree-sitter SQL is off. Keep CREATE / ALTER signatures,
	// stub CREATE FUNCTION / PROCEDURE bodies, and truncate bulky INSERT values.
	upper := strings.ToUpper(source)
	if strings.Contains(upper, "INSERT INTO") && len(source) > 800 {
		idx := strings.Index(upper, "VALUES")
		if idx > 0 {
			return source[:idx] + "VALUES (/* rows trimmed by trim */);"
		}
	}
	for _, marker := range []string{"CREATE OR REPLACE FUNCTION", "CREATE FUNCTION", "CREATE OR REPLACE PROCEDURE", "CREATE PROCEDURE"} {
		mi := strings.Index(upper, marker)
		if mi < 0 {
			continue
		}
		asIdx := strings.Index(upper[mi:], " AS ")
		beginIdx := strings.Index(upper[mi:], " BEGIN")
		cutRel := -1
		if asIdx >= 0 {
			cutRel = asIdx
		}
		if beginIdx >= 0 && (cutRel < 0 || beginIdx < cutRel) {
			cutRel = beginIdx
		}
		if cutRel >= 0 {
			cut := mi + cutRel
			return source[:cut] + " AS /* body trimmed by trim */;"
		}
	}
	return stripCommentsAndWhitespace(source)
}

func skeletonizeRuby(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	inBody := false
	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBody {
			if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "module ") {
				out = append(out, line)
				out = append(out, "  # body trimmed by trim")
				inBody = true
				depth = 1
				continue
			}
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "module ") || strings.HasPrefix(trimmed, "do ") || strings.HasSuffix(trimmed, " do") {
			depth++
		}
		if trimmed == "end" {
			depth--
			if depth <= 0 {
				out = append(out, line)
				inBody = false
				depth = 0
			}
			continue
		}
	}
	return strings.Join(out, "\n")
}

func skeletonizeBash(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	inFunc := false
	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFunc {
			isFunc := strings.HasPrefix(trimmed, "function ") ||
				(strings.Contains(trimmed, "()") && (strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, "()")))
			if isFunc {
				sig := strings.TrimRight(line, "{ \t")
				out = append(out, sig+" { /* body trimmed by trim */ }")
				opens := strings.Count(line, "{")
				closes := strings.Count(line, "}")
				depth = opens - closes
				if depth > 0 {
					inFunc = true
				}
				continue
			}
			out = append(out, line)
			continue
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			inFunc = false
			depth = 0
		}
	}
	return strings.Join(out, "\n")
}

func skeletonizeElixir(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	inBody := false
	depth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBody {
			if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "defp ") ||
				strings.HasPrefix(trimmed, "defmacro ") || strings.HasPrefix(trimmed, "defmacrop ") ||
				strings.HasPrefix(trimmed, "defmodule ") {
				out = append(out, line)
				out = append(out, "  # body trimmed by trim")
				inBody = true
				depth = 1
				continue
			}
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "defp ") ||
			strings.HasPrefix(trimmed, "defmacro ") || strings.HasPrefix(trimmed, "defmacrop ") ||
			strings.HasPrefix(trimmed, "defmodule ") || strings.HasPrefix(trimmed, "do ") {
			depth++
		}
		if trimmed == "end" {
			depth--
			if depth <= 0 {
				out = append(out, line)
				inBody = false
				depth = 0
			}
			continue
		}
	}
	return strings.Join(out, "\n")
}

func skeletonizeBraced(source string, markers []string) string {
	lines := strings.Split(source, "\n")
	var out []string
	depth := 0
	inBody := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBody {
			hasMarker := false
			for _, m := range markers {
				if strings.Contains(line, m) {
					hasMarker = true
					break
				}
			}
			opens := strings.Count(line, "{")
			closes := strings.Count(line, "}")
			if hasMarker && opens > 0 {
				// Keep signature line, stub body
				sig := strings.TrimRight(line, "{ \t")
				out = append(out, sig+" { /* body trimmed by trim */ }")
				depth += opens - closes
				if depth > 0 {
					inBody = true
				}
				continue
			}
			out = append(out, line)
			continue
		}

		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth <= 0 {
			inBody = false
			depth = 0
		}
		_ = trimmed
	}
	return strings.Join(out, "\n")
}

func skeletonizePython(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	skipIndent := -1

	for _, line := range lines {
		indent := countLeadingSpaces(line)
		trimmed := strings.TrimSpace(line)

		if skipIndent >= 0 {
			if trimmed == "" {
				continue
			}
			if indent > skipIndent {
				continue
			}
			skipIndent = -1
		}

		// Keep class headers and module-level imports / assignments.
		if strings.HasPrefix(trimmed, "class ") {
			out = append(out, line)
			continue
		}

		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
			out = append(out, line)
			if strings.HasSuffix(trimmed, ":") {
				out = append(out, strings.Repeat(" ", indent+4)+"# body trimmed by trim")
				out = append(out, strings.Repeat(" ", indent+4)+"...")
				skipIndent = indent
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func countLeadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}

func stripCommentsAndWhitespace(source string) string {
	lines := strings.Split(source, "\n")
	var out []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// Product stub markers written into optimized payloads (intentional, not UI invent).
const (
	MarkerLogRepeatedPrefix       = "[repeated "
	MarkerLogCompactedPref        = "[trim] log compacted to last "
	MarkerLogTruncated            = "[trim] truncated to max_log_bytes"
	MarkerBinaryRemovedPref       = "[binary blob removed: "
	MarkerHistoryCompacted        = "[trim] earlier conversation turns compacted; recent turns retained"
	MarkerLogRuntimeCollapsedPref = "[trim] collapsed runtime/dependency stack "
)

// CompactLogs deduplicates repeated lines, strips binary/base64/hex dumps,
// collapses noisy stacks (keeps top user frames, stubs node_modules/runtime frames),
// and optionally caps dump size / line count from .trimrc (no invent defaults).
// maxBytes <= 0: no hard byte cap. minBytes <= 0: always compact. maxLines <= 0: no line cap.
// noiseSubstrings empty: no invent noise filter (configure via .trimrc log_noise_substrings).
func CompactLogs(raw string, maxBytes, minBytes, maxLines int, noiseSubstrings []string) string {
	if minBytes > 0 && len(raw) < minBytes {
		return raw
	}
	lines := strings.Split(raw, "\n")
	var out []string
	var prev string
	repeat := 0
	collapsedRuntime := 0

	flushRuntime := func() {
		if collapsedRuntime > 0 {
			out = append(out, MarkerLogRuntimeCollapsedPref+itoa(collapsedRuntime)+" frames]")
			collapsedRuntime = 0
		}
	}

	for _, line := range lines {
		clean := stripANSI(line)
		if isNoiseLog(clean, noiseSubstrings) {
			continue
		}
		clean = redactBinaryBlobs(clean)
		if isRuntimeStackFrame(clean) {
			if repeat > 0 {
				out = append(out, MarkerLogRepeatedPrefix+itoa(repeat)+"x] "+prev)
				repeat = 0
				prev = ""
			}
			collapsedRuntime++
			continue
		}
		flushRuntime()
		if clean == prev {
			repeat++
			continue
		}
		if repeat > 0 {
			out = append(out, MarkerLogRepeatedPrefix+itoa(repeat)+"x] "+prev)
			repeat = 0
		}
		prev = clean
		out = append(out, clean)
	}
	flushRuntime()
	if repeat > 0 {
		out = append(out, MarkerLogRepeatedPrefix+itoa(repeat)+"x] "+prev)
	}

	if maxLines > 0 && len(out) > maxLines {
		out = out[len(out)-maxLines:]
		out = append([]string{MarkerLogCompactedPref + itoa(maxLines) + " meaningful lines"}, out...)
	}
	joined := strings.Join(out, "\n")
	if maxBytes > 0 && len(joined) > maxBytes {
		joined = joined[:maxBytes] + "\n" + MarkerLogTruncated
	}
	return joined
}

func isNoiseLog(line string, noiseSubstrings []string) bool {
	if len(noiseSubstrings) == 0 {
		return false
	}
	l := strings.ToLower(line)
	for _, n := range noiseSubstrings {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		if strings.Contains(l, n) {
			return true
		}
	}
	return false
}

// isRuntimeStackFrame detects dependency / runtime frames safe to collapse in stacks.
// Project source frames (src/, app/, lib/, internal/) are kept.
func isRuntimeStackFrame(line string) bool {
	l := strings.ToLower(strings.TrimSpace(line))
	if l == "" {
		return false
	}
	// Keep project-looking paths even if they mention node_modules in a message.
	keepHints := []string{"/src/", "\\src\\", "/app/", "\\app\\", "/lib/", "\\lib\\", "/internal/", "\\internal\\"}
	for _, h := range keepHints {
		if strings.Contains(l, h) && !strings.Contains(l, "node_modules") {
			return false
		}
	}
	runtimeHints := []string{
		"node_modules/",
		"node_modules\\",
		"at layer.handle",
		"at router.",
		"at process.processticksandrejections",
		"at async promise.all",
		"golang.org/",
		"runtime.goexit",
		"runtime.main",
		"<anonymous>",
	}
	for _, h := range runtimeHints {
		if strings.Contains(l, h) {
			return true
		}
	}
	// Generic "at ... (node:internal/...)" Node frames
	if strings.HasPrefix(l, "at ") && (strings.Contains(l, "node:internal") || strings.Contains(l, "(internal/")) {
		return true
	}
	return false
}

var (
	ansiRe    = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	base64Re  = regexp.MustCompile(`(?i)(?:data:[a-z0-9.+/-]+;base64,|[A-Za-z0-9+/]{80,}={0,2})`)
	hexDumpRe = regexp.MustCompile(`(?i)(?:0x)?[0-9a-f]{64,}`)
)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// redactBinaryBlobs replaces long base64 / hex dumps with a size stub.
func redactBinaryBlobs(line string) string {
	if base64Re.MatchString(line) {
		line = base64Re.ReplaceAllStringFunc(line, func(m string) string {
			return MarkerBinaryRemovedPref + itoa(len(m)) + "B]"
		})
	}
	if hexDumpRe.MatchString(line) {
		line = hexDumpRe.ReplaceAllStringFunc(line, func(m string) string {
			return MarkerBinaryRemovedPref + itoa(len(m)) + "B]"
		})
	}
	return line
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// ProcessOptions controls prompt trimming. Paths come from fence headers / body hints.
type ProcessOptions struct {
	NeverTrim   func(path string) bool
	MaxLogBytes int
	// LogCompactMinBytes: skip CompactLogs when raw length is below this. 0 = always compact.
	LogCompactMinBytes int
	// LogCompactMaxLines: keep last N meaningful lines after compaction. 0 = no line cap.
	LogCompactMaxLines   int
	ActiveFileProtection bool
	// Mode: mild (logs only), balanced (default AST), aggressive (deeper AST), custom.
	Mode string
	// MinLines overrides the small-file exemption when > 0. Mode defaults apply otherwise.
	MinLines int
	// BalancedMinLines / AggressiveMinLines / MildMinLines override mode built-ins when MinLines is 0.
	// Zero means use built-in (15 / 5 / 15). Never invent from empty config elsewhere.
	BalancedMinLines   int
	AggressiveMinLines int
	MildMinLines       int
	// DisableSkeletonize skips AST body stripping (mild / custom_logs_only).
	DisableSkeletonize bool
	// AlwaysCompactLogs forces log compaction even without error/TRACE markers.
	AlwaysCompactLogs bool
	// LogNoiseSubstrings drops matching log lines during CompactLogs (from .trimrc).
	// Empty = no invent noise filter.
	LogNoiseSubstrings []string
	// FileHistory enables path-keyed unchanged stubs and unified-diff substitution
	// across chat turns (bible Local Cache Check and Diff Conversion).
	FileHistory *FileHistory
	// CustomQueries are Tree-sitter query snippets from .trimrc custom_query (Mode 4).
	// Applied when non-empty; require a treesitter build (fail closed otherwise).
	CustomQueries []string
}

// ParseSelectionRanges extracts pinned line ranges from the user prompt / selection metadata.
func ParseSelectionRanges(prompt string) []LineRange {
	if prompt == "" {
		return nil
	}
	var out []LineRange
	for _, m := range pathLineRangeRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) < 4 {
			continue
		}
		start, end := atoiSafe(m[2]), atoiSafe(m[3])
		if start > 0 && end >= start {
			out = append(out, LineRange{Start: start, End: end, Path: strings.TrimSpace(m[1])})
		}
	}
	for _, m := range cursorCiteRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) < 4 {
			continue
		}
		start, end := atoiSafe(m[1]), atoiSafe(m[2])
		path := strings.TrimSpace(m[3])
		if start > 0 && end >= start && path != "" {
			out = append(out, LineRange{Start: start, End: end, Path: path})
		}
	}
	for _, m := range lineRangeRe.FindAllStringSubmatch(prompt, -1) {
		if len(m) < 3 {
			continue
		}
		start, end := atoiSafe(m[1]), atoiSafe(m[2])
		if start > 0 && end >= start {
			out = append(out, LineRange{Start: start, End: end})
		}
	}
	return out
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func rangesForPath(all []LineRange, pathHint string) []LineRange {
	if len(all) == 0 {
		return nil
	}
	fn := extractFilename(pathHint)
	var matched []LineRange
	var generic []LineRange
	for _, r := range all {
		if r.Path == "" {
			generic = append(generic, r)
			continue
		}
		rfn := extractFilename(r.Path)
		if strings.EqualFold(r.Path, pathHint) || (fn != "" && strings.EqualFold(rfn, fn)) {
			matched = append(matched, r)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return generic
}

// SkeletonizeWithPinnedLines keeps highlighted / selected line ranges verbatim
// while skeletonizing surrounding regions.
func SkeletonizeWithPinnedLines(lang, source string, pinned []LineRange) string {
	return SkeletonizeWithPinnedLinesOpts(lang, source, pinned, nil)
}

// SkeletonizeWithPinnedLinesOpts is SkeletonizeWithPinnedLines with custom Tree-sitter queries.
func SkeletonizeWithPinnedLinesOpts(lang, source string, pinned []LineRange, customQueries []string) string {
	if len(pinned) == 0 {
		return SkeletonizeSourceOpts(lang, source, customQueries)
	}
	lines := strings.Split(source, "\n")
	n := len(lines)
	pin := make([]bool, n)
	for _, r := range pinned {
		start, end := r.Start, r.End
		if start < 1 {
			start = 1
		}
		if end > n {
			end = n
		}
		for i := start - 1; i < end && i < n; i++ {
			if i >= 0 {
				pin[i] = true
			}
		}
	}
	var out strings.Builder
	i := 0
	first := true
	writeBlock := func(s string) {
		if s == "" {
			return
		}
		if !first {
			out.WriteByte('\n')
		}
		out.WriteString(s)
		first = false
	}
	for i < n {
		if pin[i] {
			var raw []string
			for i < n && pin[i] {
				raw = append(raw, lines[i])
				i++
			}
			writeBlock(strings.Join(raw, "\n"))
			continue
		}
		start := i
		for i < n && !pin[i] {
			i++
		}
		block := strings.Join(lines[start:i], "\n")
		writeBlock(SkeletonizeSourceOpts(lang, block, customQueries))
	}
	return out.String()
}

// ProcessPromptText finds markdown code fences and skeletonizes eligible blocks.
// neverTrim matches against real file paths extracted from fence headers or body comments.
// Mode is empty (fail closed: pass-through). Prefer ProcessPromptTextOpts with an explicit mode.
// ActiveFileProtection defaults false here (no invent); callers must set via ProcessPromptTextOpts.
func ProcessPromptText(text, userPrompt string, neverTrim func(path string) bool) (string, int, int) {
	return ProcessPromptTextOpts(text, userPrompt, ProcessOptions{
		NeverTrim:            neverTrim,
		ActiveFileProtection: false,
		Mode:                 "",
	})
}

func ProcessPromptTextOpts(text, userPrompt string, opts ProcessOptions) (string, int, int) {
	mode := NormalizeMode(opts.Mode)
	before := estimateTokens(text)
	// Fail closed: no mode configured means pass-through (no invent of balanced).
	if mode == "" {
		return text, before, before
	}
	minLines := opts.MinLines
	disableSkeletonize := opts.DisableSkeletonize
	alwaysCompact := opts.AlwaysCompactLogs
	switch mode {
	case ModeMild:
		disableSkeletonize = true
		alwaysCompact = true
		if minLines <= 0 {
			if opts.MildMinLines > 0 {
				minLines = opts.MildMinLines
			} else {
				minLines = 15
			}
		}
	case ModeAggressive:
		if minLines <= 0 {
			if opts.AggressiveMinLines > 0 {
				minLines = opts.AggressiveMinLines
			} else {
				minLines = 5
			}
		}
		alwaysCompact = true
	case ModeCustom:
		if minLines <= 0 {
			minLines = 15
		}
	default:
		if minLines <= 0 {
			if opts.BalancedMinLines > 0 {
				minLines = opts.BalancedMinLines
			} else {
				minLines = 15
			}
		}
	}

	selections := ParseSelectionRanges(userPrompt)
	result := codeFenceRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := codeFenceRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		lang, pathHint := parseFenceHeader(sub[1])
		code := sub[2]
		if pathHint == "" {
			pathHint = pathFromCodeBody(code)
		}
		if pathHint == "" {
			pathHint = lang
		}
		body := code
		changed := false
		if !disableSkeletonize {
			protected := opts.NeverTrim != nil && (opts.NeverTrim(pathHint) ||
				(extractFilename(pathHint) != "" && opts.NeverTrim(extractFilename(pathHint))))
			if ShouldSkeletonize(pathHint, code, userPrompt, protected, opts.ActiveFileProtection, minLines) {
				pinned := rangesForPath(selections, pathHint)
				body = SkeletonizeWithPinnedLinesOpts(lang, code, pinned, opts.CustomQueries)
				changed = true
			}
		}
		if opts.FileHistory != nil && pathHint != "" && pathHint != lang {
			if next, used := opts.FileHistory.Apply(pathHint, body); used {
				body = next
				changed = true
			}
		}
		if !changed {
			return match
		}
		header := lang
		if pathHint != "" && pathHint != lang {
			header = lang + ":" + pathHint
		}
		return "```" + header + "\n" + body + "\n```"
	})

	if alwaysCompact ||
		strings.Contains(strings.ToLower(result), "error") ||
		strings.Contains(result, "TRACE") {
		result = CompactLogs(result, opts.MaxLogBytes, opts.LogCompactMinBytes, opts.LogCompactMaxLines, opts.LogNoiseSubstrings)
	}

	after := estimateTokens(result)
	return result, before, after
}

func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	// Rough English/code heuristic: ~4 chars per token
	return (len(s) + 3) / 4
}
