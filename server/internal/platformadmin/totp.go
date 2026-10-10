package platformadmin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
)

// totpPeriodSec is the standard RFC 6238 period.
const totpPeriodSec int64 = 30

func parseTOTPKey(hexKey string) ([]byte, error) {
	hexKey = strings.TrimSpace(hexKey)
	if hexKey == "" {
		return nil, fmt.Errorf("empty")
	}
	b, err := hex.DecodeString(hexKey)
	if err != nil || len(b) != 32 {
		return nil, fmt.Errorf("invalid")
	}
	return b, nil
}

func encryptSecret(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

func decryptSecret(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func generateTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func hotp(secret []byte, counter uint64) int {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset]&0x7f) << 24) |
		(int(sum[offset+1]&0xff) << 16) |
		(int(sum[offset+2]&0xff) << 8) |
		int(sum[offset+3]&0xff)
	return code % int(math.Pow10(6))
}

func verifyTOTP(secretBase32, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	var want int
	_, err := fmt.Sscanf(code, "%d", &want)
	if err != nil {
		return false
	}
	sec, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secretBase32)))
	if err != nil || len(sec) == 0 {
		return false
	}
	counter := now.Unix() / totpPeriodSec
	for _, d := range []int64{-1, 0, 1} {
		if hotp(sec, uint64(counter+d)) == want {
			return true
		}
	}
	return false
}

func otpAuthURL(issuer, account, secret string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "Trim Admin"
	}
	account = strings.TrimSpace(account)
	if account == "" {
		account = "admin"
	}
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	v.Set("algorithm", "SHA1")
	v.Set("digits", "6")
	v.Set("period", "30")
	label := url.PathEscape(issuer + ":" + account)
	return "otpauth://totp/" + label + "?" + v.Encode()
}
