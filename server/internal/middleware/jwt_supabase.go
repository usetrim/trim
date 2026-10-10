package middleware

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Supabase Auth JWTs: legacy projects sign HS256 with the JWT secret; current
// projects sign ES256 and publish keys at /auth/v1/.well-known/jwks.json.
// Industry practice (Auth0 / Supabase / Clerk): verify via JWKS by kid+alg,
// keep HS256 as a fallback while operators still have a shared secret.

type supabaseJWKS struct {
	Keys []supabaseJWK `json:"keys"`
}

type supabaseJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Use string `json:"use"`
}

var (
	supabaseJWKSMu    sync.Mutex
	supabaseJWKSCache *supabaseJWKS
	supabaseJWKSAt    time.Time
	supabaseJWKSURL   string
)

type supabaseClaims struct {
	Sub         string `json:"sub"`
	Exp         int64  `json:"exp"`
	Iss         string `json:"iss"`
	Aud         any    `json:"aud"`
	AppMetadata struct {
		Provider  string   `json:"provider"`
		Providers []string `json:"providers"`
	} `json:"app_metadata"`
}

// verifySupabaseAccessToken validates a Supabase user access token (HS256 or ES256).
func verifySupabaseAccessToken(token, hsSecret, supabaseURL string) (sub string, provider string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("malformed jwt")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", err
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", "", err
	}
	alg := strings.ToUpper(strings.TrimSpace(header.Alg))
	switch alg {
	case "ES256":
		if err := verifyES256JWT(parts, strings.TrimSpace(header.Kid), supabaseURL); err != nil {
			return "", "", err
		}
	case "HS256":
		if err := verifyHS256Sig(parts, hsSecret); err != nil {
			return "", "", err
		}
	default:
		return "", "", fmt.Errorf("unsupported alg %s", alg)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", err
	}
	var claims supabaseClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", "", err
	}
	if claims.Sub == "" {
		return "", "", fmt.Errorf("missing sub")
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return "", "", fmt.Errorf("expired")
	}
	if err := validateSupabaseIssuerAudience(claims, supabaseURL); err != nil {
		return "", "", err
	}
	provider = strings.ToLower(strings.TrimSpace(claims.AppMetadata.Provider))
	if provider == "" && len(claims.AppMetadata.Providers) > 0 {
		provider = strings.ToLower(strings.TrimSpace(claims.AppMetadata.Providers[0]))
	}
	return claims.Sub, provider, nil
}

func validateSupabaseIssuerAudience(claims supabaseClaims, supabaseURL string) error {
	base := strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	if base == "" {
		// Fail closed on iss when URL is configured; without URL skip iss (legacy HS256-only deploys).
		return nil
	}
	wantIss := base + "/auth/v1"
	if !strings.EqualFold(strings.TrimSpace(claims.Iss), wantIss) {
		return fmt.Errorf("iss mismatch")
	}
	if !audContains(claims.Aud, "authenticated") {
		return fmt.Errorf("aud mismatch")
	}
	return nil
}

func audContains(raw any, want string) bool {
	want = strings.TrimSpace(want)
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v) == want
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) == want {
				return true
			}
		}
	}
	return false
}

func verifyHS256Sig(parts []string, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("jwt secret missing")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	if !hmac.Equal(sig, expected) {
		return errors.New("bad signature")
	}
	return nil
}

func verifyES256JWT(parts []string, kid, supabaseURL string) error {
	if kid == "" {
		return errors.New("missing kid")
	}
	jwks, err := fetchSupabaseJWKS(supabaseURL)
	if err != nil {
		return err
	}
	var pub *ecdsa.PublicKey
	for _, k := range jwks.Keys {
		if k.Kid != kid {
			continue
		}
		if !strings.EqualFold(k.Kty, "EC") {
			continue
		}
		if k.Alg != "" && !strings.EqualFold(k.Alg, "ES256") {
			continue
		}
		p, err := jwkToECDSA(k)
		if err != nil {
			return err
		}
		pub = p
		break
	}
	if pub == nil {
		// Key rotation: bust cache once and retry.
		invalidateSupabaseJWKS()
		jwks, err = fetchSupabaseJWKS(supabaseURL)
		if err != nil {
			return err
		}
		for _, k := range jwks.Keys {
			if k.Kid != kid || !strings.EqualFold(k.Kty, "EC") {
				continue
			}
			p, err := jwkToECDSA(k)
			if err != nil {
				return err
			}
			pub = p
			break
		}
	}
	if pub == nil {
		return errors.New("kid not found")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	// JWS ECDSA P-256: R||S, each 32 bytes (RFC 7518 §3.4).
	if len(sig) != 64 {
		return errors.New("bad es256 sig length")
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(pub, sum[:], r, s) {
		return errors.New("bad signature")
	}
	return nil
}

func jwkToECDSA(k supabaseJWK) (*ecdsa.PublicKey, error) {
	if k.Crv != "" && !strings.EqualFold(k.Crv, "P-256") {
		return nil, fmt.Errorf("unsupported curve %s", k.Crv)
	}
	xb, err := base64.RawURLEncoding.DecodeString(k.X)
	if err != nil {
		return nil, err
	}
	yb, err := base64.RawURLEncoding.DecodeString(k.Y)
	if err != nil {
		return nil, err
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xb),
		Y:     new(big.Int).SetBytes(yb),
	}, nil
}

func fetchSupabaseJWKS(supabaseURL string) (*supabaseJWKS, error) {
	base := strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	if base == "" {
		return nil, errors.New("supabase url missing for jwks")
	}
	url := base + "/auth/v1/.well-known/jwks.json"

	supabaseJWKSMu.Lock()
	if supabaseJWKSCache != nil && supabaseJWKSURL == url && time.Since(supabaseJWKSAt) < 10*time.Minute {
		cached := supabaseJWKSCache
		supabaseJWKSMu.Unlock()
		return cached, nil
	}
	supabaseJWKSMu.Unlock()

	// Fetch outside the mutex so concurrent auth requests are not stalled.
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	var jwks supabaseJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("empty jwks")
	}

	supabaseJWKSMu.Lock()
	supabaseJWKSCache = &jwks
	supabaseJWKSAt = time.Now()
	supabaseJWKSURL = url
	cached := supabaseJWKSCache
	supabaseJWKSMu.Unlock()
	return cached, nil
}

func invalidateSupabaseJWKS() {
	supabaseJWKSMu.Lock()
	defer supabaseJWKSMu.Unlock()
	supabaseJWKSCache = nil
	supabaseJWKSURL = ""
	supabaseJWKSAt = time.Time{}
}

// verifyHS256JWT is a test helper for the legacy shared-secret path only.
func verifyHS256JWT(token, secret string) (sub string, provider string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("malformed jwt")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", err
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", "", err
	}
	if !strings.EqualFold(strings.TrimSpace(header.Alg), "HS256") {
		return "", "", fmt.Errorf("not hs256")
	}
	return verifySupabaseAccessToken(token, secret, "")
}
