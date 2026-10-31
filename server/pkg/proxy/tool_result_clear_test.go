package proxy

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestToolResultClearKeepsRecentOpenAIRoleTool(t *testing.T) {
	s := NewServer(Options{
		CompressionMode:           "balanced",
		ToolResultClearKeepRecent: 1,
		ToolResultClearedFmt:      "[trim] prior tool output cleared (%d bytes; re-fetch if needed)",
	})
	oldPayload := strings.Repeat("OLD_TOOL_BYTES_", 40)
	newPayload := "NEW_TOOL_BYTES_BBBBBBBB"
	in := []byte(`{"model":"gpt-4o","messages":[
		{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"read","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"c1","content":"` + oldPayload + `"},
		{"role":"assistant","content":null,"tool_calls":[{"id":"c2","type":"function","function":{"name":"read","arguments":"{}"}}]},
		{"role":"tool","tool_call_id":"c2","content":"` + newPayload + `"},
		{"role":"user","content":"what is 2+2?"}
	]}`)
	out, before, after := s.optimizePayload(in)
	if before <= after {
		t.Fatalf("expected savings from tool clear before=%d after=%d", before, after)
	}
	msgs := gjson.GetBytes(out, "messages").Array()
	if !strings.Contains(msgs[1].Get("content").String(), "prior tool output cleared") {
		t.Fatalf("old tool must clear: %s", msgs[1].Raw)
	}
	if msgs[3].Get("content").String() != newPayload {
		t.Fatalf("recent tool must stay: %s", msgs[3].Raw)
	}
	if msgs[1].Get("tool_call_id").String() != "c1" {
		t.Fatalf("tool_call_id must stay: %s", msgs[1].Raw)
	}
}

func TestToolResultClearKeepsTrailingUserToolResult(t *testing.T) {
	s := NewServer(Options{
		CompressionMode:           "balanced",
		ToolResultClearKeepRecent: 1,
		ToolResultClearedFmt:      "[trim] prior tool output cleared (%d bytes; re-fetch if needed)",
	})
	// Two older tool_results + trailing current: keep_recent=1 clears the oldest only.
	in := []byte(`{"model":"claude-opus-4","messages":[
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"OLD_ANTHROPIC_TOOL_A"},{"type":"text","text":"earlier a"}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"2","content":"OLD_ANTHROPIC_TOOL_B"},{"type":"text","text":"earlier b"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"3","name":"Read","input":{}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"3","content":"CURRENT_KEEP"},{"type":"text","text":"FILLER A\nFILLER B\nwhat next?"}]}
	]}`)
	out, _, _ := s.optimizePayload(in)
	msgs := gjson.GetBytes(out, "messages").Array()
	firstTR := msgs[0].Get("content.0.content").String()
	if !strings.Contains(firstTR, "prior tool output cleared") {
		t.Fatalf("oldest anthropic tool_result must clear: %s", msgs[0].Raw)
	}
	secondTR := msgs[1].Get("content.0.content").String()
	if secondTR != "OLD_ANTHROPIC_TOOL_B" {
		t.Fatalf("kept-recent older tool_result must stay: %s", msgs[1].Raw)
	}
	lastTR := msgs[3].Get("content.0.content").String()
	if lastTR != "CURRENT_KEEP" {
		t.Fatalf("trailing user tool_result must stay: %s", msgs[3].Raw)
	}
}

func TestToolResultClearOffByDefault(t *testing.T) {
	s := NewServer(Options{CompressionMode: "balanced", ToolResultClearKeepRecent: 1})
	// Empty fmt = fail-closed, no invent stub.
	in := []byte(`{"model":"gpt-4o","messages":[
		{"role":"tool","tool_call_id":"c1","content":"OLD_TOOL_BYTES_AAAAAAAA"},
		{"role":"tool","tool_call_id":"c2","content":"NEW_TOOL_BYTES_BBBBBBBB"},
		{"role":"user","content":"hi"}
	]}`)
	out, _, _ := s.optimizePayload(in)
	if !strings.Contains(string(out), "OLD_TOOL_BYTES_AAAAAAAA") {
		t.Fatalf("without chrome fmt must not clear: %s", string(out))
	}
}

func TestHistoryCompactRequiresChromeStub(t *testing.T) {
	// Fail-closed: HistoryKeepTurns alone must not invent a stub.
	s := NewServer(Options{CompressionMode: "balanced", HistoryKeepTurns: 1})
	in := []byte(`{"model":"gpt-4o","messages":[
		{"role":"user","content":"old turn one with filler text"},
		{"role":"assistant","content":"old assistant reply"},
		{"role":"user","content":"latest question please"}
	]}`)
	out, _, _ := s.optimizePayload(in)
	if strings.Contains(string(out), "earlier conversation turns compacted") {
		t.Fatalf("empty HistoryCompactedStub must not stub history: %s", string(out))
	}
}

func TestToolResultClearDisabledWhenKeepZero(t *testing.T) {
	s := NewServer(Options{
		CompressionMode:           "balanced",
		ToolResultClearKeepRecent: 0,
		ToolResultClearedFmt:      "[trim] prior tool output cleared (%d bytes; re-fetch if needed)",
	})
	in := []byte(`{"model":"gpt-4o","messages":[
		{"role":"tool","tool_call_id":"c1","content":"OLD_TOOL_BYTES_AAAAAAAA"},
		{"role":"user","content":"hi"}
	]}`)
	out, _, _ := s.optimizePayload(in)
	if !strings.Contains(string(out), "OLD_TOOL_BYTES_AAAAAAAA") {
		t.Fatalf("keep_recent=0 must not clear: %s", string(out))
	}
}
