package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVerifySupabaseAccessToken_ES256(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-kid"
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": kid, "typ": "JWT"})
	now := time.Now().Unix()
	payload, _ := json.Marshal(map[string]any{
		"sub": "user-1",
		"exp": now + 3600,
		"iss": "https://example.supabase.co/auth/v1",
		"aud": "authenticated",
		"app_metadata": map[string]any{
			"provider": "google",
		},
	})
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := h + "." + p
	sum := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, priv, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	sig := append(padLeft32(r.Bytes()), padLeft32(s.Bytes())...)
	token := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	supabaseJWKSMu.Lock()
	supabaseJWKSCache = &supabaseJWKS{Keys: []supabaseJWK{{
		Kty: "EC",
		Kid: kid,
		Alg: "ES256",
		Crv: "P-256",
		X:   base64.RawURLEncoding.EncodeToString(padLeft32(priv.PublicKey.X.Bytes())),
		Y:   base64.RawURLEncoding.EncodeToString(padLeft32(priv.PublicKey.Y.Bytes())),
	}}}
	supabaseJWKSAt = time.Now()
	supabaseJWKSURL = "https://example.supabase.co/auth/v1/.well-known/jwks.json"
	supabaseJWKSMu.Unlock()
	t.Cleanup(invalidateSupabaseJWKS)

	sub, provider, err := verifySupabaseAccessToken(token, "", "https://example.supabase.co")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if sub != "user-1" || provider != "google" {
		t.Fatalf("got sub=%q provider=%q", sub, provider)
	}
}

func TestFetchLiveSupabaseJWKS(t *testing.T) {
	// Optional integration test only. Never hardcode a project URL; CI has no live Supabase.
	url := strings.TrimSpace(os.Getenv("SUPABASE_URL"))
	if url == "" {
		t.Skip("SUPABASE_URL not set; skipping live JWKS integration test")
	}
	invalidateSupabaseJWKS()
	jwks, err := fetchSupabaseJWKS(url)
	if err != nil {
		t.Fatalf("live jwks: %v", err)
	}
	if len(jwks.Keys) == 0 {
		t.Fatal("empty jwks")
	}
	foundES256 := false
	for _, k := range jwks.Keys {
		if strings.EqualFold(k.Alg, "ES256") || strings.EqualFold(k.Kty, "EC") {
			foundES256 = true
			if strings.TrimSpace(k.Kid) == "" {
				t.Fatal("es256 key missing kid")
			}
		}
	}
	if !foundES256 {
		t.Fatal("expected at least one EC/ES256 key in live JWKS")
	}
}

func TestVerifySupabaseAccessToken_HS256(t *testing.T) {
	secret := "test-secret"
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	now := time.Now().Unix()
	payload, _ := json.Marshal(map[string]any{
		"sub": "user-2",
		"exp": now + 3600,
		"app_metadata": map[string]any{
			"provider": "github",
		},
	})
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := h + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	token := signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	sub, provider, err := verifySupabaseAccessToken(token, secret, "")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if sub != "user-2" || provider != "github" {
		t.Fatalf("got sub=%q provider=%q", sub, provider)
	}
}

func padLeft32(b []byte) []byte {
	if len(b) > 32 {
		return b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// Keep big referenced for clarity around ECDSA coords in future fixtures.
var _ = big.NewInt
