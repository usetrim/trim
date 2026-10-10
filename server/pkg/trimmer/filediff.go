package trimmer

import (
	"fmt"
	"path"
	"strings"
	"sync"
)

// Markers for file-context cache / unified-diff substitution (no invent em dashes).
const (
	MarkerFileUnchanged = "[trim: unchanged file context; previously sent]"
	MarkerFileDiffHead  = "[trim: unified diff vs prior turn]"
)

// NormalizePathKey collapses path separators and cleans . / .. for history keys.
// Empty input returns empty (caller must fail closed / skip).
func NormalizePathKey(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean(p)
	if p == "." || p == "/" {
		return ""
	}
	return p
}

// FileHistory stores the last content sent per file path so repeat turns can
// emit a lightweight stub or a unified diff instead of re-sending the full file.
type FileHistory struct {
	mu       sync.Mutex
	capacity int
	order    []string
	prev     map[string]string
}

// NewFileHistory returns a path-keyed content history. capacity is max distinct paths.
func NewFileHistory(capacity int) *FileHistory {
	if capacity < 1 {
		capacity = 256
	}
	return &FileHistory{
		capacity: capacity,
		prev:     make(map[string]string, capacity),
	}
}

// ApplyFileHistoryToFences runs FileHistory.Apply on each fenced body that has a path.
// Used after LRU skeletonize so exact hash hits still get unchanged stubs / diffs.
func ApplyFileHistoryToFences(text string, h *FileHistory) string {
	if h == nil || text == "" {
		return text
	}
	return codeFenceRe.ReplaceAllStringFunc(text, func(match string) string {
		sub := codeFenceRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		lang, pathHint := parseFenceHeader(sub[1])
		code := sub[2]
		if pathHint == "" {
			pathHint = pathFromCodeBody(code)
		}
		if pathHint == "" || pathHint == lang {
			return match
		}
		next, used := h.Apply(pathHint, code)
		if !used {
			return match
		}
		header := lang + ":" + pathHint
		return "```" + header + "\n" + next + "\n```"
	})
}

// Apply stores content for path and returns either:
//   - unchanged stub when content matches the prior turn
//   - unified diff when content changed vs prior turn
//   - original content on first sighting
//
// used is true when a stub or diff replaced the full body.
func (h *FileHistory) Apply(path, content string) (out string, used bool) {
	if h == nil {
		return content, false
	}
	key := NormalizePathKey(path)
	if key == "" {
		return content, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	prev, seen := h.prev[key]
	if seen && prev == content {
		return MarkerFileUnchanged, true
	}
	if seen && prev != content {
		diff := UnifiedDiff(key, prev, content)
		h.touchLocked(key, content)
		return MarkerFileDiffHead + "\n" + diff, true
	}
	h.touchLocked(key, content)
	return content, false
}

func (h *FileHistory) touchLocked(key, content string) {
	if _, ok := h.prev[key]; ok {
		for i, k := range h.order {
			if k == key {
				h.order = append(h.order[:i], h.order[i+1:]...)
				break
			}
		}
	}
	h.order = append(h.order, key)
	h.prev[key] = content
	for len(h.order) > h.capacity {
		oldest := h.order[0]
		h.order = h.order[1:]
		delete(h.prev, oldest)
	}
}

// UnifiedDiff produces a git-style unified diff between old and new text for path.
func UnifiedDiff(path, oldText, newText string) string {
	a := splitLinesKeep(oldText)
	b := splitLinesKeep(newText)
	ops := lcsDiff(a, b)

	const ctx = 3
	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", path, path)

	// Annotate each op with running old/new 1-based line numbers before the op.
	type numbered struct {
		op      diffOp
		oldLine int // 1-based position in a before applying this op (0 if insert-only)
		newLine int
	}
	numberedOps := make([]numbered, 0, len(ops))
	oi, nj := 1, 1
	for _, o := range ops {
		n := numbered{op: o, oldLine: oi, newLine: nj}
		switch o.kind {
		case opEqual:
			oi++
			nj++
		case opDelete:
			oi++
		case opInsert:
			nj++
		}
		numberedOps = append(numberedOps, n)
	}

	i := 0
	for i < len(numberedOps) {
		// Skip leading equals until we hit an edit.
		for i < len(numberedOps) && numberedOps[i].op.kind == opEqual {
			i++
		}
		if i >= len(numberedOps) {
			break
		}
		// Hunk starts ctx equals before first edit.
		start := i - ctx
		if start < 0 {
			start = 0
		}
		// Find last edit in this contiguous region (edits separated by at most 2*ctx equals).
		end := i + 1
		lastEdit := i
		for end < len(numberedOps) {
			if numberedOps[end].op.kind != opEqual {
				lastEdit = end
				end++
				continue
			}
			// Look ahead: if next edit is within 2*ctx equals, extend hunk.
			look := end
			eqCount := 0
			foundEdit := false
			for look < len(numberedOps) && eqCount <= 2*ctx {
				if numberedOps[look].op.kind != opEqual {
					foundEdit = true
					break
				}
				eqCount++
				look++
			}
			if !foundEdit {
				break
			}
			end = look
			lastEdit = look
			end++
		}
		// Include ctx trailing equals after lastEdit.
		end = lastEdit + 1 + ctx
		if end > len(numberedOps) {
			end = len(numberedOps)
		}

		hunk := numberedOps[start:end]
		oldCount, newCount := 0, 0
		oldStart, newStart := 0, 0
		firstOld, firstNew := true, true
		for _, n := range hunk {
			switch n.op.kind {
			case opEqual:
				oldCount++
				newCount++
				if firstOld {
					oldStart = n.oldLine
					firstOld = false
				}
				if firstNew {
					newStart = n.newLine
					firstNew = false
				}
			case opDelete:
				oldCount++
				if firstOld {
					oldStart = n.oldLine
					firstOld = false
				}
			case opInsert:
				newCount++
				if firstNew {
					newStart = n.newLine
					firstNew = false
				}
			}
		}
		if oldCount == 0 {
			oldStart = hunk[0].oldLine
			if oldStart < 1 {
				oldStart = 0
			} else {
				oldStart--
			}
		}
		if newCount == 0 {
			newStart = hunk[0].newLine
			if newStart < 1 {
				newStart = 0
			} else {
				newStart--
			}
		}
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
		for _, n := range hunk {
			switch n.op.kind {
			case opEqual:
				out.WriteByte(' ')
			case opDelete:
				out.WriteByte('-')
			case opInsert:
				out.WriteByte('+')
			}
			out.WriteString(n.op.line)
			out.WriteByte('\n')
		}
		i = end
	}
	return out.String()
}

type opKind int

const (
	opEqual opKind = iota
	opDelete
	opInsert
)

type diffOp struct {
	kind opKind
	line string
}

func splitLinesKeep(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// lcsDiff is a simple LCS-based line diff (O(nm); fine for typical source files).
func lcsDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, diffOp{kind: opEqual, line: a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{kind: opDelete, line: a[i]})
			i++
		} else {
			ops = append(ops, diffOp{kind: opInsert, line: b[j]})
			j++
		}
	}
	for i < n {
		ops = append(ops, diffOp{kind: opDelete, line: a[i]})
		i++
	}
	for j < m {
		ops = append(ops, diffOp{kind: opInsert, line: b[j]})
		j++
	}
	return ops
}
