package deepopt

import (
	"strings"
	"unicode"
)

// deepTextSpan is one contiguous run of last-user text: either protected
// (never enter LLMLingua) or prose (Segmented Allowlist Deep eligible).
type deepTextSpan struct {
	text      string
	protected bool
}

// DeepProseFromUserText returns the prose islands inside last-user text after
// freezing code fences, paths, URLs, and dense JSON spans in place.
// Empty means nothing for LLMLingua (Fast-only for this turn).
//
// Industry pattern (Microsoft LLMLingua structure + CompressAgent ACC freeze +
// Claude/Cursor selective compaction): never ban a whole user turn because one
// path/URL appeared; split islands, compress prose only, keep control/code verbatim.
func DeepProseFromUserText(blob string) string {
	spans := segmentUserTextForDeep(blob)
	var b strings.Builder
	for _, s := range spans {
		if s.protected {
			continue
		}
		b.WriteString(s.text)
	}
	return strings.TrimSpace(b.String())
}

// reassembleDeepSegments is a test helper for single-island Deep results.
// Live path uses applySegmentedAllowlistDeep (per-island, order-preserving).
// When multiple prose islands exist, callers must pass per-island text via
// joinDeepSpansWithOverrides instead of collapsing into one blob.
func reassembleDeepSegments(spans []deepTextSpan, compressedProse string) string {
	overrides := map[int]string{}
	for i, s := range spans {
		if s.protected || strings.TrimSpace(s.text) == "" {
			continue
		}
		overrides[i] = compressedProse
		break // single compressed blob replaces the first prose island only
	}
	return joinDeepSpansWithOverrides(spans, overrides)
}

// joinDeepSpansWithOverrides rebuilds the user blob with selected prose spans replaced.
// Missing override keys keep the original span text (per-island fail-closed).
func joinDeepSpansWithOverrides(spans []deepTextSpan, overrides map[int]string) string {
	var b strings.Builder
	for i, s := range spans {
		if v, ok := overrides[i]; ok {
			b.WriteString(v)
			continue
		}
		b.WriteString(s.text)
	}
	return b.String()
}

func segmentUserTextForDeep(blob string) []deepTextSpan {
	if blob == "" {
		return nil
	}
	prot := markProtectedRanges(blob)
	var spans []deepTextSpan
	i := 0
	for i < len(blob) {
		start := i
		flag := prot[i]
		for i < len(blob) && prot[i] == flag {
			i++
		}
		spans = append(spans, deepTextSpan{text: blob[start:i], protected: flag})
	}
	return spans
}

func markProtectedRanges(blob string) []bool {
	n := len(blob)
	prot := make([]bool, n)
	markCodeFences(blob, prot)
	markURLs(blob, prot)
	markUNCPaths(blob, prot)
	markWindowsDrivePaths(blob, prot)
	markAbsolutePathPrefixes(blob, prot)
	markRelativeCodePathTokens(blob, prot)
	markDenseJSONObjects(blob, prot)
	return prot
}

func markRange(prot []bool, start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(prot) {
		end = len(prot)
	}
	for i := start; i < end; i++ {
		prot[i] = true
	}
}

func markCodeFences(blob string, prot []bool) {
	for i := 0; i < len(blob); {
		j := strings.Index(blob[i:], "```")
		if j < 0 {
			return
		}
		start := i + j
		rest := blob[start+3:]
		k := strings.Index(rest, "```")
		if k < 0 {
			markRange(prot, start, len(blob))
			return
		}
		end := start + 3 + k + 3
		markRange(prot, start, end)
		i = end
	}
}

func markURLs(blob string, prot []bool) {
	for i := 0; i < len(blob); {
		j := strings.Index(blob[i:], "://")
		if j < 0 {
			return
		}
		mid := i + j
		// Expand left over scheme letters/digits/+.-
		start := mid
		for start > 0 {
			c := blob[start-1]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' {
				start--
				continue
			}
			break
		}
		end := mid + 3
		for end < len(blob) {
			c := blob[end]
			if c <= ' ' || c == '<' || c == '>' || c == '"' || c == '\'' ||
				c == ')' || c == '(' || c == '[' || c == ']' || c == '{' || c == '}' {
				break
			}
			end++
		}
		if end > start {
			markRange(prot, start, end)
		}
		i = end
		if i <= mid {
			i = mid + 3
		}
	}
}

func markUNCPaths(blob string, prot []bool) {
	for i := 0; i+1 < len(blob); {
		if blob[i] == '\\' && blob[i+1] == '\\' {
			end := i + 2
			for end < len(blob) {
				c := blob[end]
				if c <= ' ' || c == '<' || c == '>' || c == '"' || c == '\'' {
					break
				}
				end++
			}
			markRange(prot, i, end)
			i = end
			continue
		}
		i++
	}
}

func markWindowsDrivePaths(blob string, prot []bool) {
	for i := 0; i+2 < len(blob); i++ {
		c := blob[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			continue
		}
		if blob[i+1] != ':' {
			continue
		}
		sep := blob[i+2]
		if sep != '\\' && sep != '/' {
			continue
		}
		// Avoid matching "http:" etc. (handled by URL). Require path-ish after.
		end := i + 3
		for end < len(blob) {
			ch := blob[end]
			if ch <= ' ' || ch == '<' || ch == '>' || ch == '"' || ch == '\'' ||
				ch == ')' || ch == '(' || ch == ',' || ch == ';' {
				break
			}
			end++
		}
		markRange(prot, i, end)
		i = end - 1
	}
}

func markAbsolutePathPrefixes(blob string, prot []bool) {
	prefixes := []string{
		"/Users/", "/home/", "/var/", "/tmp/", "/etc/", "/opt/", "/usr/", "/mnt/",
		"/workspace/", "/Workspace/",
		`\Users\`, `\Windows\`, `\Program Files\`, `\ProgramData\`,
	}
	for _, p := range prefixes {
		startSearch := 0
		for {
			idx := strings.Index(blob[startSearch:], p)
			if idx < 0 {
				break
			}
			idx += startSearch
			end := idx + len(p)
			for end < len(blob) {
				ch := blob[end]
				if ch <= ' ' || ch == '<' || ch == '>' || ch == '"' || ch == '\'' ||
					ch == ')' || ch == '(' || ch == ',' || ch == ';' {
					break
				}
				end++
			}
			markRange(prot, idx, end)
			startSearch = end
		}
	}
}

func markRelativeCodePathTokens(blob string, prot []bool) {
	// Walk whitespace-delimited tokens; protect those that look like code paths.
	i := 0
	for i < len(blob) {
		for i < len(blob) && isPathTokenSep(rune(blob[i])) {
			i++
		}
		if i >= len(blob) {
			return
		}
		start := i
		for i < len(blob) && !isPathTokenSep(rune(blob[i])) {
			i++
		}
		tok := blob[start:i]
		if lastUserHasRelativeCodePath(tok) {
			markRange(prot, start, i)
		}
	}
}

func isPathTokenSep(r rune) bool {
	return unicode.IsSpace(r) || r == ',' || r == ';' || r == '(' || r == ')' ||
		r == '[' || r == ']' || r == '{' || r == '}' || r == '<' || r == '>' ||
		r == '"' || r == '\''
}

func markDenseJSONObjects(blob string, prot []bool) {
	for i := 0; i < len(blob); i++ {
		if blob[i] != '{' {
			continue
		}
		if prot[i] {
			continue
		}
		if !looksLikeJSONObjectStart(blob, i) {
			continue
		}
		end := findBalancedBraceEnd(blob, i)
		if end < 0 {
			continue
		}
		inner := blob[i : end+1]
		if strings.Count(inner, `":`) < 2 && strings.Count(inner, `{"`) < 1 {
			continue
		}
		markRange(prot, i, end+1)
		i = end
	}
}

func looksLikeJSONObjectStart(blob string, i int) bool {
	// {"key" or { "key"
	j := i + 1
	for j < len(blob) && (blob[j] == ' ' || blob[j] == '\n' || blob[j] == '\r' || blob[j] == '\t') {
		j++
	}
	return j < len(blob) && blob[j] == '"'
}

func findBalancedBraceEnd(blob string, start int) int {
	if start >= len(blob) || blob[start] != '{' {
		return -1
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(blob); i++ {
		c := blob[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
