package proxy

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// toolResultClearKey identifies one clearable tool output payload.
// partIdx is -1 for whole-message role=tool / role=function content.
type toolResultClearKey struct {
	msgIdx  int
	partIdx int
}

// planToolResultClears returns older tool_result / role=tool keys to clear.
// Keeps the last ToolResultClearKeepRecent payloads. Empty when dials/chrome unset,
// prompt-cache requests, or nothing eligible. Never includes the trailing user turn.
func (s *Server) planToolResultClears(arr []gjson.Result, requestPromptCache bool) map[toolResultClearKey]bool {
	if requestPromptCache {
		return nil
	}
	keep := s.opts.ToolResultClearKeepRecent
	if keep <= 0 || strings.TrimSpace(s.opts.ToolResultClearedFmt) == "" {
		return nil
	}
	keys := collectClearableToolResultKeys(arr)
	if len(keys) <= keep {
		return nil
	}
	toClear := keys[:len(keys)-keep]
	out := make(map[toolResultClearKey]bool, len(toClear))
	for _, k := range toClear {
		out[k] = true
	}
	return out
}

func formatToolResultCleared(fmtStub string, nbytes int) string {
	fmtStub = strings.TrimSpace(fmtStub)
	if strings.Contains(fmtStub, "%d") || strings.Contains(fmtStub, "%v") {
		return fmt.Sprintf(fmtStub, nbytes)
	}
	return fmtStub
}

func collectClearableToolResultKeys(arr []gjson.Result) []toolResultClearKey {
	if len(arr) == 0 {
		return nil
	}
	last := len(arr) - 1
	var keys []toolResultClearKey
	for i, msg := range arr {
		role := strings.ToLower(strings.TrimSpace(msg.Get("role").String()))
		// Current agent step: trailing user turn may carry [tool_result, text]; keep verbatim.
		if i == last && role == "user" {
			continue
		}
		switch role {
		case "tool", "function":
			keys = append(keys, toolResultClearKey{msgIdx: i, partIdx: -1})
			continue
		}
		content := msg.Get("content")
		if !content.IsArray() {
			continue
		}
		for j, part := range content.Array() {
			typ := strings.ToLower(strings.TrimSpace(part.Get("type").String()))
			if !isToolResultOutputPartType(typ) {
				continue
			}
			keys = append(keys, toolResultClearKey{msgIdx: i, partIdx: j})
		}
	}
	return keys
}

func isToolResultOutputPartType(typ string) bool {
	switch typ {
	case "tool_result", "web_search_tool_result", "mcp_tool_result",
		"function_call_output", "custom_tool_call_output":
		return true
	}
	return strings.HasSuffix(typ, "_tool_result")
}

func toolResultPartRaw(part gjson.Result) string {
	c := part.Get("content")
	if !c.Exists() {
		return part.Raw
	}
	if c.Type == gjson.String {
		return c.String()
	}
	return c.Raw
}
