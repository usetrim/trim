package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Gate tests share package-level path override + token cache; run one at a time.
var cursorLinkGateTestMu sync.Mutex

func withIsolatedCursorLinkState(t *testing.T) {
	t.Helper()
	cursorLinkGateTestMu.Lock()
	dir := t.TempDir()
	prev := cursorLinkStatePathOverride
	cursorLinkStatePathOverride = filepath.Join(dir, "cursor-link.json")
	cursorLinkTokenCacheMu.Lock()
	cursorLinkTokenCache.at = time.Time{}
	cursorLinkTokenCache.ok = false
	cursorLinkTokenCache.token = ""
	cursorLinkTokenCache.enabled = false
	cursorLinkTokenCacheMu.Unlock()
	t.Cleanup(func() {
		cursorLinkStatePathOverride = prev
		cursorLinkTokenCacheMu.Lock()
		cursorLinkTokenCache.ok = false
		cursorLinkTokenCache.token = ""
		cursorLinkTokenCache.enabled = false
		cursorLinkTokenCache.at = time.Time{}
		cursorLinkTokenCacheMu.Unlock()
		cursorLinkGateTestMu.Unlock()
	})
}

func writeCursorLinkStateFile(t *testing.T, enabled bool, token string) {
	t.Helper()
	path := cursorLinkStateFilePath()
	body := []byte(`{"enabled":false}`)
	if enabled {
		body = []byte(`{"enabled":true,"access_token":"` + token + `"}`)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	cursorLinkTokenCacheMu.Lock()
	cursorLinkTokenCache.ok = false
	cursorLinkTokenCache.at = time.Time{}
	cursorLinkTokenCacheMu.Unlock()
}

func TestCursorLinkGate_TokenPath(t *testing.T) {
	withIsolatedCursorLinkState(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	})
	h := withCursorLinkGate(inner, "secrettok")
	req := httptest.NewRequest(http.MethodGet, "/cl/secrettok/v1/models", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code %d", rr.Code)
	}
	body, _ := io.ReadAll(rr.Body)
	if string(body) != "/v1/models" {
		t.Fatalf("path rewrite got %q", body)
	}
}

func TestCursorLinkGate_BadToken(t *testing.T) {
	withIsolatedCursorLinkState(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withCursorLinkGate(inner, "secrettok")
	req := httptest.NewRequest(http.MethodGet, "/cl/wrong/v1/models", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rr.Code)
	}
}

func TestCursorLinkGate_CloudflareBareV1Forbidden(t *testing.T) {
	withIsolatedCursorLinkState(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := withCursorLinkGate(inner, "secrettok")
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Cf-Ray", "abc")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "cursor_link_token_required") {
		t.Fatalf("body %s", rr.Body.String())
	}
}

func TestCursorLinkGate_LocalV1StillWorks(t *testing.T) {
	withIsolatedCursorLinkState(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("local"))
	})
	h := withCursorLinkGate(inner, "secrettok")
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "local" {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
}

func TestCursorLinkGate_LiveFileTokenWithoutRestart(t *testing.T) {
	withIsolatedCursorLinkState(t)
	writeCursorLinkStateFile(t, true, "liveTok123")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	})
	// Startup token empty (enable-after-start); live file must win.
	h := withCursorLinkGate(inner, "")
	req := httptest.NewRequest(http.MethodGet, "/cl/liveTok123/v1/models", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "/v1/models" {
		t.Fatalf("code %d body %s", rr.Code, rr.Body.String())
	}
	req2 := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req2.Header.Set("Cf-Ray", "xyz")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("bare CF /v1 code %d", rr2.Code)
	}
}

func TestCursorLinkGate_DisabledClearsGate(t *testing.T) {
	withIsolatedCursorLinkState(t)
	writeCursorLinkStateFile(t, false, "")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("open"))
	})
	h := withCursorLinkGate(inner, "startupTok")
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Cf-Ray", "abc")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Body.String() != "open" {
		t.Fatalf("disabled should not gate CF /v1: code %d body %s", rr.Code, rr.Body.String())
	}
}
