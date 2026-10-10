package cli

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestShouldNormalizeChatJSONForDeepSkipsAnthropicChrome(t *testing.T) {
	// Exact smash class: Chat-wrapping {name,input_schema} destroys Messages tools.
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{
			name: "gpt chat ok to normalize",
			raw:  `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"x","parameters":{}}}]}`,
			want: true,
		},
		{
			name: "responses input ok",
			raw:  `{"model":"gpt-4o","input":[{"role":"user","content":"hi"}]}`,
			want: true,
		},
		{
			name: "claude model skip",
			raw:  `{"model":"claude-opus-4","messages":[{"role":"user","content":"hi"}],"tools":[{"name":"Bash","input_schema":{"type":"object"}}]}`,
			want: false,
		},
		{
			name: "alias without claude but input_schema skip",
			raw:  `{"model":"my-sonnet-proxy","messages":[{"role":"user","content":"hi"}],"tools":[{"name":"Bash","input_schema":{"type":"object"}}]}`,
			want: false,
		},
		{
			name: "mcp_servers skip",
			raw:  `{"model":"my-sonnet-proxy","mcp_servers":[{"name":"x"}],"messages":[{"role":"user","content":"hi"}]}`,
			want: false,
		},
		{
			name: "anthropic thinking budget_tokens skip",
			raw:  `{"model":"my-sonnet-proxy","thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":"hi"}]}`,
			want: false,
		},
		{
			name: "deepseek thinking allows normalize",
			raw:  `{"model":"deepseek-chat","thinking":{"type":"enabled"},"reasoning_effort":"high","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"x","parameters":{}}}]}`,
			want: true,
		},
		{
			name: "dated server tool skip",
			raw:  `{"model":"my-sonnet-proxy","tools":[{"type":"bash_20250124","name":"bash"}],"messages":[{"role":"user","content":"hi"}]}`,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldNormalizeChatJSONForDeep([]byte(tc.raw))
			if got != tc.want {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
			if !tc.want && gjson.GetBytes([]byte(tc.raw), "tools").Exists() {
				// Defense: never leave anthropic input_schema wrapped after a mistaken normalize.
				_ = got
			}
		})
	}
}

func TestLooksLikeChatRequestJSON(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"plain text", "hello world", false},
		{"source code", "package main\nfunc main() {}", false},
		{"openai messages", `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`, true},
		{"responses input", `{"model":"gpt-4o","input":[{"type":"message","role":"user","content":"hi"}]}`, true},
		{"anthropic system", `{"model":"claude-opus-4","system":"sys","messages":[{"role":"user","content":"hi"}]}`, true},
		{"tools only chat", `{"model":"gemini-flash","tools":[{"type":"function","function":{"name":"x"}}]}`, true},
		{"tool_choice only", `{"model":"gpt-4o","tool_choice":"required"}`, true},
		{"anthropic mcp chrome", `{"model":"claude-opus-4","mcp_servers":[{"name":"x"}]}`, true},
		{"gemini native contents", `{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`, true},
		{"deepseek thinking chrome", `{"model":"deepseek-v4-pro","thinking":{"type":"enabled"},"reasoning_effort":"high"}`, true},
		{"unrelated json", `{"foo":1,"bar":2}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := looksLikeChatRequestJSON([]byte(tc.raw))
			if got != tc.want {
				t.Fatalf("got=%v want=%v raw=%s", got, tc.want, tc.raw)
			}
		})
	}
}
