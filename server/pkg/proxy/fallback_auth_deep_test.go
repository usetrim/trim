package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/sjson"
	"github.com/usetrim/trim/server/pkg/localchrome"
	"github.com/usetrim/trim/server/pkg/proxy"
)

// Regression: native Anthropic door (no adapter) + FallbackUncompressed must NOT
// erase Deep/Fast savings when upstream returns 401. Old bug: any >=400 triggered
// raw retry and after=before, so Claude looked like 0% Saved while Gemini (adapter)
// kept savings.
func TestProxyAuthFailureKeepsDeepSavingsNoAdapter(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "DEEP_OK") {
			t.Errorf("expected Deep-shrunk body forwarded on 401 path, got len=%d", len(body))
		}
		if strings.Contains(string(body), "FILLER noise for fallback") {
			t.Errorf("raw uncompressed filler must not be re-sent on 401 fallback")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"type":"authentication_error","message":"bad key"}}`))
	}))
	defer up.Close()

	deep := func(optimized []byte, _ string) (proxy.DeepResult, error) {
		out, err := sjson.SetBytes(optimized, "messages.0.content.1.text", "DEEP_OK what is 2+2?")
		if err != nil {
			return proxy.DeepResult{}, err
		}
		return proxy.DeepResult{Body: out, Origin: 2000, Compressed: 400, Status: proxy.DeepStatusApplied}, nil
	}

	filler := strings.Repeat("FILLER noise for fallback auth regression. ", 80)
	srv := proxy.NewServer(proxy.Options{
		UpstreamAnthropic:    up.URL,
		DoorAnthropic:        "anthropic",
		CompressionMode:      "balanced",
		FallbackUncompressed: true,
		DeepOptimize:         deep,
		UpstreamHTTPTimeout:  5 * time.Second,
		Chrome:               localchrome.Chrome{ErrBodyRead: "body"},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reqBody := `{
		"model":"claude-opus-4",
		"max_tokens":32,
		"system":"# Environment\n` + strings.Repeat("agent chrome ", 40) + `",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"<system-reminder>\nkeep\n</system-reminder>\n"},
			{"type":"text","text":"` + filler + `\nwhat is 2+2?"}
		]}]
	}`
	res, err := http.Post(ts.URL+"/v1/messages", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, b)
	}
	before, err1 := strconv.Atoi(res.Header.Get("X-Trim-Tokens-Before"))
	after, err2 := strconv.Atoi(res.Header.Get("X-Trim-Tokens-After"))
	if err1 != nil || err2 != nil {
		t.Fatalf("token headers before=%q after=%q", res.Header.Get("X-Trim-Tokens-Before"), res.Header.Get("X-Trim-Tokens-After"))
	}
	if after >= before {
		t.Fatalf("401 must keep Deep savings, got before=%d after=%d (fallback wrongly reset?)", before, after)
	}
	if res.Header.Get("X-Trim-Door") != "anthropic" {
		t.Fatalf("door=%q", res.Header.Get("X-Trim-Door"))
	}
}
