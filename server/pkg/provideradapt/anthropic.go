package provideradapt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// TranslateRequest converts an OpenAI chat-completions body to the adapter dialect.
func TranslateRequest(cfg AdapterConfig, openaiBody []byte, upstreamModel string) (path string, body []byte, err error) {
	path = strings.TrimSpace(cfg.UpstreamPath)
	if path == "" {
		return "", nil, fmt.Errorf("upstream_path required")
	}
	switch strings.TrimSpace(cfg.Dialect) {
	case DialectAnthropicMessages:
		out, err := openAIChatToAnthropicMessages(openaiBody, upstreamModel, cfg.DefaultMaxTokens)
		return path, out, err
	case DialectOpenAICompat:
		out, err := openAICompatPassthrough(openaiBody, upstreamModel)
		return path, out, err
	default:
		return "", nil, fmt.Errorf("unsupported dialect %q", cfg.Dialect)
	}
}

// TranslateResponseBody maps a non-stream upstream body back to OpenAI chat shape.
func TranslateResponseBody(cfg AdapterConfig, upstreamBody []byte, statusCode int) ([]byte, error) {
	switch strings.TrimSpace(cfg.Dialect) {
	case DialectAnthropicMessages:
		return anthropicMessagesToOpenAIChat(upstreamBody, statusCode)
	case DialectOpenAICompat:
		return upstreamBody, nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q", cfg.Dialect)
	}
}

// NeedsResponseTranslate is true when the adapter changes response shape (non-passthrough dialects).
func NeedsResponseTranslate(cfg AdapterConfig) bool {
	return strings.TrimSpace(cfg.Dialect) == DialectAnthropicMessages
}

func openAICompatPassthrough(body []byte, upstreamModel string) ([]byte, error) {
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid JSON body")
	}
	upstreamModel = strings.TrimSpace(upstreamModel)
	if upstreamModel == "" {
		return nil, fmt.Errorf("model required")
	}
	cur := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	out := body
	if cur != upstreamModel {
		var err error
		out, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, err
		}
	}
	// Official DeepSeek: thinking mode rejects tool_choice=required / named-function
	// (HTTP 400). V4 enables thinking by default - Cursor/GPT agents that force tools
	// must be fail-closed to "auto" or the whole door 400s (flexibility class).
	out = normalizeDeepSeekToolChoiceForThinking(out)
	// Official DeepSeek thinking + tools: every prior assistant turn must pass back
	// reasoning_content (missing/"" → HTTP 400 on multi-turn / V4 Pro). Heal here.
	out = EnsureDeepSeekToolCallReasoningContent(out)
	// Official Mistral / mistral-common: tool_call id must be ^[a-zA-Z0-9]{9}$.
	// Cursor/OpenAI call_… ids → HTTP 400. Remap assistant + tool messages together.
	out = EnsureMistralToolCallIDs(out)
	// Official Mistral openai-compat: store / max_completion_tokens / logit_bias /
	// logprobs → 422. Fail-closed strip + map (keep presence/frequency_penalty +
	// reasoning_effort - listed on docs.mistral.ai Chat Completions).
	out = SanitizeMistralOpenAICompatRequest(out)
	// Official Gemini OpenAI-compat: tool_choice is none|auto|required only.
	// allowed_tools / custom / named-function objects 400 - map to closest Chat string.
	out = normalizeGeminiToolChoiceForOpenAICompat(out)
	// Official Gemini openai-compat: assistant tool_calls with content:null → 400
	// ("Expected string or list of content parts, got: null"). Heal before upstream.
	out = EnsureGeminiAssistantToolCallContent(out)
	return out, nil
}

// normalizeGeminiToolChoiceForOpenAICompat maps OpenAI-only tool_choice object shapes
// onto Gemini-supported string values (none|auto|required). Official Gemini OpenAI
// compatibility docs do not list allowed_tools / custom / function objects.
func normalizeGeminiToolChoiceForOpenAICompat(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !strings.Contains(model, "gemini") {
		return body
	}
	tc := gjson.GetBytes(body, "tool_choice")
	if !tc.Exists() || tc.Type == gjson.Null {
		return body
	}
	if tc.Type == gjson.String {
		switch strings.ToLower(strings.TrimSpace(tc.String())) {
		case "none", "auto", "required":
			return body
		case "any":
			out, err := sjson.SetBytes(body, "tool_choice", "required")
			if err != nil {
				return body
			}
			return out
		default:
			out, err := sjson.SetBytes(body, "tool_choice", "auto")
			if err != nil {
				return body
			}
			return out
		}
	}
	if !tc.IsObject() {
		return body
	}
	typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
	mapped := "auto"
	switch typ {
	case "none":
		mapped = "none"
	case "auto":
		mapped = "auto"
	case "required", "any":
		mapped = "required"
	case "function", "custom", "tool":
		// Forced named tool - closest Gemini string is required.
		mapped = "required"
	case "allowed_tools":
		if strings.EqualFold(strings.TrimSpace(tc.Get("mode").String()), "required") {
			mapped = "required"
		} else {
			mapped = "auto"
		}
	default:
		if tc.Get("function").Exists() || tc.Get("name").Exists() {
			mapped = "required"
		}
	}
	out, err := sjson.SetBytes(body, "tool_choice", mapped)
	if err != nil {
		return body
	}
	return out
}

// EnsureGeminiAssistantToolCallContent heals Gemini openai-compat history so
// assistant tool_calls turns never carry content:null / missing / "" - Google's
// OpenAI-compat endpoint returns HTTP 400 "Expected string or list of content
// parts, got: null" (and rejects empty string). Use a single space as the
// fail-closed placeholder (same class as DeepSeek reasoning_content " ").
// Also heals empty tool_calls[].id → function name so matching role=tool
// tool_call_id / name enrichment can succeed (live Gemini empty-id class).
func EnsureGeminiAssistantToolCallContent(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !strings.Contains(model, "gemini") {
		return body
	}
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.IsArray() {
		return body
	}
	out := body
	for i, m := range msgs.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "assistant" && role != "model" {
			continue
		}
		tcs := m.Get("tool_calls")
		if !tcs.IsArray() || len(tcs.Array()) == 0 {
			continue
		}
		content := m.Get("content")
		needsContent := !content.Exists() || content.Type == gjson.Null ||
			(content.Type == gjson.String && strings.TrimSpace(content.String()) == "")
		if needsContent {
			var err error
			out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.content", i), " ")
			if err != nil {
				return body
			}
		}
		for j, tc := range tcs.Array() {
			id := strings.TrimSpace(tc.Get("id").String())
			if id != "" {
				continue
			}
			name := strings.TrimSpace(tc.Get("function.name").String())
			if name == "" {
				name = strings.TrimSpace(tc.Get("name").String())
			}
			if name == "" {
				name = fmt.Sprintf("tool_%d_%d", i, j)
			}
			// Gemini often returns id:""; use function name as stable id.
			var err error
			out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.tool_calls.%d.id", i, j), name)
			if err != nil {
				return body
			}
		}
	}
	// Pair role=tool messages that have empty tool_call_id with healed assistant ids
	// by matching tool message name to function name.
	msgs2 := gjson.GetBytes(out, "messages")
	if !msgs2.IsArray() {
		return out
	}
	for i, m := range msgs2.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "tool" && role != "function" {
			continue
		}
		tcid := strings.TrimSpace(m.Get("tool_call_id").String())
		if tcid != "" {
			continue
		}
		name := strings.TrimSpace(m.Get("name").String())
		if name == "" {
			continue
		}
		var err error
		out, err = sjson.SetBytes(out, fmt.Sprintf("messages.%d.tool_call_id", i), name)
		if err != nil {
			return body
		}
	}
	// Official Gemini openai-compat (Google forum): refusal:null → HTTP 400
	// "Value is not a string: null". OpenAI SDKs serialize missing refusal as null;
	// strip null refusal / empty tool_calls[] so history replay does not 400.
	msgs3 := gjson.GetBytes(out, "messages")
	if msgs3.IsArray() {
		for i, m := range msgs3.Array() {
			if r := m.Get("refusal"); r.Exists() && r.Type == gjson.Null {
				var err error
				out, err = sjson.DeleteBytes(out, fmt.Sprintf("messages.%d.refusal", i))
				if err != nil {
					return body
				}
			}
			if tcs := m.Get("tool_calls"); tcs.Exists() && tcs.IsArray() && len(tcs.Array()) == 0 {
				var err error
				out, err = sjson.DeleteBytes(out, fmt.Sprintf("messages.%d.tool_calls", i))
				if err != nil {
					return body
				}
			}
		}
	}
	return out
}

// normalizeDeepSeekToolChoiceForThinking downgrades forced tool_choice to "auto" when
// DeepSeek thinking mode is active. Official docs + live API: required / named function
// / any are rejected with "Thinking mode does not support this tool_choice".
// When thinking is explicitly disabled, forced choices pass through unchanged.
func normalizeDeepSeekToolChoiceForThinking(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !strings.Contains(model, "deepseek") {
		return body
	}
	if !deepSeekThinkingModeActive(body, model) {
		return body
	}
	tc := gjson.GetBytes(body, "tool_choice")
	if !tc.Exists() || tc.Type == gjson.Null {
		return body
	}
	forced := false
	switch {
	case tc.Type == gjson.String:
		switch strings.ToLower(strings.TrimSpace(tc.String())) {
		case "required", "any":
			forced = true
		case "auto", "none":
			return body
		}
	case tc.IsObject():
		typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
		switch typ {
		case "function", "custom", "tool", "any", "required":
			forced = true
		case "allowed_tools":
			// mode=required is a forced subset (same 400 class under thinking).
			if strings.EqualFold(strings.TrimSpace(tc.Get("mode").String()), "required") {
				forced = true
			}
		case "auto", "none":
			return body
		default:
			// Unknown object shapes that force a call - fail-closed.
			if tc.Get("function").Exists() || tc.Get("name").Exists() {
				forced = true
			}
		}
	}
	if !forced {
		return body
	}
	out, err := sjson.SetBytes(body, "tool_choice", "auto")
	if err != nil {
		return body
	}
	return out
}

// EnsureDeepSeekToolCallReasoningContent heals thinking-mode multi-turn
// reasoning_content passback for DeepSeek AND Kimi/Moonshot OpenAI-compat doors
// (official DeepSeek thinking docs + Kimi Code error reference):
//
// Official DeepSeek thinking_mode docs (Tool Calls):
//   - When the request carries tools, reasoning_content of ALL prior assistant turns
//     must be passed back - even turns without tool_calls.
//   - Missing field → HTTP 400 "reasoning_content in the thinking mode must be passed back".
//
// Official Kimi Code: "thinking is enabled but reasoning_content is missing in
// assistant tool call message" → same heal with non-empty placeholder " ".
//
// Live V4 Pro / Kimi K2.6 quirk (Hermes / OpenClaw / pi 2026): some deployments reject
// reasoning_content:"" as "not passed back". Use a single space " " as the
// fail-closed placeholder when the field is missing or empty - never fabricate prose.
//
// Also: assistant tool_calls with content:"" (empty string) is rejected by strict
// DeepSeek validators; OpenAI allows null - coerce "" → null on those turns.
func EnsureDeepSeekToolCallReasoningContent(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !needsReasoningContentPassback(model) {
		return body
	}
	if !reasoningContentPassbackThinkingActive(body, model) {
		return body
	}
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.IsArray() {
		return body
	}
	tools := gjson.GetBytes(body, "tools")
	hasTools := tools.IsArray() && len(tools.Array()) > 0
	if !hasTools {
		// Legacy functions[] is also a tools-carrying request for passback rules.
		fns := gjson.GetBytes(body, "functions")
		hasTools = fns.IsArray() && len(fns.Array()) > 0
	}
	// Without tools, passback is optional (docs: ignored). Still heal assistant
	// tool_calls / function_call turns - those always require reasoning_content.
	needAllAssistants := hasTools

	out := body
	for i, m := range msgs.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "assistant" && role != "model" {
			continue
		}
		hasToolCalls := false
		if tcs := m.Get("tool_calls"); tcs.IsArray() && len(tcs.Array()) > 0 {
			hasToolCalls = true
		}
		if !hasToolCalls && m.Get("function_call").Exists() {
			hasToolCalls = true
		}
		if !needAllAssistants && !hasToolCalls {
			continue
		}

		// Heal empty-string content on tool turns (strict DeepSeek OpenAI-compat).
		if hasToolCalls {
			c := m.Get("content")
			if c.Exists() && c.Type == gjson.String && c.String() == "" {
				path := fmt.Sprintf("messages.%d.content", i)
				var err error
				out, err = sjson.SetBytes(out, path, nil)
				if err != nil {
					return body
				}
				// Refresh message view after mutate.
				m = gjson.GetBytes(out, fmt.Sprintf("messages.%d", i))
			}
		}

		rc := m.Get("reasoning_content")
		if rc.Exists() && rc.Type == gjson.String && strings.TrimSpace(rc.String()) != "" {
			continue
		}
		// Missing, null, or empty/whitespace → single-space placeholder.
		path := fmt.Sprintf("messages.%d.reasoning_content", i)
		var err error
		out, err = sjson.SetBytes(out, path, " ")
		if err != nil {
			return body
		}
	}
	return out
}

// deepSeekThinkingModeActive reports whether this request will run in DeepSeek thinking
// mode (forced tool_choice unsupported; reasoning_content passback required with tools).
// Official docs: thinking is enabled by default; only explicit disable / effort=none turns it off.
func deepSeekThinkingModeActive(body []byte, modelLower string) bool {
	return reasoningContentPassbackThinkingActive(body, modelLower)
}

// needsReasoningContentPassback reports OpenAI-compat families that 400 when
// thinking-mode assistant history omits reasoning_content (DeepSeek + Kimi/Moonshot).
func needsReasoningContentPassback(modelLower string) bool {
	return strings.Contains(modelLower, "deepseek") ||
		strings.Contains(modelLower, "kimi") ||
		strings.Contains(modelLower, "moonshot") ||
		// Official MiniMax: M2.x thinking cannot be disabled; tool loops require
		// reasoning_content / full assistant content (incl. <think>) round-trip.
		strings.Contains(modelLower, "minimax")
}

// reasoningContentPassbackThinkingActive reports whether thinking mode is active
// for reasoning_content passback healing (DeepSeek default-on; Kimi when enabled /
// history already carries reasoning_content / k2.5–k2.6 thinking models).
func reasoningContentPassbackThinkingActive(body []byte, modelLower string) bool {
	if typ := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String())); typ != "" {
		return typ != "disabled"
	}
	if effort := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "reasoning_effort").String())); effort == "none" {
		return false
	}
	// DeepSeek: thinking enabled by default when omitted.
	if strings.Contains(modelLower, "deepseek") {
		return true
	}
	// Official MiniMax: M2.x thinking cannot be disabled; M3 defaults thinking on
	// unless thinking.type=disabled (already handled above).
	if strings.Contains(modelLower, "minimax") {
		return true
	}
	// Kimi/Moonshot: bare thinking{} or prior reasoning_content means thinking was on.
	if gjson.GetBytes(body, "thinking").Exists() {
		return true
	}
	msgs := gjson.GetBytes(body, "messages")
	if msgs.IsArray() {
		for _, m := range msgs.Array() {
			rc := m.Get("reasoning_content")
			if rc.Exists() && rc.Type == gjson.String && strings.TrimSpace(rc.String()) != "" {
				return true
			}
		}
	}
	if strings.Contains(modelLower, "k2.5") || strings.Contains(modelLower, "k2.6") ||
		strings.Contains(modelLower, "k2-5") || strings.Contains(modelLower, "k2-6") ||
		strings.Contains(modelLower, "thinking") {
		return true
	}
	return false
}

const mistralToolCallIDAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// isMistralFamilyModel reports models that enforce mistral-common tool_call_id rules
// (mistral / codestral / devstral / ministral / mixtral / pixtral).
func isMistralFamilyModel(modelLower string) bool {
	for _, needle := range []string{
		"mistral", "codestral", "devstral", "ministral", "mixtral", "pixtral", "magistral",
	} {
		if strings.Contains(modelLower, needle) {
			return true
		}
	}
	return false
}

func isValidMistralToolCallID(id string) bool {
	if len(id) != 9 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

// mistralCompatibleToolCallID maps an arbitrary OpenAI/Cursor tool call id onto
// mistral-common's ^[a-zA-Z0-9]{9}$ space. Deterministic so assistant.tool_calls[].id
// and role=tool tool_call_id stay paired across the remap.
func mistralCompatibleToolCallID(src string) string {
	src = strings.TrimSpace(src)
	if isValidMistralToolCallID(src) {
		return src
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(src))
	n := h.Sum64()
	out := make([]byte, 9)
	for i := 0; i < 9; i++ {
		out[i] = mistralToolCallIDAlphabet[n%62]
		n /= 62
		if n == 0 {
			// Re-seed from a rotated hash so short inputs still fill 9 chars.
			n = h.Sum64()>>uint((i+1)%13) | 1
		}
	}
	return string(out)
}

// EnsureMistralToolCallIDs rewrites tool_calls[].id and matching tool_call_id values
// to mistral-common's 9-char alphanumeric form. Also strips prefix:true from assistant
// messages that carry tool_calls (prefix is for text forcing, not tool loops - litellm
// / Mistral docs: unexpected roles after tool when prefix pollutes the turn).
func EnsureMistralToolCallIDs(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !isMistralFamilyModel(model) {
		return body
	}
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.IsArray() {
		return body
	}
	idMap := map[string]string{}
	used := map[string]string{} // newID → original (collision detect)
	mapID := func(raw string) string {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return raw
		}
		if v, ok := idMap[raw]; ok {
			return v
		}
		cand := mistralCompatibleToolCallID(raw)
		// Resolve rare hash collisions across distinct source ids.
		for salt := 0; salt < 62*62; salt++ {
			if owner, taken := used[cand]; !taken || owner == raw {
				break
			}
			b := []byte(cand)
			b[8] = mistralToolCallIDAlphabet[(int(b[8])+salt+1)%62]
			b[7] = mistralToolCallIDAlphabet[(int(b[7])+salt/62+1)%62]
			cand = string(b)
		}
		idMap[raw] = cand
		used[cand] = raw
		return cand
	}

	out := body
	for i, m := range msgs.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		switch role {
		case "assistant", "model":
			if tcs := m.Get("tool_calls"); tcs.IsArray() {
				for j, tc := range tcs.Array() {
					id := strings.TrimSpace(tc.Get("id").String())
					if id == "" {
						continue
					}
					newID := mapID(id)
					if newID == id {
						continue
					}
					path := fmt.Sprintf("messages.%d.tool_calls.%d.id", i, j)
					var err error
					out, err = sjson.SetBytes(out, path, newID)
					if err != nil {
						return body
					}
				}
				// prefix:true + tool_calls breaks Mistral multi-step tool loops.
				if m.Get("prefix").Bool() {
					path := fmt.Sprintf("messages.%d.prefix", i)
					var err error
					out, err = sjson.SetBytes(out, path, false)
					if err != nil {
						return body
					}
				}
			}
		case "tool", "function":
			id := strings.TrimSpace(m.Get("tool_call_id").String())
			if id == "" {
				continue
			}
			newID := mapID(id)
			if newID == id {
				continue
			}
			path := fmt.Sprintf("messages.%d.tool_call_id", i)
			var err error
			out, err = sjson.SetBytes(out, path, newID)
			if err != nil {
				return body
			}
		}
	}
	return out
}

// SanitizeMistralOpenAICompatRequest strips OpenAI-only request fields that Mistral's
// Chat Completions endpoint rejects with HTTP 422 (store, max_completion_tokens,
// logit_bias, logprobs), migrates max_completion_tokens → max_tokens, maps
// tool_choice "required" → "any" (Mistral-native synonym; both accepted officially),
// and clamps temperature into [0,1] (OpenAI allows 0–2; Mistral schema is [0,1]).
// Keeps presence_penalty / frequency_penalty / reasoning_effort / n - listed on
// docs.mistral.ai Chat Completions (openclaw's older blanket strip was over-aggressive).
func SanitizeMistralOpenAICompatRequest(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !isMistralFamilyModel(model) {
		return body
	}
	out := body

	// max_completion_tokens is OpenAI reasoning chrome - Mistral wants max_tokens.
	if mct := gjson.GetBytes(out, "max_completion_tokens"); mct.Exists() {
		if !gjson.GetBytes(out, "max_tokens").Exists() && mct.Int() > 0 {
			var err error
			out, err = sjson.SetBytes(out, "max_tokens", mct.Int())
			if err != nil {
				return body
			}
		}
		var err error
		out, err = sjson.DeleteBytes(out, "max_completion_tokens")
		if err != nil {
			return body
		}
	}

	// OpenAI-only / rejected by Mistral openai-compat (openclaw live 422 class).
	for _, key := range []string{"store", "logit_bias", "logprobs", "top_logprobs"} {
		if !gjson.GetBytes(out, key).Exists() {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, key)
		if err != nil {
			return body
		}
	}

	// Official Mistral: tool_choice accepts "any" | "required"; map OpenAI "required"
	// onto Mistral-native "any" for older gateway validators that only list any.
	tc := gjson.GetBytes(out, "tool_choice")
	if tc.Exists() && tc.Type != gjson.Null {
		switch {
		case tc.Type == gjson.String && strings.EqualFold(strings.TrimSpace(tc.String()), "required"):
			var err error
			out, err = sjson.SetBytes(out, "tool_choice", "any")
			if err != nil {
				return body
			}
		case tc.IsObject():
			typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
			if typ == "required" {
				var err error
				out, err = sjson.SetBytes(out, "tool_choice.type", "any")
				if err != nil {
					return body
				}
			}
		}
	}

	// Temperature must be in [0,1] (422 if >1 like OpenAI's 0–2 range).
	if t := gjson.GetBytes(out, "temperature"); t.Exists() && t.Type == gjson.Number {
		v := t.Float()
		if v > 1 {
			var err error
			out, err = sjson.SetBytes(out, "temperature", 1.0)
			if err != nil {
				return body
			}
		} else if v < 0 {
			var err error
			out, err = sjson.SetBytes(out, "temperature", 0.0)
			if err != nil {
				return body
			}
		}
	}
	return out
}

func openAIChatToAnthropicMessages(body []byte, model string, defaultMaxTokens int) ([]byte, error) {
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid JSON body")
	}
	if defaultMaxTokens < 1 {
		return nil, fmt.Errorf("default_max_tokens required")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("model required")
	}

	var systemBlocks []map[string]any
	var msgs []map[string]any
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return nil, fmt.Errorf("messages array required")
	}
	// Tool_calls that openai_to_anthropic will not emit as tool_use (custom / empty name)
	// must also drop matching role=tool messages - otherwise orphan tool_result blocks
	// cause official Anthropic HTTP 400 (unexpected tool_use_id).
	skippedToolCallIDs := collectSkippedOpenAIToolCallIDs(messages)
	// Official Anthropic computer/browser: tool_result must echo toolset_name from the
	// matching tool_use. Chat histories often put toolset_name only on tool_calls -
	// inherit onto tool_result before upstream (omit → HTTP 400).
	toolsetByCallID := collectOpenAIToolsetNamesByCallID(messages)
	// Official Anthropic mid-conversation system messages (2026): after the first
	// user/assistant/tool turn, role:system must stay IN messages (not folded into
	// top-level system) so prompt-cache prefixes remain valid. Leading system/developer
	// still map to top-level system (classic Chat Completions → Messages shape).
	seenNonSystem := false
	for _, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		content := m.Get("content")
		// Official OpenAI: role=function is deprecated but still appears in older
		// GPT/Cursor histories. Gemini openai_compat accepts it for compatibility.
		// Map to tool - never 400 the whole openai_to_anthropic turn.
		if role == "function" {
			role = "tool"
		}
		// Rare native Gemini leak into Chat-shaped body.
		if role == "model" {
			role = "assistant"
		}
		switch role {
		case "system", "developer":
			blocks := openAISystemContentToAnthropicBlocks(content)
			if len(blocks) == 0 {
				continue
			}
			if !seenNonSystem {
				// Leading system/developer → top-level Anthropic system.
				// Preserve cache_control / map OpenAI prompt_cache_breakpoint.
				systemBlocks = append(systemBlocks, blocks...)
				continue
			}
			// Mid-conversation: keep role:system in messages (developer maps to system).
			am := map[string]any{"role": "system", "content": anthropicSystemField(blocks)}
			if cc := m.Get("cache_control"); cc.Exists() {
				am["cache_control"] = cc.Value()
			}
			msgs = append(msgs, am)
		case "user", "assistant", "tool":
			seenNonSystem = true
			msgIn := m
			if role == "tool" {
				tid := strings.TrimSpace(m.Get("tool_call_id").String())
				if tid == "" {
					// Legacy role=function: enrich tool_call_id from prior tool_use name.
					if id := lookupAnthropicToolUseIDByName(msgs, strings.TrimSpace(m.Get("name").String())); id != "" {
						if patched, perr := sjson.SetBytes([]byte(m.Raw), "tool_call_id", id); perr == nil {
							msgIn = gjson.ParseBytes(patched)
							tid = id
						}
					}
				}
				if tid != "" && skippedToolCallIDs[tid] {
					continue
				}
				if tid == "" {
					// Cannot emit orphan tool_result (Anthropic HTTP 400). Skip rather than
					// fail the whole turn - flexibility for legacy GPT function histories.
					continue
				}
			}
			am, err := openAIMessageToAnthropic(role, msgIn)
			if err != nil {
				if role == "tool" {
					continue
				}
				return nil, err
			}
			if am != nil {
				if role == "tool" {
					am = inheritToolsetNameOnAnthropicToolResult(am, toolsetByCallID)
				}
				msgs = append(msgs, am)
			}
		case "":
			// Empty role: skip (never invent role=user from chrome).
			continue
		default:
			return nil, fmt.Errorf("unsupported message role %q", role)
		}
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("at least one non-system message required")
	}
	msgs = mergeAdjacentSameRole(msgs)
	// Official Anthropic: tool_result / mcp_tool_result blocks must lead the user message
	// after an assistant tool_use. mergeAdjacentSameRole can append text first - reorder.
	for i := range msgs {
		msgs[i] = prioritizeToolResultBlocks(msgs[i])
	}

	maxTokens := defaultMaxTokens
	if v := gjson.GetBytes(body, "max_tokens"); v.Exists() && v.Int() > 0 {
		maxTokens = int(v.Int())
	} else if v := gjson.GetBytes(body, "max_completion_tokens"); v.Exists() && v.Int() > 0 {
		maxTokens = int(v.Int())
	}

	out := map[string]any{
		"model":      model,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}
	if sys := anthropicSystemField(systemBlocks); sys != nil {
		out["system"] = sys
	}
	if gjson.GetBytes(body, "stream").Bool() {
		out["stream"] = true
	}
	// Do not forward temperature/top_p: Anthropic's newer models reject non-default sampling
	// (official docs). Omit rather than invent or risk 400s from Cursor defaults.
	if t := gjson.GetBytes(body, "stop"); t.Exists() && t.Type != gjson.Null {
		out["stop_sequences"] = openAIStopToAnthropic(t)
	}
	if tools := gjson.GetBytes(body, "tools"); tools.IsArray() && len(tools.Array()) > 0 {
		atools, err := openAIToolsToAnthropic(tools)
		if err != nil {
			return nil, err
		}
		// After skipping Chat-only custom tools, omit empty tools[] (Anthropic rejects
		// tool_choice / empty tools chrome with no callable tools).
		if len(atools) > 0 {
			out["tools"] = atools
		}
	} else if fns := gjson.GetBytes(body, "functions"); fns.IsArray() && len(fns.Array()) > 0 {
		// Deprecated OpenAI functions[] (pre-tools). Defense when Normalize did not run.
		atools, err := openAIToolsToAnthropic(fns)
		if err != nil {
			return nil, err
		}
		if len(atools) > 0 {
			out["tools"] = atools
		}
	}
	if _, hasTools := out["tools"]; hasTools {
		if tc := gjson.GetBytes(body, "tool_choice"); tc.Exists() && tc.Type != gjson.Null {
			out["tool_choice"] = openAIToolChoiceToAnthropic(tc)
		} else if fc := gjson.GetBytes(body, "function_call"); fc.Exists() && fc.Type != gjson.Null {
			// Deprecated function_call → Anthropic tool_choice (string none/auto or {name}).
			out["tool_choice"] = openAIToolChoiceToAnthropic(fc)
		}
	}
	// Official Anthropic passthrough fields agents may send on the Chat Completions
	// body (Cursor / openai_to_anthropic). Never drop - rebuild must not strip chrome.
	forwardAnthropicPassthroughFields(body, out)
	// OpenAI Structured Outputs → Anthropic output_config.format when not already set.
	if _, has := out["output_config"]; !has {
		if oc := openAIResponseFormatToAnthropicOutputConfig(gjson.GetBytes(body, "response_format")); oc != nil {
			out["output_config"] = oc
		}
	}

	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	// Official Anthropic: thinking (enabled/adaptive) rejects forced tool_choice any/tool.
	// Cursor/GPT agents often send required/any with thinking - fail-closed to auto.
	raw = NormalizeAnthropicThinkingForModel(raw)
	// Official: forced tool_choice only conflicts with manual extended thinking
	// (type=enabled). Adaptive supports any/tool except a few models - run AFTER migrate.
	raw = NormalizeAnthropicToolChoiceForThinking(raw)
	raw = SanitizeAnthropicThinkingIncompatibleFields(raw)
	// Official Anthropic Claude 4.x: temperature and top_p cannot both be specified.
	raw = SanitizeAnthropicTemperatureTopPMutex(raw)
	// Official Anthropic: Claude 4.6+ / Mythos reject trailing assistant prefill (HTTP 400).
	raw = SanitizeAnthropicTrailingAssistantPrefill(raw)
	return raw, nil
}

// forwardAnthropicPassthroughFields copies official Messages top-level fields that
// openai_to_anthropic agents may include on a Chat Completions-shaped body.
// Deep/Fast already preserve these on native /v1/messages via sjson; the adapter
// rebuilds a new map and must not silently drop them (flexibility + cache/MCP/container).
func forwardAnthropicPassthroughFields(body []byte, out map[string]any) {
	for _, key := range []string{
		"thinking",
		"cache_control",
		"metadata",
		"container",
		"mcp_servers",
		"service_tier",
		"output_config",
		"output_format", // legacy; Anthropic still accepts briefly alongside output_config
		"context_management",
		"inference_geo",
		"user_profile_id",
		"speed",
		// Official Anthropic beta/stable request chrome agents may send on Chat door:
		// cache diagnostics + server-side refusal fallbacks. Dropping them silently
		// disables cache-miss diagnosis / fallback routing (flexibility class).
		"diagnostics",
		"fallbacks",
		"fallback_credit_token",
	} {
		v := gjson.GetBytes(body, key)
		if !v.Exists() || v.Type == gjson.Null {
			continue
		}
		out[key] = v.Value()
	}
}

// openAIResponseFormatToAnthropicOutputConfig maps Chat Completions response_format
// json_schema into Anthropic output_config.format (official structured outputs).
func openAIResponseFormatToAnthropicOutputConfig(rf gjson.Result) map[string]any {
	if !rf.Exists() || rf.Type == gjson.Null {
		return nil
	}
	typ := strings.ToLower(strings.TrimSpace(rf.Get("type").String()))
	if typ != "json_schema" {
		return nil
	}
	schema := rf.Get("json_schema.schema")
	if !schema.Exists() {
		schema = rf.Get("schema")
	}
	if !schema.Exists() {
		return nil
	}
	return map[string]any{
		"format": map[string]any{
			"type":   "json_schema",
			"schema": schema.Value(),
		},
	}
}

// openAISystemContentToAnthropicBlocks maps Chat Completions system/developer content
// into Anthropic TextBlockParam objects, preserving cache breakpoints.
func openAISystemContentToAnthropicBlocks(content gjson.Result) []map[string]any {
	if !content.Exists() || content.Type == gjson.Null {
		return nil
	}
	if content.Type == gjson.String {
		text := content.String()
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return []map[string]any{{"type": "text", "text": text}}
	}
	if !content.IsArray() {
		// Never dump structured non-array system chrome as text (garble class).
		return nil
	}
	var blocks []map[string]any
	for _, part := range content.Array() {
		if part.Type == gjson.String {
			text := part.String()
			if strings.TrimSpace(text) == "" {
				continue
			}
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
			continue
		}
		t := part.Get("text")
		if !t.Exists() {
			continue
		}
		text := t.String()
		if strings.TrimSpace(text) == "" {
			continue
		}
		block := map[string]any{"type": "text", "text": text}
		if cc := part.Get("cache_control"); cc.Exists() {
			block["cache_control"] = cc.Value()
		} else if part.Get("prompt_cache_breakpoint").Exists() {
			// OpenAI explicit breakpoint → Anthropic ephemeral cache_control on the same block.
			block["cache_control"] = map[string]any{"type": "ephemeral"}
		}
		blocks = append(blocks, block)
	}
	return blocks
}

// anthropicSystemField returns a plain string when no block chrome is needed, otherwise
// the official TextBlockParam array (required to carry cache_control).
func anthropicSystemField(blocks []map[string]any) any {
	if len(blocks) == 0 {
		return nil
	}
	needArray := false
	var texts []string
	for _, b := range blocks {
		if _, ok := b["cache_control"]; ok {
			needArray = true
		}
		if t, ok := b["text"].(string); ok {
			texts = append(texts, t)
		}
	}
	if !needArray {
		joined := strings.TrimSpace(strings.Join(texts, "\n"))
		if joined == "" {
			return nil
		}
		return joined
	}
	return blocks
}

func contentToPlain(content gjson.Result) string {
	if !content.Exists() || content.Type == gjson.Null {
		return ""
	}
	if content.Type == gjson.String {
		return content.String()
	}
	if content.IsArray() {
		var b strings.Builder
		for _, part := range content.Array() {
			typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
			// Chat Completions refusal → plain text (no Anthropic refusal block type).
			if typ == "refusal" {
				if r := strings.TrimSpace(part.Get("refusal").String()); r != "" {
					if b.Len() > 0 {
						b.WriteString("\n")
					}
					b.WriteString(r)
				}
				continue
			}
			// Only text / Responses leftovers. Never pull incidental .text from other blocks
			// (search_result / tool chrome) and never dump structured JSON as text.
			if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
				continue
			}
			if t := part.Get("text"); t.Exists() {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(t.String())
				continue
			}
			if part.Type == gjson.String {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(part.String())
			}
		}
		return b.String()
	}
	// Object/number/bool: never dump .Raw into a text field (same smash/garble class).
	return ""
}

// appendOpenAIMessageRefusalAsText maps official Chat Completions top-level
// message.refusal onto an Anthropic text block when content is null/empty.
// Without this, openai_to_anthropic silently drops refusal history (smash class).
func appendOpenAIMessageRefusalAsText(blocks []any, m gjson.Result) []any {
	refusal := strings.TrimSpace(m.Get("refusal").String())
	if refusal == "" {
		return blocks
	}
	for _, b := range blocks {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := bm["type"].(string); typ != "text" {
			continue
		}
		if t, _ := bm["text"].(string); strings.TrimSpace(t) == refusal {
			return blocks
		}
	}
	return append(blocks, map[string]any{"type": "text", "text": refusal})
}

// openAIToolContentToAnthropic maps OpenAI tool-message content into Anthropic
// tool_result.content (string or content-block array). Preserves search_result /
// image / document structure; never stringifies JSON Raw.
func openAIToolContentToAnthropic(content gjson.Result) any {
	if !content.Exists() || content.Type == gjson.Null {
		return ""
	}
	if content.Type == gjson.String {
		return content.String()
	}
	if content.IsArray() {
		var blocks []any
		for _, part := range content.Array() {
			typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
			switch typ {
			case "text", "input_text", "output_text", "":
				if t := part.Get("text"); t.Exists() {
					blocks = append(blocks, map[string]any{"type": "text", "text": t.String()})
				} else if part.Type == gjson.String {
					blocks = append(blocks, map[string]any{"type": "text", "text": part.String()})
				}
			case "image_url":
				url := part.Get("image_url.url").String()
				if url == "" {
					continue
				}
				src, err := openAIImageURLToAnthropicSource(url)
				if err != nil {
					continue
				}
				blocks = append(blocks, map[string]any{"type": "image", "source": src})
			case "image", "document", "search_result", "tool_reference", "container_upload",
				// Official Anthropic browser toolset: browser_state lives inside tool_result.
				"browser_state":
				blocks = append(blocks, part.Value())
			default:
				// Preserve unknown typed Anthropic-shaped parts; never stringify.
				if typ != "" {
					blocks = append(blocks, part.Value())
				}
			}
		}
		if len(blocks) == 0 {
			return ""
		}
		return blocks
	}
	// Structured object tool payloads: refuse Raw dump (garble). Empty is safer than JSON chrome as text.
	return ""
}

func openAIMessageToAnthropic(role string, m gjson.Result) (map[string]any, error) {
	switch role {
	case "tool":
		toolID := strings.TrimSpace(m.Get("tool_call_id").String())
		if toolID == "" {
			return nil, fmt.Errorf("tool message missing tool_call_id")
		}
		tr := map[string]any{
			"type":        "tool_result",
			"tool_use_id": toolID,
			"content":     openAIToolContentToAnthropic(m.Get("content")),
		}
		// Official Anthropic computer/browser toolsets: tool_result must echo toolset_name.
		if ts := strings.TrimSpace(m.Get("toolset_name").String()); ts != "" {
			tr["toolset_name"] = ts
		}
		// Official Anthropic: tool_result may carry cache_control (cache prefix through
		// that block). Also preserve is_error when clients set it.
		if cc := m.Get("cache_control"); cc.Exists() {
			tr["cache_control"] = cc.Value()
		} else if content := m.Get("content"); content.IsArray() {
			for _, part := range content.Array() {
				if cc := part.Get("cache_control"); cc.Exists() {
					tr["cache_control"] = cc.Value()
					break
				}
				if part.Get("prompt_cache_breakpoint").Exists() {
					tr["cache_control"] = map[string]any{"type": "ephemeral"}
					break
				}
			}
		}
		if m.Get("is_error").Exists() {
			tr["is_error"] = m.Get("is_error").Bool()
		}
		return map[string]any{
			"role":    "user",
			"content": []any{tr},
		}, nil
	case "assistant":
		var blocks []any
		content := m.Get("content")
		// Anthropic-shaped content already in the OpenAI message (thinking / tool_use / text):
		// pass through verbatim so signatures survive openai_to_anthropic round-trips.
		if content.IsArray() && assistantContentHasAnthropicBlocks(content) {
			for _, part := range content.Array() {
				blocks = append(blocks, sanitizeAnthropicContentBlock(part.Value()))
			}
			// Also append any OpenAI tool_calls not already represented as tool_use.
			existing := map[string]bool{}
			for _, b := range blocks {
				if bm, ok := b.(map[string]any); ok {
					if typ, _ := bm["type"].(string); typ == "tool_use" {
						if id, _ := bm["id"].(string); id != "" {
							existing[id] = true
						}
					}
				}
			}
			if tcs := m.Get("tool_calls"); tcs.IsArray() {
				for _, tc := range tcs.Array() {
					id := strings.TrimSpace(tc.Get("id").String())
					if id != "" && existing[id] {
						continue
					}
					if block := openAIToolCallToAnthropicToolUse(tc); block != nil {
						blocks = append(blocks, block)
						if id != "" {
							existing[id] = true
						}
					}
				}
			} else if fc := m.Get("function_call"); fc.IsObject() {
				name := strings.TrimSpace(fc.Get("name").String())
				if name != "" {
					id := strings.TrimSpace(fc.Get("id").String())
					if id == "" {
						id = "call_legacy_fc"
					}
					if !existing[id] {
						argsRaw := fc.Get("arguments").String()
						if argsRaw == "" && fc.Get("arguments").Exists() && fc.Get("arguments").Type != gjson.String {
							argsRaw = fc.Get("arguments").Raw
						}
						var input any = map[string]any{}
						if strings.TrimSpace(argsRaw) != "" {
							if err := json.Unmarshal([]byte(argsRaw), &input); err != nil {
								input = map[string]any{"raw": argsRaw}
							}
						}
						blocks = append(blocks, map[string]any{
							"type":  "tool_use",
							"id":    id,
							"name":  name,
							"input": input,
						})
					}
				}
			}
			blocks = appendOpenAIMessageRefusalAsText(blocks, m)
			if len(blocks) == 0 {
				blocks = append(blocks, map[string]any{"type": "text", "text": ""})
			}
			return map[string]any{"role": "assistant", "content": blocks}, nil
		}
		// Official Anthropic: thinking / redacted_thinking must lead the assistant turn
		// before text / tool_use. Prefer thinking_blocks (with signatures) over inventing
		// unsigned thinking from reasoning_content (unsigned → HTTP 400).
		if tbs := m.Get("thinking_blocks"); tbs.IsArray() {
			reasoning := strings.TrimSpace(m.Get("reasoning_content").String())
			var pending []map[string]any
			emptySigned := 0
			emptySignedIdx := -1
			for _, tb := range tbs.Array() {
				typ := strings.ToLower(strings.TrimSpace(tb.Get("type").String()))
				switch typ {
				case "thinking":
					thinkText := tb.Get("thinking").String()
					sig := strings.TrimSpace(tb.Get("signature").String())
					block := map[string]any{"type": "thinking", "thinking": thinkText}
					if sig != "" {
						block["signature"] = sig
					}
					if strings.TrimSpace(thinkText) == "" && sig != "" {
						emptySigned++
						emptySignedIdx = len(pending)
					}
					pending = append(pending, block)
				case "redacted_thinking":
					pending = append(pending, map[string]any{
						"type": "redacted_thinking",
						"data": tb.Get("data").String(),
					})
				}
			}
			// Streaming clients accumulate thinking text in reasoning_content and the
			// signature via thinking_blocks delta (thinking:""). Merge when unambiguous.
			if emptySigned == 1 && emptySignedIdx >= 0 && reasoning != "" {
				pending[emptySignedIdx]["thinking"] = reasoning
			}
			for _, b := range pending {
				blocks = append(blocks, b)
			}
		}
		text := contentToPlain(content)
		if content.IsArray() && openAIContentHasCacheChrome(content) {
			// Preserve per-part cache breakpoints instead of smashing to plain text.
			for _, part := range content.Array() {
				typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
				if typ == "refusal" {
					// Official Chat Completions refusal parts (may carry cache breakpoints).
					if r := strings.TrimSpace(part.Get("refusal").String()); r != "" {
						b := map[string]any{"type": "text", "text": r}
						if cc := part.Get("cache_control"); cc.Exists() {
							b["cache_control"] = cc.Value()
						} else if part.Get("prompt_cache_breakpoint").Exists() {
							b["cache_control"] = map[string]any{"type": "ephemeral"}
						}
						blocks = append(blocks, b)
					}
					continue
				}
				if typ != "" && typ != "text" && typ != "input_text" && typ != "output_text" {
					continue
				}
				if !part.Get("text").Exists() && part.Type != gjson.String {
					continue
				}
				blocks = append(blocks, openAITextPartToAnthropic(part))
			}
		} else if strings.TrimSpace(text) != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": text})
		}
		if tcs := m.Get("tool_calls"); tcs.IsArray() {
			for _, tc := range tcs.Array() {
				if block := openAIToolCallToAnthropicToolUse(tc); block != nil {
					blocks = append(blocks, block)
				}
			}
		} else if fc := m.Get("function_call"); fc.IsObject() {
			// Official OpenAI deprecated assistant.function_call (pre-tool_calls).
			// Without this, openai_to_anthropic silently drops the tool_use and the
			// following role=function/tool result cannot bind - same smash class as
			// custom-tool skip without orphan tool_result drop.
			name := strings.TrimSpace(fc.Get("name").String())
			if name != "" {
				id := strings.TrimSpace(fc.Get("id").String())
				if id == "" {
					id = "call_legacy_fc"
				}
				argsRaw := fc.Get("arguments").String()
				if argsRaw == "" && fc.Get("arguments").Exists() && fc.Get("arguments").Type != gjson.String {
					argsRaw = fc.Get("arguments").Raw
				}
				var input any = map[string]any{}
				if strings.TrimSpace(argsRaw) != "" {
					if err := json.Unmarshal([]byte(argsRaw), &input); err != nil {
						input = map[string]any{"raw": argsRaw}
					}
				}
				blocks = append(blocks, map[string]any{
					"type":  "tool_use",
					"id":    id,
					"name":  name,
					"input": input,
				})
			}
		}
		// Official Chat Completions: top-level message.refusal when content is null.
		blocks = appendOpenAIMessageRefusalAsText(blocks, m)
		if len(blocks) == 0 {
			blocks = append(blocks, map[string]any{"type": "text", "text": ""})
		}
		return map[string]any{"role": "assistant", "content": blocks}, nil
	case "user":
		content := m.Get("content")
		if content.IsArray() {
			var blocks []any
			for _, part := range content.Array() {
				typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
				switch typ {
				case "text", "input_text", "output_text", "":
					blocks = append(blocks, openAITextPartToAnthropic(part))
				case "image_url":
					url := part.Get("image_url.url").String()
					if url == "" {
						continue
					}
					src, serr := openAIImageURLToAnthropicSource(url)
					if serr != nil {
						// Skip bad images - never 400 the whole turn (same class as
						// input_audio: preserve text / real question for Claude).
						continue
					}
					blocks = append(blocks, map[string]any{"type": "image", "source": src})
				case "image", "document", "container_upload":
					// Already Anthropic-shaped (or Claude Code native) - pass through verbatim.
					blocks = append(blocks, part.Value())
				case "file":
					// OpenAI Chat Completions PDF file part → Anthropic document source.
					if doc := openAIFilePartToAnthropicDocument(part); doc != nil {
						blocks = append(blocks, doc)
					}
				case "input_audio", "audio":
					// OpenAI audio parts have no Messages equivalent; skip rather than
					// smash raw JSON into a text block (that garbles the real question).
					continue
				case "refusal":
					// Official Chat Completions refusal parts have no Messages block type.
					// Map to text so history survives; never forward type=refusal (Anthropic 400).
					if r := strings.TrimSpace(part.Get("refusal").String()); r != "" {
						b := map[string]any{"type": "text", "text": r}
						if cc := part.Get("cache_control"); cc.Exists() {
							b["cache_control"] = cc.Value()
						} else if part.Get("prompt_cache_breakpoint").Exists() {
							b["cache_control"] = map[string]any{"type": "ephemeral"}
						}
						blocks = append(blocks, b)
					}
					continue
				default:
					// Unknown typed part: preserve structure when possible; never dump Raw as text.
					if typ != "" {
						blocks = append(blocks, part.Value())
					} else if t := part.Get("text"); t.Exists() {
						blocks = append(blocks, map[string]any{"type": "text", "text": t.String()})
					}
				}
			}
			if len(blocks) == 0 {
				blocks = append(blocks, map[string]any{"type": "text", "text": ""})
			}
			// Official Anthropic: tool_result blocks must lead mixed user content.
			blocks = orderToolResultsFirst(blocks)
			return map[string]any{"role": "user", "content": blocks}, nil
		}
		return map[string]any{
			"role":    "user",
			"content": contentToPlain(content),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported role %q", role)
	}
}

func assistantContentHasAnthropicBlocks(content gjson.Result) bool {
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		switch strings.ToLower(strings.TrimSpace(part.Get("type").String())) {
		case "thinking", "redacted_thinking", "tool_use", "server_tool_use", "mcp_tool_use":
			return true
		}
	}
	return false
}

// openAITextPartToAnthropic maps a Chat Completions text part to Anthropic TextBlockParam,
// preserving Anthropic cache_control and mapping OpenAI prompt_cache_breakpoint.
func openAITextPartToAnthropic(part gjson.Result) map[string]any {
	text := ""
	if t := part.Get("text"); t.Exists() {
		text = t.String()
	} else if part.Type == gjson.String {
		text = part.String()
	}
	block := map[string]any{"type": "text", "text": text}
	if cc := part.Get("cache_control"); cc.Exists() {
		block["cache_control"] = cc.Value()
	} else if part.Get("prompt_cache_breakpoint").Exists() {
		block["cache_control"] = map[string]any{"type": "ephemeral"}
	}
	return block
}

func openAIContentHasCacheChrome(content gjson.Result) bool {
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		if part.Get("cache_control").Exists() || part.Get("prompt_cache_breakpoint").Exists() {
			return true
		}
	}
	return false
}

// lookupAnthropicToolUseIDByName finds the most recent tool_use id with the given name
// in already-translated Anthropic messages (for legacy OpenAI role=function enrichment).
func lookupAnthropicToolUseIDByName(msgs []map[string]any, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		role, _ := msgs[i]["role"].(string)
		if !strings.EqualFold(strings.TrimSpace(role), "assistant") {
			continue
		}
		for _, b := range asBlocks(msgs[i]["content"]) {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := bm["type"].(string)
			if typ != "tool_use" {
				continue
			}
			n, _ := bm["name"].(string)
			if strings.TrimSpace(n) != name {
				continue
			}
			if id, _ := bm["id"].(string); strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}
	return ""
}

// collectSkippedOpenAIToolCallIDs returns Chat tool_call ids that will not become
// Anthropic tool_use (custom tools, empty name, …). Matching role=tool results must
// be skipped to avoid orphan tool_result → Anthropic HTTP 400.
func collectSkippedOpenAIToolCallIDs(messages gjson.Result) map[string]bool {
	out := map[string]bool{}
	if !messages.IsArray() {
		return out
	}
	for _, m := range messages.Array() {
		tcs := m.Get("tool_calls")
		if !tcs.IsArray() {
			continue
		}
		for _, tc := range tcs.Array() {
			if openAIToolCallToAnthropicToolUse(tc) != nil {
				continue
			}
			id := strings.TrimSpace(tc.Get("id").String())
			if id != "" {
				out[id] = true
			}
		}
	}
	return out
}

// collectOpenAIToolsetNamesByCallID maps tool_call / tool_use ids → toolset_name from
// assistant history (Chat tool_calls.*.toolset_name or Anthropic-shaped content blocks).
func collectOpenAIToolsetNamesByCallID(messages gjson.Result) map[string]string {
	out := map[string]string{}
	if !messages.IsArray() {
		return out
	}
	for _, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		if role != "assistant" && role != "model" {
			continue
		}
		if tcs := m.Get("tool_calls"); tcs.IsArray() {
			for _, tc := range tcs.Array() {
				id := strings.TrimSpace(tc.Get("id").String())
				if id == "" {
					continue
				}
				ts := strings.TrimSpace(tc.Get("toolset_name").String())
				if ts == "" {
					ts = strings.TrimSpace(tc.Get("function.toolset_name").String())
				}
				if ts != "" {
					out[id] = ts
				}
			}
		}
		content := m.Get("content")
		if !content.IsArray() {
			continue
		}
		for _, part := range content.Array() {
			typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
			if typ != "tool_use" {
				continue
			}
			id := strings.TrimSpace(part.Get("id").String())
			ts := strings.TrimSpace(part.Get("toolset_name").String())
			if id != "" && ts != "" {
				out[id] = ts
			}
		}
	}
	return out
}

// inheritToolsetNameOnAnthropicToolResult copies toolset_name onto tool_result blocks
// that omitted it, using the matching prior tool_use / tool_calls id map.
func inheritToolsetNameOnAnthropicToolResult(am map[string]any, byID map[string]string) map[string]any {
	if am == nil || len(byID) == 0 {
		return am
	}
	content, ok := am["content"].([]any)
	if !ok {
		return am
	}
	for _, b := range content {
		bm, ok := b.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := bm["type"].(string)
		if strings.ToLower(strings.TrimSpace(typ)) != "tool_result" {
			continue
		}
		if existing, _ := bm["toolset_name"].(string); strings.TrimSpace(existing) != "" {
			continue
		}
		id, _ := bm["tool_use_id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if ts := strings.TrimSpace(byID[id]); ts != "" {
			bm["toolset_name"] = ts
		}
	}
	return am
}

// sanitizeAnthropicContentBlock strips request-illegal null caller from tool_use /
// server_tool_use blocks when replaying Anthropic-shaped content through the Chat door.
// Official: caller may appear on responses; caller:null on requests 400s Bedrock/strict hosts.
// Non-null programmatic callers and toolset_name are preserved.
func sanitizeAnthropicContentBlock(v any) any {
	bm, ok := v.(map[string]any)
	if !ok {
		return v
	}
	typ, _ := bm["type"].(string)
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "tool_use", "server_tool_use":
		if caller, has := bm["caller"]; has && caller == nil {
			delete(bm, "caller")
		}
	}
	return bm
}

func openAIToolCallToAnthropicToolUse(tc gjson.Result) map[string]any {
	typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
	// Official Chat Completions custom tool_calls have no Anthropic tool_use equivalent
	// (freeform custom.input ≠ JSON tool_use.input). Skip - never emit name:"" tool_use
	// (that 400s Claude and garbles the agent loop).
	if typ == "custom" || (typ == "" && tc.Get("custom").Exists()) {
		return nil
	}
	id := strings.TrimSpace(tc.Get("id").String())
	name := strings.TrimSpace(tc.Get("function.name").String())
	if name == "" {
		name = strings.TrimSpace(tc.Get("name").String())
	}
	if name == "" {
		return nil
	}
	argsRaw := tc.Get("function.arguments").String()
	var args any
	if strings.TrimSpace(argsRaw) == "" {
		args = map[string]any{}
	} else if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
		args = map[string]any{"raw": argsRaw}
	}
	block := map[string]any{
		"type":  "tool_use",
		"id":    id,
		"name":  name,
		"input": args,
	}
	// Official Anthropic: tool_use blocks may carry cache_control.
	if cc := tc.Get("cache_control"); cc.Exists() {
		block["cache_control"] = cc.Value()
	} else if cc := tc.Get("function.cache_control"); cc.Exists() {
		block["cache_control"] = cc.Value()
	}
	// Official Anthropic computer/browser toolsets: toolset_name must round-trip on
	// tool_use (and matching tool_result). Dropping it → HTTP 400 on member results.
	if ts := strings.TrimSpace(tc.Get("toolset_name").String()); ts != "" {
		block["toolset_name"] = ts
	} else if ts := strings.TrimSpace(tc.Get("function.toolset_name").String()); ts != "" {
		block["toolset_name"] = ts
	}
	// Preserve non-null caller (programmatic tool calling). Never forward caller:null
	// (Bedrock/strict Anthropic reject null caller on replay).
	if c := tc.Get("caller"); c.Exists() && c.Type != gjson.Null {
		block["caller"] = c.Value()
	} else if c := tc.Get("function.caller"); c.Exists() && c.Type != gjson.Null {
		block["caller"] = c.Value()
	}
	return block
}

func mergeAdjacentSameRole(msgs []map[string]any) []map[string]any {
	if len(msgs) < 2 {
		return msgs
	}
	var out []map[string]any
	for _, m := range msgs {
		if len(out) == 0 {
			out = append(out, m)
			continue
		}
		prev := out[len(out)-1]
		if prev["role"] == m["role"] {
			role, _ := prev["role"].(string)
			// Official Anthropic: assistant tool_use must be followed by user tool_result.
			// OpenAI may emit consecutive assistant tool_calls (parallel) - those merge.
			// Never merge a tool_use turn with a later text/thinking answer (that hides a
			// missing tool_result gap and 400s Claude / garbles the agent loop).
			if strings.EqualFold(strings.TrimSpace(role), "assistant") {
				prevTool := contentHasBlockType(prev["content"], "tool_use")
				curTool := contentHasBlockType(m["content"], "tool_use")
				if prevTool || curTool {
					if !(prevTool && curTool && assistantContentToolUseOnly(prev["content"]) &&
						assistantContentToolUseOnly(m["content"])) {
						out = append(out, m)
						continue
					}
				}
			}
			// Official Anthropic: in a user turn, tool_result blocks must come FIRST;
			// text/images after. OpenAI→Messages conversion can produce adjacent
			// user(text) + user(tool_result) that merge - reorder fail-closed.
			if strings.EqualFold(strings.TrimSpace(role), "user") {
				prev["content"] = appendUserContentToolResultsFirst(prev["content"], m["content"])
			} else {
				prev["content"] = appendContent(prev["content"], m["content"])
			}
			out[len(out)-1] = prev
			continue
		}
		out = append(out, m)
	}
	return out
}

func contentHasBlockType(content any, typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	for _, b := range asBlocks(content) {
		if blockType(b) == typ {
			return true
		}
	}
	return false
}

// assistantContentToolUseOnly reports an assistant turn that is only client tool_use
// blocks (OpenAI parallel tool_calls → adjacent assistants). Text/thinking/server
// chrome means this is not a safe merge target with another tool_use-only turn.
func assistantContentToolUseOnly(content any) bool {
	blocks := asBlocks(content)
	if len(blocks) == 0 {
		return false
	}
	for _, b := range blocks {
		typ := blockType(b)
		if typ != "tool_use" {
			return false
		}
	}
	return true
}

// prioritizeToolResultBlocks moves tool_result / mcp_tool_result blocks to the front
// of a user message (official Anthropic: results must immediately follow tool_use;
// leading non-result text → HTTP 400).
func prioritizeToolResultBlocks(msg map[string]any) map[string]any {
	role, _ := msg["role"].(string)
	if strings.ToLower(strings.TrimSpace(role)) != "user" {
		return msg
	}
	blocks := asBlocks(msg["content"])
	if len(blocks) < 2 {
		return msg
	}
	var tools, rest []any
	for _, b := range blocks {
		typ := blockType(b)
		switch typ {
		case "tool_result", "mcp_tool_result":
			tools = append(tools, b)
		default:
			if strings.HasSuffix(typ, "_tool_result") {
				tools = append(tools, b)
			} else {
				rest = append(rest, b)
			}
		}
	}
	if len(tools) == 0 {
		return msg
	}
	msg["content"] = append(tools, rest...)
	return msg
}

// SanitizeAnthropicTrailingAssistantPrefill heals Claude 4.6+ / Mythos / Fable /
// Claude 5.x requests that end on an assistant turn. Official Anthropic: prefilling
// the last assistant message returns HTTP 400 ("This model does not support
// assistant message prefill. The conversation must end with a user message.").
// Mid-conversation assistant turns stay; only a trailing assistant without pending
// tool_use is healed by appending a continuation user turn (Anthropic migration guide).
func SanitizeAnthropicTrailingAssistantPrefill(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !anthropicModelRejectsAssistantPrefill(model) {
		return body
	}
	msgs := gjson.GetBytes(body, "messages")
	if !msgs.IsArray() {
		return body
	}
	arr := msgs.Array()
	if len(arr) == 0 {
		return body
	}
	last := arr[len(arr)-1]
	role := strings.ToLower(strings.TrimSpace(last.Get("role").String()))
	if role != "assistant" {
		return body
	}
	// Incomplete tool loop: trailing assistant with tool_use needs tool_result, not Continue.
	if anthropicAssistantHasPendingToolUse(last) {
		return body
	}
	prevText := strings.TrimSpace(anthropicMessagePlainText(last))
	userText := "Continue from where you left off."
	if prevText != "" {
		const maxPrev = 4000
		if len(prevText) > maxPrev {
			prevText = prevText[len(prevText)-maxPrev:]
		}
		userText = "Your previous response was interrupted and ended with:\n" + prevText +
			"\nContinue from where you left off."
	}
	newMsg := map[string]any{
		"role":    "user",
		"content": userText,
	}
	out, err := sjson.SetBytes(body, fmt.Sprintf("messages.%d", len(arr)), newMsg)
	if err != nil {
		return body
	}
	return out
}

// anthropicModelRejectsAssistantPrefill reports models that 400 on a trailing
// assistant message (Claude 4.6+, Claude 5.x, Fable, Mythos). Claude 4.5 and older
// still accept prefills.
func anthropicModelRejectsAssistantPrefill(modelLower string) bool {
	m := modelLower
	if strings.Contains(m, "fable") || strings.Contains(m, "mythos") {
		return true
	}
	// Explicitly protect Claude 4.5 (still allows prefill).
	if strings.Contains(m, "-4-5") || strings.Contains(m, "4.5") {
		return false
	}
	// Claude 4.6 / 4.7 / 4.8 / 4.9+
	for _, needle := range []string{"-4-6", "-4-7", "-4-8", "-4-9", "4.6", "4.7", "4.8", "4.9"} {
		if strings.Contains(m, needle) {
			return true
		}
	}
	// Claude 5.x family
	if strings.Contains(m, "opus-5") || strings.Contains(m, "sonnet-5") ||
		strings.Contains(m, "haiku-5") || strings.Contains(m, "claude-5") {
		return true
	}
	if strings.Contains(m, "-5-") || strings.HasSuffix(m, "-5") {
		return true
	}
	return false
}

func anthropicAssistantHasPendingToolUse(m gjson.Result) bool {
	content := m.Get("content")
	if !content.IsArray() {
		return false
	}
	for _, part := range content.Array() {
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		switch typ {
		case "tool_use", "server_tool_use", "mcp_tool_use":
			return true
		}
	}
	return false
}

func anthropicMessagePlainText(m gjson.Result) string {
	content := m.Get("content")
	if content.Type == gjson.String {
		return content.String()
	}
	if !content.IsArray() {
		return ""
	}
	var b strings.Builder
	for _, part := range content.Array() {
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		if typ == "" || typ == "text" {
			if t := part.Get("text"); t.Exists() {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(t.String())
			}
		}
	}
	return b.String()
}

// SanitizeAnthropicMessagesToolOrder enforces official Anthropic Messages ordering:
//   - user: tool_result / mcp_tool_result / *_tool_result lead the content array
//   - assistant: client tool_use blocks come last (server_tool_use / web_search_tool_result /
//     mcp / code-execution results must appear before tool_use; text before tool_use)
// Safe for Claude Code native /v1/messages and adapter-translated Anthropic bodies.
func SanitizeAnthropicMessagesToolOrder(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	out := body
	for i, m := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(m.Get("role").String()))
		content := m.Get("content")
		if !content.IsArray() || content.Get("#").Int() < 2 {
			continue
		}
		var fixed []any
		switch role {
		case "user":
			fixed = reorderUserToolResultsFirst(content.Array())
		case "assistant":
			fixed = reorderAssistantToolUseLast(content.Array())
		default:
			continue
		}
		if fixed == nil {
			continue
		}
		path := fmt.Sprintf("messages.%d.content", i)
		var err error
		out, err = sjson.SetBytes(out, path, fixed)
		if err != nil {
			return body
		}
	}
	return EnsureAnthropicToolResultToolsetName(out)
}

// EnsureAnthropicToolResultToolsetName copies toolset_name from prior tool_use blocks
// onto matching tool_result blocks that omitted it. Official Anthropic computer/browser
// toolsets reject member results without the echoing toolset_name (HTTP 400).
func EnsureAnthropicToolResultToolsetName(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return body
	}
	byID := map[string]string{}
	for _, m := range messages.Array() {
		content := m.Get("content")
		if !content.IsArray() {
			continue
		}
		for _, part := range content.Array() {
			if strings.ToLower(strings.TrimSpace(part.Get("type").String())) != "tool_use" {
				continue
			}
			id := strings.TrimSpace(part.Get("id").String())
			ts := strings.TrimSpace(part.Get("toolset_name").String())
			if id != "" && ts != "" {
				byID[id] = ts
			}
		}
	}
	if len(byID) == 0 {
		return body
	}
	out := body
	for i, m := range messages.Array() {
		content := m.Get("content")
		if !content.IsArray() {
			continue
		}
		for j, part := range content.Array() {
			if strings.ToLower(strings.TrimSpace(part.Get("type").String())) != "tool_result" {
				continue
			}
			if strings.TrimSpace(part.Get("toolset_name").String()) != "" {
				continue
			}
			id := strings.TrimSpace(part.Get("tool_use_id").String())
			ts := strings.TrimSpace(byID[id])
			if ts == "" {
				continue
			}
			path := fmt.Sprintf("messages.%d.content.%d.toolset_name", i, j)
			var err error
			out, err = sjson.SetBytes(out, path, ts)
			if err != nil {
				return body
			}
		}
	}
	return out
}

func reorderUserToolResultsFirst(parts []gjson.Result) []any {
	var tools, rest []any
	seenNonToolBeforeTool := false
	seenNonTool := false
	for _, part := range parts {
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		isTool := typ == "tool_result" || typ == "mcp_tool_result" || strings.HasSuffix(typ, "_tool_result")
		if isTool {
			if seenNonTool {
				seenNonToolBeforeTool = true
			}
			tools = append(tools, part.Value())
		} else {
			seenNonTool = true
			rest = append(rest, part.Value())
		}
	}
	if len(tools) == 0 || !seenNonToolBeforeTool {
		return nil
	}
	return append(tools, rest...)
}

func reorderAssistantToolUseLast(parts []gjson.Result) []any {
	// Official: client tool_use must be last so the next user tool_result is adjacent.
	// server_tool_use / web_search_tool_result / mcp / code-execution results stay in prefix.
	var prefix, toolUses []any
	toolUseSeenEarly := false
	seenToolUse := false
	for _, part := range parts {
		typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
		if typ == "tool_use" {
			seenToolUse = true
			toolUses = append(toolUses, part.Value())
			continue
		}
		if seenToolUse {
			toolUseSeenEarly = true
		}
		prefix = append(prefix, part.Value())
	}
	if len(toolUses) == 0 || !toolUseSeenEarly {
		return nil
	}
	return append(prefix, toolUses...)
}

func blockType(b any) string {
	switch t := b.(type) {
	case map[string]any:
		if typ, ok := t["type"].(string); ok {
			return strings.ToLower(strings.TrimSpace(typ))
		}
	}
	return ""
}

// openAIFilePartToAnthropicDocument maps OpenAI Chat Completions file parts to
// Anthropic document blocks (Files API file_id or base64 PDF data).
func openAIFilePartToAnthropicDocument(part gjson.Result) map[string]any {
	fileID := strings.TrimSpace(part.Get("file.file_id").String())
	if fileID == "" {
		fileID = strings.TrimSpace(part.Get("file_id").String())
	}
	if fileID != "" {
		return map[string]any{
			"type": "document",
			"source": map[string]any{
				"type":    "file",
				"file_id": fileID,
			},
		}
	}
	data := strings.TrimSpace(part.Get("file.file_data").String())
	if data == "" {
		data = strings.TrimSpace(part.Get("file_data").String())
	}
	if data == "" {
		return nil
	}
	// data:application/pdf;base64,... or raw base64
	mediaType := "application/pdf"
	payload := data
	if strings.HasPrefix(strings.ToLower(data), "data:") {
		rest := data[len("data:"):]
		semi := strings.Index(rest, ";")
		comma := strings.Index(rest, ",")
		if semi > 0 && comma > semi {
			mediaType = rest[:semi]
			payload = rest[comma+1:]
		} else if comma > 0 {
			payload = rest[comma+1:]
		}
	}
	return map[string]any{
		"type": "document",
		"source": map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       payload,
		},
	}
}

func appendContent(a, b any) any {
	ab := asBlocks(a)
	bb := asBlocks(b)
	return append(ab, bb...)
}

// appendUserContentToolResultsFirst concatenates user content blocks and ensures
// every tool_result leads the turn (official Anthropic Messages adjacency rule).
func appendUserContentToolResultsFirst(a, b any) any {
	ab := asBlocks(a)
	bb := asBlocks(b)
	all := make([]any, 0, len(ab)+len(bb))
	all = append(all, ab...)
	all = append(all, bb...)
	return orderToolResultsFirst(all)
}

// orderToolResultsFirst moves tool_result blocks ahead of text/image/document/etc.
// Official: "In the user message containing tool results, the tool_result blocks must
// come FIRST in the content array. Any text must come AFTER all tool results."
func orderToolResultsFirst(blocks []any) []any {
	if len(blocks) < 2 {
		return blocks
	}
	var tools, rest []any
	for _, b := range blocks {
		if blockType(b) == "tool_result" {
			tools = append(tools, b)
			continue
		}
		rest = append(rest, b)
	}
	if len(tools) == 0 || len(rest) == 0 {
		return blocks
	}
	// Already tool-first?
	if blockType(blocks[0]) == "tool_result" {
		ordered := true
		seenNonTool := false
		for _, b := range blocks {
			if blockType(b) == "tool_result" {
				if seenNonTool {
					ordered = false
					break
				}
			} else {
				seenNonTool = true
			}
		}
		if ordered {
			return blocks
		}
	}
	out := make([]any, 0, len(blocks))
	out = append(out, tools...)
	out = append(out, rest...)
	return out
}

func asBlocks(v any) []any {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return nil
		}
		return []any{map[string]any{"type": "text", "text": t}}
	case []any:
		return t
	case []map[string]any:
		// Mid-conversation system / TextBlockParam[] often arrives as []map[string]any.
		// Never json.Marshal into a text block (same smash/garble class as whole-body Deep).
		out := make([]any, 0, len(t))
		for _, b := range t {
			out = append(out, b)
		}
		return out
	case map[string]any:
		return []any{t}
	default:
		// Unknown structured chrome: refuse Raw/JSON dump into text (garble class).
		return nil
	}
}

func openAIStopToAnthropic(t gjson.Result) any {
	if t.Type == gjson.String {
		return []string{t.String()}
	}
	if t.IsArray() {
		var out []string
		for _, x := range t.Array() {
			out = append(out, x.String())
		}
		return out
	}
	return []string{}
}

// isAnthropicDatedToolType detects official Anthropic server-tool type strings
// (bash_20250124, web_search_20250305, text_editor_20250124, …).
func isAnthropicDatedToolType(typ string) bool {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" || typ == "function" || typ == "custom" {
		return false
	}
	for i := 0; i+9 <= len(typ); i++ {
		if typ[i] != '_' {
			continue
		}
		digits := typ[i+1:]
		if len(digits) < 8 {
			continue
		}
		ok := true
		for _, c := range digits[:8] {
			if c < '0' || c > '9' {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func openAIToolsToAnthropic(tools gjson.Result) ([]any, error) {
	var out []any
	for _, t := range tools.Array() {
		typ := strings.ToLower(strings.TrimSpace(t.Get("type").String()))
		// Anthropic dated server tools on a Chat-shaped body: pass through verbatim
		// (bash_20250124, web_search_20250305, …). Never reject the whole request.
		if isAnthropicDatedToolType(typ) {
			out = append(out, t.Value())
			continue
		}
		// Official Chat Completions custom tools have no Messages tool_use schema equivalent.
		// Skip - do not 400 the entire openai_to_anthropic turn (flexibility for GPT/Cursor).
		if typ == "custom" || (typ == "" && t.Get("custom").Exists()) {
			continue
		}
		// Unknown hosted/Responses tool types (web_search, file_search, …): skip that entry
		// only. Rejecting the whole tools array used to 400 mixed GPT/Cursor→Claude turns
		// even when valid function tools were present (same flexibility class as custom).
		if typ != "" && typ != "function" {
			continue
		}
		name := t.Get("function.name").String()
		if name == "" {
			name = t.Get("name").String()
		}
		if name == "" {
			// Nameless entry after custom/unknown skip - drop, do not 400 the request.
			continue
		}
		desc := t.Get("function.description").String()
		if desc == "" {
			desc = t.Get("description").String()
		}
		schema := t.Get("function.parameters")
		if !schema.Exists() {
			schema = t.Get("parameters")
		}
		if !schema.Exists() {
			// Defense-in-depth: Anthropic-shaped flat tools on Chat Completions door.
			schema = t.Get("input_schema")
		}
		var inputSchema any = map[string]any{"type": "object", "properties": map[string]any{}}
		if schema.Exists() && schema.Type != gjson.Null {
			_ = json.Unmarshal([]byte(schema.Raw), &inputSchema)
		}
		tool := map[string]any{
			"name":         name,
			"description":  desc,
			"input_schema": inputSchema,
		}
		// Official Anthropic: cache_control on tool definitions is part of the
		// tools→system→messages cache prefix. Never drop it in openai_to_anthropic.
		if cc := t.Get("cache_control"); cc.Exists() {
			tool["cache_control"] = cc.Value()
		} else if cc := t.Get("function.cache_control"); cc.Exists() {
			tool["cache_control"] = cc.Value()
		}
		// Official tool extras that affect validation / loading - preserve when present.
		if t.Get("strict").Exists() {
			tool["strict"] = t.Get("strict").Bool()
		} else if t.Get("function.strict").Exists() {
			tool["strict"] = t.Get("function.strict").Bool()
		}
		if t.Get("defer_loading").Exists() {
			tool["defer_loading"] = t.Get("defer_loading").Bool()
		} else if t.Get("function.defer_loading").Exists() {
			tool["defer_loading"] = t.Get("function.defer_loading").Bool()
		}
		// Official Anthropic tool extras: input_examples / allowed_callers /
		// eager_input_streaming. Dropping them silently degrades tool quality /
		// programmatic callers (same flexibility class as defer_loading).
		if ex := t.Get("input_examples"); ex.Exists() {
			tool["input_examples"] = ex.Value()
		} else if ex := t.Get("function.input_examples"); ex.Exists() {
			tool["input_examples"] = ex.Value()
		}
		if ac := t.Get("allowed_callers"); ac.Exists() {
			tool["allowed_callers"] = ac.Value()
		} else if ac := t.Get("function.allowed_callers"); ac.Exists() {
			tool["allowed_callers"] = ac.Value()
		}
		if t.Get("eager_input_streaming").Exists() {
			tool["eager_input_streaming"] = t.Get("eager_input_streaming").Bool()
		} else if t.Get("function.eager_input_streaming").Exists() {
			tool["eager_input_streaming"] = t.Get("function.eager_input_streaming").Bool()
		}
		out = append(out, tool)
	}
	return out, nil
}

// NormalizeAnthropicToolChoiceForThinking downgrades forced tool_choice (any / tool /
// required) to auto when it would 400. Official Anthropic:
//   - Manual extended thinking (type=enabled): only auto|none - forced → 400.
//   - Adaptive thinking: forced tool use is allowed, EXCEPT Claude Opus 5.5 /
//     Sonnet 5.5 / Fable 5.1 / Mythos 5.1 (still 400 on forced).
// Call AFTER NormalizeAnthropicThinkingForModel so enabled→adaptive migrate wins first.
func NormalizeAnthropicToolChoiceForThinking(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	typ := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String()))
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	mustDowngrade := false
	switch typ {
	case "enabled":	
		// Manual extended thinking - always incompatible with forced tools.
		mustDowngrade = true
	case "adaptive", "between_tools":
		mustDowngrade = anthropicAdaptiveRejectsForcedToolChoice(model)
	default:
		// Bare thinking object / unknown type: fail-closed if thinking looks active.
		if anthropicThinkingModeActive(body) && typ != "disabled" {
			// Prefer treating unknown as manual-safe (downgrade) only when not adaptive-family.
			mustDowngrade = !anthropicModelRequiresAdaptiveThinking(model) ||
				anthropicAdaptiveRejectsForcedToolChoice(model)
		}
	}
	if !mustDowngrade {
		return body
	}
	tc := gjson.GetBytes(body, "tool_choice")
	if !tc.Exists() || tc.Type == gjson.Null {
		return body
	}
	forced := false
	switch {
	case tc.Type == gjson.String:
		switch strings.ToLower(strings.TrimSpace(tc.String())) {
		case "required", "any":
			forced = true
		case "auto", "none":
			return body
		}
	case tc.IsObject():
		t := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
		switch t {
		case "any", "tool", "required":
			forced = true
		case "auto", "none":
			return body
		default:
			if strings.TrimSpace(tc.Get("name").String()) != "" ||
				tc.Get("function").Exists() {
				forced = true
			}
		}
	}
	if !forced {
		return body
	}
	choice := map[string]any{"type": "auto"}
	if tc.IsObject() && tc.Get("disable_parallel_tool_use").Exists() {
		choice["disable_parallel_tool_use"] = tc.Get("disable_parallel_tool_use").Bool()
	}
	out, err := sjson.SetBytes(body, "tool_choice", choice)
	if err != nil {
		return body
	}
	return out
}

// anthropicAdaptiveRejectsForcedToolChoice reports models where adaptive thinking
// still 400s on tool_choice any/tool (official Anthropic thinking docs exception list).
func anthropicAdaptiveRejectsForcedToolChoice(modelLower string) bool {
	m := modelLower
	if strings.Contains(m, "fable") || strings.Contains(m, "mythos") {
		return true
	}
	// Opus 5.5 / Sonnet 5.5 (and dated -5-5) - not plain Opus 5 / Sonnet 5.
	if strings.Contains(m, "opus-5-5") || strings.Contains(m, "sonnet-5-5") ||
		strings.Contains(m, "opus-5.5") || strings.Contains(m, "sonnet-5.5") {
		return true
	}
	return false
}

// SanitizeAnthropicTemperatureTopPMutex enforces official Anthropic Claude 4.x rule:
// "`temperature` and `top_p` cannot both be specified for this model. Please use only one."
// Prefer temperature (drop top_p) when both are present. No-op when sampling was already
// fully stripped by SanitizeAnthropicThinkingIncompatibleFields (Claude 4.7+ / thinking).
func SanitizeAnthropicTemperatureTopPMutex(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !anthropicModelRejectsBothTemperatureAndTopP(model) {
		return body
	}
	if !gjson.GetBytes(body, "temperature").Exists() || !gjson.GetBytes(body, "top_p").Exists() {
		return body
	}
	out, err := sjson.DeleteBytes(body, "top_p")
	if err != nil {
		return body
	}
	return out
}

// anthropicModelRejectsBothTemperatureAndTopP reports Claude 4.x models that 400 when
// temperature and top_p are both set (live Anthropic + LiteLLM 2025/2026).
func anthropicModelRejectsBothTemperatureAndTopP(modelLower string) bool {
	m := modelLower
	if !(strings.Contains(m, "claude") || strings.Contains(m, "opus") ||
		strings.Contains(m, "sonnet") || strings.Contains(m, "haiku") ||
		strings.Contains(m, "fable") || strings.Contains(m, "mythos")) {
		return false
	}
	// Claude 3.x still allows both; only Claude 4+ / 5+ / Fable / Mythos.
	if strings.Contains(m, "fable") || strings.Contains(m, "mythos") {
		return true
	}
	if strings.Contains(m, "claude-3") || strings.Contains(m, "opus-3") ||
		strings.Contains(m, "sonnet-3") || strings.Contains(m, "haiku-3") {
		return false
	}
	// Claude 4.x / 5.x families (opus-4, sonnet-4-5, claude-4, claude-5, …).
	if strings.Contains(m, "-4") || strings.Contains(m, "4.") ||
		strings.Contains(m, "-5") || strings.Contains(m, "5.") ||
		strings.Contains(m, "opus-4") || strings.Contains(m, "sonnet-4") ||
		strings.Contains(m, "haiku-4") || strings.Contains(m, "opus-5") ||
		strings.Contains(m, "sonnet-5") || strings.Contains(m, "haiku-5") {
		return true
	}
	return false
}

// SanitizeAnthropicThinkingIncompatibleFields removes temperature / top_p / top_k when
// official Anthropic docs reject them:
//   - any request with thinking active (extended or adaptive), OR
//   - Claude Opus 4.7+ / Opus 5 / Sonnet 5 / Fable / Mythos - reject sampling on EVERY
//     request regardless of thinking (official 400 class).
func SanitizeAnthropicThinkingIncompatibleFields(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !anthropicThinkingModeActive(body) && !anthropicModelRejectsSamplingParams(model) {
		return body
	}
	out := body
	for _, key := range []string{"temperature", "top_p", "top_k"} {
		if !gjson.GetBytes(out, key).Exists() {
			continue
		}
		var err error
		out, err = sjson.DeleteBytes(out, key)
		if err != nil {
			return body
		}
	}
	return out
}

// anthropicModelRejectsSamplingParams reports models that 400 on non-default
// temperature/top_p/top_k on every request (official Anthropic thinking docs).
func anthropicModelRejectsSamplingParams(modelLower string) bool {
	m := modelLower
	if strings.Contains(m, "fable") || strings.Contains(m, "mythos") {
		return true
	}
	// Opus 4.7 / 4.8 / 4.9+ (not 4.5 / 4.6 - 4.6 only rejects alongside thinking).
	if strings.Contains(m, "opus-4-7") || strings.Contains(m, "opus-4-8") ||
		strings.Contains(m, "opus-4-9") || strings.Contains(m, "opus-4.7") ||
		strings.Contains(m, "opus-4.8") || strings.Contains(m, "opus-4.9") {
		return true
	}
	// Claude 5.x family (opus-5 / sonnet-5 / haiku-5 / claude-5) - exclude 4.5.
	if strings.Contains(m, "-4-5") || strings.Contains(m, "4.5") {
		return false
	}
	if strings.Contains(m, "opus-5") || strings.Contains(m, "sonnet-5") ||
		strings.Contains(m, "haiku-5") || strings.Contains(m, "claude-5") {
		return true
	}
	return false
}

// NormalizeAnthropicThinkingForModel migrates thinking.type=enabled (+ budget_tokens)
// to adaptive on models that reject extended thinking (Claude 4.7+, Claude 5.x,
// Fable/Mythos). Official: "thinking.type.enabled is not supported… Use adaptive".
// Older Claude 4.5 models keep enabled+budget_tokens unchanged (adaptive 400s there).
func NormalizeAnthropicThinkingForModel(body []byte) []byte {
	if !gjson.ValidBytes(body) {
		return body
	}
	model := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	if !anthropicModelRequiresAdaptiveThinking(model) {
		return body
	}
	typ := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "thinking.type").String()))
	if typ != "enabled" {
		return body
	}
	out, err := sjson.SetBytes(body, "thinking.type", "adaptive")
	if err != nil {
		return body
	}
	if gjson.GetBytes(out, "thinking.budget_tokens").Exists() {
		out, err = sjson.DeleteBytes(out, "thinking.budget_tokens")
		if err != nil {
			return body
		}
	}
	return out
}

// anthropicModelRequiresAdaptiveThinking reports models that 400 on
// thinking.type=enabled (must use adaptive + output_config.effort).
func anthropicModelRequiresAdaptiveThinking(modelLower string) bool {
	m := modelLower
	if strings.Contains(m, "fable") || strings.Contains(m, "mythos") {
		return true
	}
	// Claude 4.7 / 4.8 / 4.9 family (claude-opus-4-7, dated -4-7-…).
	if strings.Contains(m, "-4-7") || strings.Contains(m, "-4-8") || strings.Contains(m, "-4-9") {
		return true
	}
	if strings.Contains(m, "4.7") || strings.Contains(m, "4.8") || strings.Contains(m, "4.9") {
		return true
	}
	// Claude 5.x family - do NOT match Claude 4.5 (…-4-5-… / …-4-5).
	if strings.Contains(m, "-4-5") || strings.Contains(m, "4.5") {
		return false
	}
	if strings.Contains(m, "opus-5") || strings.Contains(m, "sonnet-5") ||
		strings.Contains(m, "haiku-5") || strings.Contains(m, "claude-5") {
		return true
	}
	if strings.Contains(m, "-5-") || strings.HasSuffix(m, "-5") {
		return true
	}
	return false
}

// anthropicThinkingModeActive reports whether the request enables thinking
// (enabled / adaptive / between_tools / bare thinking object). Explicit disabled → false.
func anthropicThinkingModeActive(body []byte) bool {
	t := gjson.GetBytes(body, "thinking")
	if !t.Exists() || t.Type == gjson.Null {
		return false
	}
	typ := strings.ToLower(strings.TrimSpace(t.Get("type").String()))
	if typ == "disabled" {
		return false
	}
	return true
}

func openAIToolChoiceToAnthropic(tc gjson.Result) any {
	if tc.Type == gjson.String {
		switch strings.ToLower(tc.String()) {
		case "none":
			return map[string]any{"type": "none"}
		case "auto":
			return map[string]any{"type": "auto"}
		case "required":
			return map[string]any{"type": "any"}
		default:
			return map[string]any{"type": "auto"}
		}
	}
	if tc.IsObject() {
		typ := strings.ToLower(strings.TrimSpace(tc.Get("type").String()))
		// Chat custom / allowed_tools have no Anthropic force equivalent → auto.
		// (Anthropic Messages has none|auto|any|tool; not allowed_tools.)
		if typ == "custom" || typ == "allowed_tools" {
			return map[string]any{"type": "auto"}
		}
		// Already Anthropic-shaped tool_choice on a Chat body (passthrough agents).
		if typ == "any" || typ == "auto" || typ == "none" {
			out := map[string]any{"type": typ}
			if tc.Get("disable_parallel_tool_use").Exists() {
				out["disable_parallel_tool_use"] = tc.Get("disable_parallel_tool_use").Bool()
			}
			return out
		}
		if typ == "tool" {
			name := strings.TrimSpace(tc.Get("name").String())
			if name != "" {
				out := map[string]any{"type": "tool", "name": name}
				if tc.Get("disable_parallel_tool_use").Exists() {
					out["disable_parallel_tool_use"] = tc.Get("disable_parallel_tool_use").Bool()
				}
				return out
			}
			return map[string]any{"type": "auto"}
		}
		name := strings.TrimSpace(tc.Get("function.name").String())
		if name == "" {
			name = strings.TrimSpace(tc.Get("name").String())
		}
		if name != "" {
			return map[string]any{"type": "tool", "name": name}
		}
	}
	return map[string]any{"type": "auto"}
}

func anthropicMessagesToOpenAIChat(body []byte, statusCode int) ([]byte, error) {
	if statusCode >= 400 {
		return anthropicErrorToOpenAI(body, statusCode)
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid Anthropic JSON")
	}
	id := gjson.GetBytes(body, "id").String()
	model := gjson.GetBytes(body, "model").String()
	stopReason := gjson.GetBytes(body, "stop_reason").String()
	finish := anthropicStopToFinish(stopReason)

	var text strings.Builder
	var reasoning strings.Builder
	var toolCalls []map[string]any
	var thinkingBlocks []any
	content := gjson.GetBytes(body, "content")
	if content.IsArray() {
		for _, block := range content.Array() {
			switch block.Get("type").String() {
			case "text":
				text.WriteString(block.Get("text").String())
			case "thinking":
				// Preserve full block (incl. signature) for openai_to_anthropic round-trip.
				thinkingBlocks = append(thinkingBlocks, block.Value())
				reasoning.WriteString(block.Get("thinking").String())
			case "redacted_thinking":
				thinkingBlocks = append(thinkingBlocks, block.Value())
				// Opaque blob; expose presence without inventing decoded text.
				if reasoning.Len() > 0 {
					reasoning.WriteString("\n")
				}
				reasoning.WriteString("[redacted_thinking]")
			case "tool_use":
				args, _ := json.Marshal(block.Get("input").Value())
				tc := map[string]any{
					"id":   block.Get("id").String(),
					"type": "function",
					"function": map[string]any{
						"name":      block.Get("name").String(),
						"arguments": string(args),
					},
				}
				// Round-trip Anthropic toolset_name / non-null caller on Chat tool_calls
				// so openai_to_anthropic can rebuild valid computer/browser member calls.
				if ts := strings.TrimSpace(block.Get("toolset_name").String()); ts != "" {
					tc["toolset_name"] = ts
				}
				if c := block.Get("caller"); c.Exists() && c.Type != gjson.Null {
					tc["caller"] = c.Value()
				}
				toolCalls = append(toolCalls, tc)
			}
		}
	}

	msg := map[string]any{
		"role":    "assistant",
		"content": text.String(),
	}
	if reasoning.Len() > 0 {
		msg["reasoning_content"] = reasoning.String()
	}
	if len(thinkingBlocks) > 0 {
		// Official: thinking blocks must be echoed unmodified (signature/data intact).
		msg["thinking_blocks"] = thinkingBlocks
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
		if text.Len() == 0 {
			msg["content"] = nil
		}
	}

	usage := map[string]any{
		"prompt_tokens":     gjson.GetBytes(body, "usage.input_tokens").Int(),
		"completion_tokens": gjson.GetBytes(body, "usage.output_tokens").Int(),
		"total_tokens": gjson.GetBytes(body, "usage.input_tokens").Int() +
			gjson.GetBytes(body, "usage.output_tokens").Int(),
	}

	out := map[string]any{
		"id":      id,
		"object":  "chat.completion",
		"model":   model,
		"choices": []any{map[string]any{"index": 0, "message": msg, "finish_reason": finish}},
		"usage":   usage,
	}
	return json.Marshal(out)
}

func anthropicErrorToOpenAI(body []byte, statusCode int) ([]byte, error) {
	msg := strings.TrimSpace(gjson.GetBytes(body, "error.message").String())
	typ := strings.TrimSpace(gjson.GetBytes(body, "error.type").String())
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if typ == "" {
		typ = "api_error"
	}
	out := map[string]any{
		"error": map[string]any{
			"message": msg,
			"type":    typ,
			"code":    statusCode,
		},
	}
	return json.Marshal(out)
}

func anthropicStopToFinish(stop string) string {
	switch stop {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		if stop == "" {
			return "stop"
		}
		return stop
	}
}

// PipeAnthropicSSEToOpenAI translates Anthropic SSE to OpenAI chat-completion chunks.
func PipeAnthropicSSEToOpenAI(dst http.ResponseWriter, src io.Reader, model string) error {
	flusher, _ := dst.(http.Flusher)
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 2048)
	toolIndex := map[string]int{}
	blockToTool := map[int]int{}
	nextTool := 0
	var thinkingBuf strings.Builder
	for {
		n, err := src.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			for {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					break
				}
				line := string(buf[:idx])
				buf = buf[idx+1:]
				line = strings.TrimRight(line, "\r")
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if payload == "" || payload == "[DONE]" {
					continue
				}
				chunks, done, cerr := AnthropicSSEEventToOpenAIChunks(payload, model, toolIndex, blockToTool, &nextTool, &thinkingBuf)
				if cerr != nil {
					return cerr
				}
				for _, ch := range chunks {
					if _, werr := fmt.Fprintf(dst, "data: %s\n\n", ch); werr != nil {
						return werr
					}
					if flusher != nil {
						flusher.Flush()
					}
				}
				if done {
					if _, werr := io.WriteString(dst, "data: [DONE]\n\n"); werr != nil {
						return werr
					}
					if flusher != nil {
						flusher.Flush()
					}
					return nil
				}
			}
		}
		if err == io.EOF {
			if _, werr := io.WriteString(dst, "data: [DONE]\n\n"); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// AnthropicSSEEventToOpenAIChunks maps one Anthropic SSE JSON payload to OpenAI-compatible
// chat.completion.chunk data lines. Exported for round-trip tests (thinking signatures, tools).
// thinkingBuf accumulates thinking_delta text so signature_delta can emit a complete
// thinking_blocks entry (text+signature). Emitting signature with blanked thinking while
// the original had text causes Anthropic HTTP 400 "blocks must remain as they were".
func AnthropicSSEEventToOpenAIChunks(payload, model string, toolIndex map[string]int, blockToTool map[int]int, nextTool *int, thinkingBuf *strings.Builder) ([]string, bool, error) {
	typ := gjson.Get(payload, "type").String()
	switch typ {
	case "message_start":
		id := gjson.Get(payload, "message.id").String()
		m := gjson.Get(payload, "message.model").String()
		if m != "" {
			model = m
		}
		chunk, err := sjson.Set("{}", "id", id)
		if err != nil {
			return nil, false, err
		}
		chunk, _ = sjson.Set(chunk, "object", "chat.completion.chunk")
		chunk, _ = sjson.Set(chunk, "model", model)
		chunk, _ = sjson.SetRaw(chunk, "choices", `[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]`)
		return []string{chunk}, false, nil
	case "content_block_start":
		block := gjson.Get(payload, "content_block")
		blockIdx := int(gjson.Get(payload, "index").Int())
		blockType := block.Get("type").String()
		switch blockType {
		case "tool_use":
			id := block.Get("id").String()
			name := block.Get("name").String()
			idx := *nextTool
			*nextTool = idx + 1
			toolIndex[id] = idx
			blockToTool[blockIdx] = idx
			delta := fmt.Sprintf(
				`{"index":0,"delta":{"tool_calls":[{"index":%d,"id":%q,"type":"function","function":{"name":%q,"arguments":""}}]},"finish_reason":null}`,
				idx, id, name,
			)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", "["+delta+"]")
			return []string{chunk}, false, nil
		case "thinking":
			// New thinking block - reset accumulator so prior block text cannot leak.
			if thinkingBuf != nil {
				thinkingBuf.Reset()
			}
			return nil, false, nil
		case "redacted_thinking":
			// Official: redacted_thinking arrives as a full block (opaque data); must be
			// echoed unmodified on later openai_to_anthropic turns.
			if thinkingBuf != nil {
				thinkingBuf.Reset()
			}
			data := block.Get("data").String()
			esc, _ := json.Marshal(data)
			raw := fmt.Sprintf(
				`[{"index":0,"delta":{"thinking_blocks":[{"type":"redacted_thinking","data":%s}]},"finish_reason":null}]`,
				esc,
			)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", raw)
			return []string{chunk}, false, nil
		default:
			// text: deltas follow (text_delta).
			return nil, false, nil
		}
	case "content_block_delta":
		delta := gjson.Get(payload, "delta")
		blockIdx := int(gjson.Get(payload, "index").Int())
		switch delta.Get("type").String() {
		case "text_delta":
			text := delta.Get("text").String()
			esc, _ := json.Marshal(text)
			raw := fmt.Sprintf(`[{"index":0,"delta":{"content":%s},"finish_reason":null}]`, esc)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", raw)
			return []string{chunk}, false, nil
		case "thinking_delta":
			text := delta.Get("thinking").String()
			if thinkingBuf != nil {
				thinkingBuf.WriteString(text)
			}
			// Dual-channel: reasoning_content for DeepSeek/OpenAI-compat UIs; full
			// text is also attached on signature_delta thinking_blocks for round-trip.
			esc, _ := json.Marshal(text)
			raw := fmt.Sprintf(`[{"index":0,"delta":{"reasoning_content":%s},"finish_reason":null}]`, esc)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", raw)
			return []string{chunk}, false, nil
		case "signature_delta":
			// Official Anthropic streaming: signature_delta arrives just before
			// content_block_stop. Emit complete thinking+signature so clients that
			// only persist thinking_blocks do not blank text (permanent HTTP 400).
			sig := delta.Get("signature").String()
			thinkText := ""
			if thinkingBuf != nil {
				thinkText = thinkingBuf.String()
				thinkingBuf.Reset()
			}
			thinkEsc, _ := json.Marshal(thinkText)
			sigEsc, _ := json.Marshal(sig)
			raw := fmt.Sprintf(
				`[{"index":0,"delta":{"thinking_blocks":[{"type":"thinking","thinking":%s,"signature":%s}]},"finish_reason":null}]`,
				thinkEsc, sigEsc,
			)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", raw)
			_ = blockIdx
			return []string{chunk}, false, nil
		case "input_json_delta":
			partial := delta.Get("partial_json").String()
			idx, ok := blockToTool[blockIdx]
			if !ok {
				idx = *nextTool - 1
				if idx < 0 {
					idx = 0
				}
			}
			esc, _ := json.Marshal(partial)
			raw := fmt.Sprintf(
				`[{"index":0,"delta":{"tool_calls":[{"index":%d,"function":{"arguments":%s}}]},"finish_reason":null}]`,
				idx, esc,
			)
			chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
			chunk, _ = sjson.Set(chunk, "model", model)
			chunk, _ = sjson.SetRaw(chunk, "choices", raw)
			return []string{chunk}, false, nil
		default:
			return nil, false, nil
		}
	case "message_delta":
		stop := gjson.Get(payload, "delta.stop_reason").String()
		finish := anthropicStopToFinish(stop)
		esc, _ := json.Marshal(finish)
		raw := fmt.Sprintf(`[{"index":0,"delta":{},"finish_reason":%s}]`, esc)
		chunk, _ := sjson.Set("{}", "object", "chat.completion.chunk")
		chunk, _ = sjson.Set(chunk, "model", model)
		chunk, _ = sjson.SetRaw(chunk, "choices", raw)
		return []string{chunk}, false, nil
	case "message_stop":
		return nil, true, nil
	case "error":
		msg := gjson.Get(payload, "error.message").String()
		return nil, false, fmt.Errorf("%s", msg)
	default:
		return nil, false, nil
	}
}

// openAIImageURLToAnthropicSource maps OpenAI image_url.url to Anthropic image source
// (url or base64), matching Anthropic Vision docs.
func openAIImageURLToAnthropicSource(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty image_url")
	}
	if strings.HasPrefix(raw, "data:") {
		// data:[<mediatype>][;base64],<data>
		rest := strings.TrimPrefix(raw, "data:")
		comma := strings.IndexByte(rest, ',')
		if comma < 0 {
			return nil, fmt.Errorf("invalid data image_url")
		}
		meta := rest[:comma]
		data := rest[comma+1:]
		if data == "" {
			return nil, fmt.Errorf("empty data image_url payload")
		}
		mediaType := "image/png"
		parts := strings.Split(meta, ";")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			mediaType = strings.TrimSpace(parts[0])
		}
		hasB64 := false
		for _, p := range parts[1:] {
			if strings.EqualFold(strings.TrimSpace(p), "base64") {
				hasB64 = true
				break
			}
		}
		if !hasB64 {
			return nil, fmt.Errorf("data image_url must be base64")
		}
		return map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       data,
		}, nil
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return map[string]any{"type": "url", "url": raw}, nil
	}
	return nil, fmt.Errorf("unsupported image_url scheme")
}
