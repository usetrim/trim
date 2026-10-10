package platformadmin

import (
	"crypto"
	"crypto/rsa"
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

// Cloudflare Access JWT gate: when CF_ACCESS_TEAM_DOMAIN and CF_ACCESS_AUD are both set,
// every admin request must present a valid Cf-Access-Jwt-Assertion. Empty pair = gate off
// (self-host without CF Access). Partial config is rejected at boot.

type cfJWKS struct {
	Keys []cfJWK `json:"keys"`
}

type cfJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

var (
	cfJWKSMu    sync.Mutex
	cfJWKSCache *cfJWKS
	cfJWKSAt    time.Time
)

func (h *Handler) adminCFAccessOK(r *http.Request) (bool, string) {
	team := strings.TrimSpace(h.Config.CFAccessTeamDomain)
	aud := strings.TrimSpace(h.Config.CFAccessAUD)
	if team == "" && aud == "" {
		return true, ""
	}
	if team == "" || aud == "" {
		return false, "ADMIN_CF_ACCESS_INVALID"
	}
	tok := strings.TrimSpace(r.Header.Get("Cf-Access-Jwt-Assertion"))
	if tok == "" {
		return false, "ADMIN_CF_ACCESS_REQUIRED"
	}
	if err := verifyCFAccessJWT(tok, team, aud); err != nil {
		return false, "ADMIN_CF_ACCESS_INVALID"
	}
	return true, ""
}

func verifyCFAccessJWT(token, teamDomain, audience string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("malformed jwt")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return err
	}
	if !strings.EqualFold(header.Alg, "RS256") || strings.TrimSpace(header.Kid) == "" {
		return errors.New("unsupported alg")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	var claims struct {
		Aud any    `json:"aud"`
		Iss string `json:"iss"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return err
	}
	if claims.Exp <= 0 || time.Now().Unix() > claims.Exp {
		return errors.New("expired")
	}
	issExpected := "https://" + strings.TrimSuffix(strings.TrimSpace(teamDomain), "/")
	if !strings.EqualFold(strings.TrimSpace(claims.Iss), issExpected) {
		return errors.New("iss mismatch")
	}
	if !audMatches(claims.Aud, audience) {
		return errors.New("aud mismatch")
	}
	jwks, err := fetchCFJWKS(teamDomain)
	if err != nil {
		return err
	}
	var key *rsa.PublicKey
	for _, k := range jwks.Keys {
		if k.Kid == header.Kid && strings.EqualFold(k.Kty, "RSA") {
			pub, err := jwkToRSA(k)
			if err != nil {
				return err
			}
			key = pub
			break
		}
	}
	if key == nil {
		return errors.New("kid not found")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig)
}

func audMatches(raw any, want string) bool {
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

func fetchCFJWKS(teamDomain string) (*cfJWKS, error) {
	cfJWKSMu.Lock()
	defer cfJWKSMu.Unlock()
	if cfJWKSCache != nil && time.Since(cfJWKSAt) < 10*time.Minute {
		return cfJWKSCache, nil
	}
	url := fmt.Sprintf("https://%s/cdn-cgi/access/certs", strings.TrimSuffix(strings.TrimSpace(teamDomain), "/"))
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	var jwks cfJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("empty jwks")
	}
	cfJWKSCache = &jwks
	cfJWKSAt = time.Now()
	return cfJWKSCache, nil
}

func jwkToRSA(k cfJWK) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, err
	}
	var eInt int
	for _, b := range eb {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, errors.New("invalid e")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: eInt}, nil
}
