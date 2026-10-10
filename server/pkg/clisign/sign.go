package clisign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Canonical payload for Trim CLI request signatures.
// Format: timestamp\nMETHOD\npath\nhex(sha256(body))
func Canonical(ts, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	return strings.Join([]string{
		strings.TrimSpace(ts),
		strings.ToUpper(strings.TrimSpace(method)),
		normalizePath(path),
		hex.EncodeToString(sum[:]),
	}, "\n")
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// Sign returns hex HMAC-SHA256 of the canonical payload.
func Sign(secret, ts, method, path string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(Canonical(ts, method, path, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify reports whether signature matches (constant-time).
func Verify(secret, ts, method, path string, body []byte, signature string) bool {
	if secret == "" || signature == "" || ts == "" {
		return false
	}
	expected := Sign(secret, ts, method, path, body)
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}

// HeaderName is the HTTP header carrying the CLI HMAC signature.
const HeaderName = "X-Trim-Signature"

func FormatError(detail string) string {
	return fmt.Sprintf("CLI signature invalid: %s", detail)
}
