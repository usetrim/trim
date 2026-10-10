package deepopt

import (
	"fmt"
	"os"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// compressFn is the Deep engine entrypoint (swapped in unit tests).
var compressFn = Compress

// ChatDeepChrome is fail-closed copy for live/proxy Deep rewriting.
// Empty CompactStub means refuse to rewrite (no invent English stub).
type ChatDeepChrome struct {
	CompactStub string
	Unavailable string
}

func (c ChatDeepChrome) err(msg string) error {
	if msg != "" {
		return fmt.Errorf("%s", msg)
	}
	if c.Unavailable != "" {
		return fmt.Errorf("%s", c.Unavailable)
	}
	return fmt.Errorf("deep chat chrome unavailable")
}

// CompressOpenAIChatJSON runs Deep on the last user turn only for every OpenAI-shape
// door (GPT, Gemini OpenAI-compat, DeepSeek, Mistral, …). Preserves system/developer/
// assistant/tool messages and top-level tools/tool_choice. Same LLMLingua rules as Anthropic.
func CompressOpenAIChatJSON(body []byte, engine Engine, targetToken int, question string, ux UXChrome, timeoutSec int, chrome ChatDeepChrome, rt Runtime) ([]byte, int, int, error) {
	return compressChatMessagesJSON(body, engine, targetToken, question, ux, timeoutSec, chrome, rt)
}

// CompressAnthropicMessagesJSON Deep-compresses Anthropic Messages API bodies.
//
// Hardening for Claude Code / agent payloads (best practice):
//   - Never dump system + tools + scaffolding into LLMLingua.
//   - Only compress last user text (keep <system-reminder> / cache_control / signed parts).
//   - User turns may be [tool_result, ..., text]: tool chrome stays; text may Deep-rewrite.
//   - Preserve all other messages and top-level fields as-is.
//   - Fail-closed on expansion (return original body).
func CompressAnthropicMessagesJSON(body []byte, engine Engine, targetToken int, question string, ux UXChrome, timeoutSec int, chrome ChatDeepChrome, rt Runtime) ([]byte, int, int, error) {
	return compressChatMessagesJSON(body, engine, targetToken, question, ux, timeoutSec, chrome, rt)
}

// compressChatMessagesJSON is the shared last-user-only Deep path for Anthropic Messages
// and OpenAI Chat Completions (GPT / Gemini / DeepSeek / Mistral openai_compat).
func compressChatMessagesJSON(body []byte, engine Engine, targetToken int, question string, ux UXChrome, timeoutSec int, chrome ChatDeepChrome, rt Runtime) ([]byte, int, int, error) {
	if !gjson.ValidBytes(body) {
		return nil, 0, 0, chrome.err(chrome.Unavailable)
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		n := estimateTokens(body)
		return body, n, n, nil
	}

	if strings.TrimSpace(chrome.CompactStub) == "" {
		return nil, 0, 0, chrome.err(chrome.Unavailable)
	}

	arr := messages.Array()
	// Only Deep-rewrite a fresh trailing user turn. Mid-tool-loop / assistant-ending
	// bodies stay Fast-only. Prior tool history is OK: we never feed tools into LLMLingua.
	if ShouldSkipDeepForChatTurn(arr) {
		n := estimateTokens(body)
		return body, n, n, nil
	}

	lastUser := len(arr) - 1
	// lastUserDeepPlainText already covers OpenAI string content + array text parts.
	// Never fall back to messageCompressibleText: that ignored thought_signature /
	// cache_control freezes and could feed agent chrome into LLMLingua (garble class).
	blob := strings.TrimSpace(lastUserDeepPlainText(arr[lastUser]))
	if blob == "" {
		n := estimateTokens(body)
		return body, n, n, nil
	}
	// Coding-agent fail-closed: LLMLingua is lossy and can mangle paths/URLs/fenced code
	// even when system/tools are frozen (same symptom class as “garbled” for GPT/Claude/Gemini).
	if LastUserUnsafeForDeepLLMLingua(blob) {
		n := estimateTokens(body)
		return body, n, n, nil
	}

	// Microsoft LLMLingua: keep the question high-sensitivity; compress filler/context only.
	ctxText, qText := splitDeepContextQuestion(blob)
	q := strings.TrimSpace(question)
	if q == "" {
		q = qText
	}
	if q == "" {
		q = ctxText
	}
	req := Request{
		Text:        ctxText,
		Question:    q,
		TargetToken: targetToken,
		Engine:      string(engine),
	}
	if err := rt.ApplyEngine(&req, engine); err != nil {
		return nil, 0, 0, err
	}
	res, err := compressFn(req, ux, timeoutSec)
	if err != nil {
		return nil, 0, 0, err
	}

	origin := res.OriginTokens
	compressed := res.CompressedTokens
	if origin <= 0 {
		origin = estimateTokens([]byte(blob))
	}
	if compressed <= 0 {
		compressed = estimateTokens([]byte(res.CompressedPrompt))
	}

	compressedPrompt := res.CompressedPrompt
	appendedQuestion := false
	if ctxText != blob && strings.TrimSpace(qText) != "" &&
		!strings.Contains(compressedPrompt, qText) {
		compressedPrompt = strings.TrimSpace(compressedPrompt) + "\n" + qText
		appendedQuestion = true
	}

	// Fail-closed with comparable units. If we appended the preserved question,
	// engine compressed_tokens no longer cover the full last-user text - re-estimate
	// both sides with the same estimator (byte/4). Mixing engine tokens with
	// estimateTokens(appendedPrompt) falsely rejected every Deep result (live 0% Saved).
	if appendedQuestion {
		origin = estimateTokens([]byte(blob))
		compressed = estimateTokens([]byte(compressedPrompt))
	}
	if compressed >= origin {
		if os.Getenv("TRIM_DEEP_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "trim-deep-debug: reject compressed>=origin origin=%d compressed=%d appended=%v blob_len=%d prompt_len=%d\n",
				origin, compressed, appendedQuestion, len(blob), len(compressedPrompt))
		}
		n := estimateTokens(body)
		return body, n, n, nil
	}
	// Byte-length guard when engine counts were missing, or after question append.
	if (res.OriginTokens <= 0 && res.CompressedTokens <= 0) || appendedQuestion {
		if len(compressedPrompt) >= len(blob) {
			if os.Getenv("TRIM_DEEP_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "trim-deep-debug: reject byte-guard blob_len=%d prompt_len=%d\n", len(blob), len(compressedPrompt))
			}
			n := estimateTokens(body)
			return body, n, n, nil
		}
	}

	newContent := rewriteLastUserContent(arr[lastUser], compressedPrompt)
	path := fmt.Sprintf("messages.%d.content", lastUser)
	out, err := sjson.SetBytes(body, path, newContent)
	if err != nil {
		return nil, 0, 0, err
	}
	if os.Getenv("TRIM_DEEP_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "trim-deep-debug: appended=%v origin=%d compressed=%d blob_len=%d prompt_len=%d body_in=%d body_out=%d\n",
			appendedQuestion, origin, compressed, len(blob), len(compressedPrompt), len(body), len(out))
	}
	return out, origin, compressed, nil
}

// splitDeepContextQuestion separates noisy user filler (context) from the trailing
// question (high compression sensitivity per Microsoft LLMLingua DOCUMENT.md).
func splitDeepContextQuestion(blob string) (contextText, question string) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return "", ""
	}
	lines := strings.Split(blob, "\n")
	last := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = i
			break
		}
	}
	if last < 0 {
		return blob, blob
	}
	question = strings.TrimSpace(lines[last])
	if last == 0 {
		// Single-line user turn: treat as question-only (Deep may still run if large).
		return blob, question
	}
	contextText = strings.TrimSpace(strings.Join(lines[:last], "\n"))
	if contextText == "" {
		return blob, question
	}
	return contextText, question
}

// MessagesHaveToolScaffolding reports whether any message carries tool protocol chrome.
func MessagesHaveToolScaffolding(msgs []gjson.Result) bool {
	for _, m := range msgs {
		if MessageHasToolScaffolding(m) {
			return true
		}
	}
	return false
}

// ShouldSkipDeepForChatTurn is true when Deep must not rewrite this request:
// not a trailing user turn, or message-level cache_control, or a trailing user turn
// that is only tool scaffolding with no compressible plain text.
//
// Anthropic Official: a user message may be [tool_result, ..., text]. Prior code
// skipped Deep whenever any tool_result was present, so agent follow-up questions
// never got LLMLingua. Last-user-only Deep already preserves non-text parts
// (tool_result / mcp / images) and only rewrites text - so tool chrome + text is OK.
func ShouldSkipDeepForChatTurn(msgs []gjson.Result) bool {
	if len(msgs) == 0 {
		return true
	}
	last := msgs[len(msgs)-1]
	role := strings.ToLower(strings.TrimSpace(last.Get("role").String()))
	if role != "user" {
		return true
	}
	// Anthropic cache_control OR OpenAI prompt_cache_breakpoint at message level:
	// mutating the turn would invalidate the reusable prefix (official prompt-caching).
	if last.Get("cache_control").Exists() || last.Get("prompt_cache_breakpoint").Exists() {
		return true
	}
	// Official OpenAI Chat Completions: message-level annotations / refusal on the
	// trailing turn must stay verbatim (same smash class as content-part citations).
	if last.Get("annotations").Exists() {
		return true
	}
	if r := last.Get("refusal"); r.Exists() && r.Type != gjson.Null && strings.TrimSpace(r.String()) != "" {
		return true
	}
	// Official Chat Completions audio round-trip: message.audio pairs with content
	// transcript. Deep rewriting the text desyncs the audio id reference.
	if last.Get("audio").Exists() && last.Get("audio").Type != gjson.Null {
		return true
	}
	// Official MiniMax: <think> spans in content must stay verbatim (interleaved thinking).
	if contentHasMiniMaxThinkTags(last) {
		return true
	}
	if MessageHasToolScaffolding(last) {
		// Pure tool_result / tool-only user turn: nothing for LLMLingua.
		return strings.TrimSpace(lastUserDeepPlainText(last)) == ""
	}
	return false
}

// contentHasMiniMaxThinkTags reports MiniMax interleaved-thinking markers that must
// never enter LLMLingua (official: preserve <think>…</think> / reasoning_details).
func contentHasMiniMaxThinkTags(m gjson.Result) bool {
	c := m.Get("content")
	if c.Type == gjson.String {
		s := c.String()
		return strings.Contains(s, "<think>") || strings.Contains(s, "</think>")
	}
	if !c.IsArray() {
		return false
	}
	for _, part := range c.Array() {
		if t := part.Get("text"); t.Exists() {
			s := t.String()
			if strings.Contains(s, "<think>") || strings.Contains(s, "</think>") {
				return true
			}
		}
	}
	return false
}

// MessageHasToolScaffolding detects one message that carries tool protocol chrome.
func MessageHasToolScaffolding(m gjson.Result) bool {
	role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
	switch role {
	case "tool", "function":
		return true
	}
	if m.Get("tool_calls").Exists() || m.Get("function_call").Exists() {
		return true
	}
	if strings.TrimSpace(m.Get("tool_call_id").String()) != "" {
		return true
	}
	content := m.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if isToolProtocolPartType(strings.ToLower(strings.TrimSpace(part.Get("type").String()))) {
			return true
		}
	}
	return false
}

// isToolProtocolPartType matches Anthropic/OpenAI tool protocol content-block types.
// Includes MCP + server tools; suffix heuristic covers future *_tool_use / *_tool_result
// blocks (code execution, web fetch, text editor, tool search, …) without inventing names.
func isToolProtocolPartType(typ string) bool {
	switch typ {
	case "tool_use", "tool_result", "server_tool_use", "web_search_tool_result",
		"function_call", "function_call_output", "tool_call",
		"mcp_tool_use", "mcp_tool_result", "mcp_tool_listing", "tool_reference":
		return true
	}
	if strings.HasSuffix(typ, "_tool_result") || strings.HasSuffix(typ, "_tool_use") {
		return true
	}
	return false
}

func stripSystemReminderBlocks(s string) string {
	const open, closeTag = "<system-reminder>", "</system-reminder>"
	for {
		i := strings.Index(s, open)
		if i < 0 {
			return strings.TrimSpace(s)
		}
		rest := s[i+len(open):]
		j := strings.Index(rest, closeTag)
		if j < 0 {
			return strings.TrimSpace(s[:i])
		}
		s = s[:i] + rest[j+len(closeTag):]
	}
}

// extractSystemReminderBlocks returns Claude Code reminder blocks in order (with tags).
func extractSystemReminderBlocks(s string) []string {
	const open, closeTag = "<system-reminder>", "</system-reminder>"
	var out []string
	for {
		i := strings.Index(s, open)
		if i < 0 {
			return out
		}
		rest := s[i+len(open):]
		j := strings.Index(rest, closeTag)
		if j < 0 {
			out = append(out, s[i:])
			return out
		}
		out = append(out, open+rest[:j]+closeTag)
		s = rest[j+len(closeTag):]
	}
}

// reassembleStringContentWithReminders keeps embedded <system-reminder> blocks verbatim
// when Deep rewrites string-shaped user content (Claude Code sometimes uses string, not array).
func reassembleStringContentWithReminders(raw, compressed string) string {
	reminders := extractSystemReminderBlocks(raw)
	if len(reminders) == 0 {
		return compressed
	}
	var b strings.Builder
	for _, r := range reminders {
		b.WriteString(r)
		b.WriteByte('\n')
	}
	b.WriteString(compressed)
	return b.String()
}

// contentPartFrozenForDeep reports parts that must stay verbatim for all doors:
// Claude Code <system-reminder>, Anthropic cache_control, OpenAI prompt_cache_breakpoint
// (official Chat Completions explicit prompt caching), Gemini thought signatures
// (thought_signature / thoughtSignature / extra_content.google), and multimodal / RAG
// blocks (parity with proxy.optimizePayload messageHasMultimodalParts).
func contentPartFrozenForDeep(part gjson.Result) bool {
	if part.Get("cache_control").Exists() {
		return true
	}
	// OpenAI official: prompt_cache_breakpoint marks the end of a reusable prefix.
	// Deep must not rewrite/drop that part (same smash class as Anthropic cache_control).
	if part.Get("prompt_cache_breakpoint").Exists() {
		return true
	}
	// Anthropic citations / OpenAI annotations: mutating text orphans structured refs.
	if part.Get("citations").Exists() || part.Get("annotations").Exists() {
		return true
	}
	if partHasThoughtSignature(part) {
		return true
	}
	typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
	switch typ {
	case "search_result", "document", "image", "image_url", "container_upload",
		"file", "input_audio", "audio", "inline_data", "inlinedata", "refusal",
		// Responses leftovers before Chat migrate (GPT/Cursor BYOK); never feed to LLMLingua.
		"input_image", "input_file",
		// Official Anthropic browser toolset state block (inside tool_result / user content).
		"browser_state",
		// Mistral ThinkChunk / Anthropic thinking / OpenAI reasoning parts (defense if
		// they ever appear on a trailing user turn - never smash into LLMLingua).
		"thinking", "redacted_thinking", "reasoning":
		return true
	}
	if t := part.Get("text"); t.Exists() && strings.Contains(t.String(), "<system-reminder>") {
		return true
	}
	// Official MiniMax openai-compat: thinking may be embedded as <think>…</think>
	// in text (reasoning_split=false). Never feed those spans to LLMLingua.
	if t := part.Get("text"); t.Exists() {
		s := t.String()
		if strings.Contains(s, "<think>") || strings.Contains(s, "</think>") {
			return true
		}
	}
	return false
}

// partIsCacheBreakpoint marks Anthropic cache_control or OpenAI prompt_cache_breakpoint.
// Official: cache prefix is everything up to and including this block; mutating any
// earlier sibling in the same message invalidates the reusable prefix (Anthropic + OpenAI).
func partIsCacheBreakpoint(part gjson.Result) bool {
	return part.Get("cache_control").Exists() || part.Get("prompt_cache_breakpoint").Exists()
}

// arrayHasCacheBreakpointAfter reports whether any part after idx is a cache breakpoint.
func arrayHasCacheBreakpointAfter(parts []gjson.Result, idx int) bool {
	for i := idx + 1; i < len(parts); i++ {
		if partIsCacheBreakpoint(parts[i]) {
			return true
		}
	}
	return false
}

// partHasThoughtSignature detects Gemini thinking signatures that must be returned verbatim
// (official: omit → HTTP 400 on function-calling turns). Also freezes thought:true
// summary parts (official Gemini thinking: do not merge/mutate thought parts).
func partHasThoughtSignature(part gjson.Result) bool {
	if strings.TrimSpace(part.Get("thought_signature").String()) != "" {
		return true
	}
	if strings.TrimSpace(part.Get("thoughtSignature").String()) != "" {
		return true
	}
	if strings.TrimSpace(part.Get("extra_content.google.thought_signature").String()) != "" {
		return true
	}
	if part.Get("thought").Bool() {
		return true
	}
	return false
}

// lastUserDeepPlainText is the text Deep may rewrite: last-user text parts
// excluding Claude Code <system-reminder> and any cache_control parts (those stay verbatim).
func lastUserDeepPlainText(m gjson.Result) string {
	content := m.Get("content")
	switch {
	case content.Type == gjson.String:
		// Message-level cache_control is handled by ShouldSkipDeepForChatTurn.
		return stripSystemReminderBlocks(content.String())
	case content.IsArray():
		parts := content.Array()
		var chunks []string
		for i, part := range parts {
			if contentPartFrozenForDeep(part) {
				continue
			}
			// Official Anthropic/OpenAI: never feed text that sits before a later
			// cache breakpoint into LLMLingua (byte change invalidates the prefix).
			if arrayHasCacheBreakpointAfter(parts, i) {
				continue
			}
			t := part.Get("text")
			if !t.Exists() {
				continue
			}
			typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
			// Chat Completions: "text". Responses leftovers: input_text / output_text (same .text field).
			if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
				continue
			}
			chunks = append(chunks, t.String())
		}
		return strings.Join(chunks, "\n")
	default:
		return ""
	}
}

func rewriteLastUserContent(m gjson.Result, compressed string) any {
	content := m.Get("content")
	if !content.IsArray() {
		if content.Type == gjson.String {
			raw := content.String()
			if strings.Contains(raw, "<system-reminder>") {
				return reassembleStringContentWithReminders(raw, compressed)
			}
		}
		return compressed
	}
	parts := content.Array()
	out := make([]any, 0, len(parts)+1)
	inserted := false
	for i, part := range parts {
		if contentPartFrozenForDeep(part) || arrayHasCacheBreakpointAfter(parts, i) {
			out = append(out, part.Value())
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
			// Multimodal / tool / RAG parts stay in original order (GPT/Gemini vision).
			out = append(out, part.Value())
			continue
		}
		// Compressible text: replace the first slot in-place so images/files between
		// text parts are not shoved before the question (official multimodal order).
		if !inserted {
			out = append(out, map[string]any{
				"type": "text",
				"text": compressed,
			})
			inserted = true
		}
		// Later compressible text parts are merged into the single compressed block.
	}
	if !inserted {
		out = append(out, map[string]any{
			"type": "text",
			"text": compressed,
		})
	}
	return out
}

func lastUserQuestion(msgs []gjson.Result) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		role := strings.ToLower(strings.TrimSpace(msgs[i].Get("role").String()))
		if role == "user" {
			// Same freeze rules as Deep rewrite - never treat signed/reminder chrome as the question.
			return strings.TrimSpace(lastUserDeepPlainText(msgs[i]))
		}
	}
	return ""
}

func estimateTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}

// SkipLiveDeepForRequestPath is true for proxy paths where live Deep must not run (Fast-only).
// Anthropic /v1/messages/count_tokens is a companion RPC: running LLMLingua there adds
// latency and can disagree with the subsequent /v1/messages body when Deep fail-closed differs.
func SkipLiveDeepForRequestPath(path string) bool {
	pl := strings.ToLower(strings.TrimSpace(path))
	return strings.Contains(pl, "count_tokens")
}

// EstimateLiveDeepInputTokens estimates compressible chat text tokens for live when-to-invoke.
// Uses trailing-user plain text only (not system/assistant/developer/tool, not <system-reminder>).
// Matches Compress* so agent chrome cannot falsely trigger Deep; prior tool history alone does not block.
func EstimateLiveDeepInputTokens(body []byte, path string) int {
	_ = path
	if !gjson.ValidBytes(body) {
		return 0
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return 0
	}
	arr := messages.Array()
	if ShouldSkipDeepForChatTurn(arr) {
		return 0
	}
	last := arr[len(arr)-1]
	// Gate must match Compress*: only last-user plain text (not frozen signatures/reminders).
	// Using messageCompressibleText here historically counted Gemini/Anthropic chrome and
	// falsely triggered Deep - same class as the Claude Code system-token gate bug.
	plain := strings.TrimSpace(lastUserDeepPlainText(last))
	if plain == "" {
		return 0
	}
	if LastUserUnsafeForDeepLLMLingua(plain) {
		return 0
	}
	return estimateTokens([]byte(plain))
}

// LastUserUnsafeForDeepLLMLingua reports last-user text that must not enter LLMLingua.
// Microsoft LLMLingua removes tokens by design; on coding-agent turns that can corrupt
// file paths, URLs, fenced code, or dense JSON - producing “garbled” behavior even when
// system/tools chrome is correctly frozen (Claude Code / Cursor / GPT / Gemini / DeepSeek).
// Fail-closed to Fast-only for every door.
func LastUserUnsafeForDeepLLMLingua(blob string) bool {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return false
	}
	if strings.Contains(blob, "```") {
		return true
	}
	if strings.Contains(blob, "://") {
		return true
	}
	// Windows UNC / extended paths: \\server\share, \\?\C:\...
	// LLMLingua often mangles backslash runs (same garble class as drive letters).
	if strings.Contains(blob, `\\`) {
		return true
	}
	// Windows absolute paths: C:\... or D:/...
	for i := 0; i+2 < len(blob); i++ {
		c := blob[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			if blob[i+1] == ':' && (blob[i+2] == '\\' || blob[i+2] == '/') {
				return true
			}
		}
	}
	for _, p := range []string{
		"/Users/", "/home/", "/var/", "/tmp/", "/etc/", "/opt/", "/usr/", "/mnt/",
		"/workspace/", "/Workspace/",
		// Windows roots without drive letter (shells / IDE paste).
		`\Users\`, `\Windows\`, `\Program Files\`, `\ProgramData\`,
	} {
		if strings.Contains(blob, p) {
			return true
		}
	}
	// Dense JSON pasted into the user turn (tool dumps / schemas) - tokenize risk.
	if strings.Count(blob, `{"`) >= 2 || strings.Count(blob, `":`) >= 4 {
		return true
	}
	// Relative code paths: src/foo.go, .\pkg\bar.ts, ../lib/x.py, or dir trees like pkg/proxy
	if lastUserHasRelativeCodePath(blob) {
		return true
	}
	return false
}

func lastUserHasRelativeCodePath(blob string) bool {
	lower := strings.ToLower(blob)
	// Known coding-tree prefixes (Claude Code / Cursor / GPT agents) even without a file ext.
	for _, prefix := range []string{
		"src/", "src\\", "pkg/", "pkg\\", "lib/", "lib\\", "cmd/", "cmd\\",
		"internal/", "internal\\", "components/", "components\\", "hooks/", "hooks\\",
		"pages/", "pages\\", "services/", "services\\", "utils/", "utils\\",
		"scripts/", "scripts\\", "tests/", "tests\\", "test/", "test\\",
		"./src/", ".\\src\\", "../", "..\\",
	} {
		if strings.Contains(lower, prefix) {
			return true
		}
	}
	exts := []string{
		".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
		".py", ".rs", ".java", ".kt", ".cs", ".cpp", ".cc", ".c", ".h", ".hpp",
		".json", ".yaml", ".yml", ".toml", ".md", ".sql", ".sh", ".ps1",
		".rb", ".php", ".swift", ".dart",
	}
	for _, ext := range exts {
		start := 0
		for {
			idx := strings.Index(lower[start:], ext)
			if idx < 0 {
				break
			}
			idx += start
			// Walk left for path separators before the extension.
			j := idx - 1
			for j >= 0 {
				c := lower[j]
				if c == '/' || c == '\\' {
					return true
				}
				if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' {
					j--
					continue
				}
				break
			}
			start = idx + len(ext)
		}
	}
	return false
}

// DeepExpanded reports whether Deep grew the payload vs Fast-only input (fail-closed gate).
// Must stay the logical inverse of proxy.shouldAcceptDeepResult (same checks / order).
func DeepExpanded(fastBody, deepBody []byte, originTokens, compressedTokens int) bool {
	if len(deepBody) == 0 {
		return true
	}
	if len(deepBody) > len(fastBody) {
		return true
	}
	if originTokens > 0 && compressedTokens >= originTokens {
		return true
	}
	if estimateTokens(deepBody) > estimateTokens(fastBody) {
		return true
	}
	return false
}
