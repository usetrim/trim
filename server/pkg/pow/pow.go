package pow

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HeaderName is the CLI / client proof-of-work response header.
const HeaderName = "X-Trim-PoW"

// ChallengeHeader is set on 428 responses so clients can solve and retry.
const ChallengeHeader = "X-Trim-PoW-Challenge"

// DefaultDifficulty is used when ops leave TRIM_POW_DIFFICULTY unset (free tier only).
const DefaultDifficulty = 16

// Challenge is a time-bucketed Hashcash-style puzzle scoped to a user.
type Challenge struct {
	UserID     string
	Bucket     int64
	Difficulty int
}

// Encode serializes the challenge for the response header.
func (c Challenge) Encode() string {
	return fmt.Sprintf("%s:%d:%d", c.UserID, c.Bucket, c.Difficulty)
}

// ParseChallenge parses Encode() output.
func ParseChallenge(raw string) (Challenge, error) {
	parts := strings.Split(raw, ":")
	if len(parts) != 3 {
		return Challenge{}, fmt.Errorf("invalid challenge")
	}
	bucket, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Challenge{}, err
	}
	diff, err := strconv.Atoi(parts[2])
	if err != nil || diff < 8 || diff > 24 {
		return Challenge{}, fmt.Errorf("invalid difficulty")
	}
	return Challenge{UserID: parts[0], Bucket: bucket, Difficulty: diff}, nil
}

// CurrentChallenge builds a challenge for the current 5-minute UTC bucket.
func CurrentChallenge(userID string, difficulty int) Challenge {
	if difficulty < 8 {
		difficulty = DefaultDifficulty
	}
	if difficulty > 24 {
		difficulty = 24
	}
	bucket := time.Now().UTC().Unix() / 300
	return Challenge{UserID: userID, Bucket: bucket, Difficulty: difficulty}
}

// Verify checks that nonce solves the challenge (SHA-256 leading zero bits).
func Verify(c Challenge, nonce string) bool {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" || c.UserID == "" {
		return false
	}
	payload := fmt.Sprintf("%s:%d:%s", c.UserID, c.Bucket, nonce)
	sum := sha256.Sum256([]byte(payload))
	return hasLeadingZeroBits(sum[:], c.Difficulty)
}

// Solve finds a nonce for the challenge. Used by the Trim CLI on free tier.
func Solve(c Challenge) string {
	for i := 0; ; i++ {
		nonce := strconv.FormatInt(int64(i), 16)
		if Verify(c, nonce) {
			return nonce
		}
		if i > 1<<24 {
			return ""
		}
	}
}

func hasLeadingZeroBits(digest []byte, bits int) bool {
	if bits <= 0 {
		return true
	}
	fullBytes := bits / 8
	rem := bits % 8
	if fullBytes > len(digest) {
		return false
	}
	for i := 0; i < fullBytes; i++ {
		if digest[i] != 0 {
			return false
		}
	}
	if rem == 0 {
		return true
	}
	if fullBytes >= len(digest) {
		return false
	}
	mask := byte(0xFF << (8 - rem))
	return digest[fullBytes]&mask == 0
}
