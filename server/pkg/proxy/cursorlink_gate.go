package proxy

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// withCursorLinkGate enforces /cl/<token>/… for public Cursor Link traffic while
// leaving plain /v1 (local on-device clients) and /health /dashboard unchanged.
//
// When a Cloudflare edge header is present on a bare /v1 path, reject it so the
// trycloudflare hostname cannot bypass the token (attackers must use /cl/<token>/v1).
//
// Token resolution is live from ~/.trim/cursor-link.json when Enabled so
// trim cursor-link enable / rotate apply without restarting Trim. Startup token
// from trim start is the fallback when the file is missing.
func withCursorLinkGate(next http.Handler, startupToken string) http.Handler {
	startupToken = strings.TrimSpace(startupToken)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := resolveCursorLinkToken(startupToken)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		prefix := "/cl/" + token
		path := r.URL.Path
		if strings.HasPrefix(path, "/cl/") {
			rest, ok := strings.CutPrefix(path, prefix)
			if !ok {
				http.Error(w, `{"error":"cursor_link_unauthorized"}`, http.StatusUnauthorized)
				return
			}
			if rest == "" {
				rest = "/"
			}
			if !strings.HasPrefix(rest, "/") {
				rest = "/" + rest
			}
			r2 := r.Clone(r.Context())
			r2.URL.Path = rest
			next.ServeHTTP(w, r2)
			return
		}
		// Public tunnel hitting bare /v1 without token → forbid.
		if looksLikeCloudflareEdge(r) && strings.HasPrefix(path, "/v1/") && path != "/v1/control/shutdown" {
			http.Error(w, `{"error":"cursor_link_token_required","hint":"use trim cursor-link url"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func looksLikeCloudflareEdge(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get("Cf-Ray")) != "" {
		return true
	}
	if strings.TrimSpace(r.Header.Get("CF-Connecting-IP")) != "" {
		return true
	}
	if strings.Contains(strings.ToLower(r.Header.Get("CDN-Loop")), "cloudflare") {
		return true
	}
	return false
}

type cursorLinkFileState struct {
	Enabled     bool   `json:"enabled"`
	AccessToken string `json:"access_token"`
}

// cursorLinkStatePathOverride is for unit tests only.
var cursorLinkStatePathOverride string

var (
	cursorLinkTokenCacheMu sync.Mutex
	cursorLinkTokenCache   struct {
		token   string
		enabled bool
		ok      bool
		at      time.Time
	}
)

func cursorLinkStateFilePath() string {
	if p := strings.TrimSpace(cursorLinkStatePathOverride); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".trim", "cursor-link.json")
}

func resolveCursorLinkToken(startupToken string) string {
	liveTok, enabled, ok := loadLiveCursorLinkToken()
	if ok {
		if !enabled {
			return ""
		}
		if liveTok != "" {
			return liveTok
		}
	}
	return strings.TrimSpace(startupToken)
}

func loadLiveCursorLinkToken() (token string, enabled bool, ok bool) {
	path := cursorLinkStateFilePath()
	if path == "" {
		return "", false, false
	}

	cursorLinkTokenCacheMu.Lock()
	defer cursorLinkTokenCacheMu.Unlock()
	if time.Since(cursorLinkTokenCache.at) < 2*time.Second && cursorLinkTokenCache.ok {
		return cursorLinkTokenCache.token, cursorLinkTokenCache.enabled, true
	}

	b, err := os.ReadFile(path)
	if err != nil {
		cursorLinkTokenCache.ok = false
		cursorLinkTokenCache.at = time.Now()
		return "", false, false
	}
	var st cursorLinkFileState
	if err := json.Unmarshal(b, &st); err != nil {
		cursorLinkTokenCache.ok = false
		cursorLinkTokenCache.at = time.Now()
		return "", false, false
	}
	tok := strings.TrimSpace(st.AccessToken)
	cursorLinkTokenCache.token = tok
	cursorLinkTokenCache.enabled = st.Enabled
	cursorLinkTokenCache.ok = true
	cursorLinkTokenCache.at = time.Now()
	return tok, st.Enabled, true
}
