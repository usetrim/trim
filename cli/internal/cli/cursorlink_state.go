package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cursorLinkState is persisted under ~/.trim/cursor-link.json.
// enabled = user wants a Cursor Agent public HTTPS link while Trim runs.
// AccessToken gates /cl/<token>/v1 on the local proxy (rotate to revoke).
// Runtime fields (URL, PID) are filled when the tunnel is up.
type cursorLinkState struct {
	Enabled        bool   `json:"enabled"`
	AccessToken    string `json:"access_token,omitempty"`
	PublicOrigin   string `json:"public_origin,omitempty"`
	PublicBaseURL  string `json:"public_base_url,omitempty"`
	LocalPort      string `json:"local_port,omitempty"`
	PID            int    `json:"pid,omitempty"`
	CloudflaredBin string `json:"cloudflared_bin,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

func cursorLinkStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".trim")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "cursor-link.json"), nil
}

func loadCursorLinkState() (cursorLinkState, error) {
	path, err := cursorLinkStatePath()
	if err != nil {
		return cursorLinkState{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cursorLinkState{}, nil
		}
		return cursorLinkState{}, err
	}
	var st cursorLinkState
	if err := json.Unmarshal(b, &st); err != nil {
		return cursorLinkState{}, err
	}
	return st, nil
}

func saveCursorLinkState(st cursorLinkState) error {
	path, err := cursorLinkStatePath()
	if err != nil {
		return err
	}
	st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func cursorLinkPublicBaseURL(st cursorLinkState) string {
	if u := strings.TrimSpace(st.PublicBaseURL); u != "" {
		return u
	}
	origin := strings.TrimRight(strings.TrimSpace(st.PublicOrigin), "/")
	tok := strings.TrimSpace(st.AccessToken)
	if origin == "" || tok == "" {
		return ""
	}
	return origin + "/cl/" + tok + "/v1"
}

func ensureCursorLinkAccessToken(st *cursorLinkState) error {
	if st == nil {
		return nil
	}
	if strings.TrimSpace(st.AccessToken) != "" {
		return nil
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	st.AccessToken = hex.EncodeToString(b)
	return nil
}

func rotateCursorLinkAccessToken() (string, error) {
	st, err := loadCursorLinkState()
	if err != nil {
		return "", err
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	st.AccessToken = hex.EncodeToString(b)
	// Force URL rebuild on next ensure / status.
	if origin := strings.TrimRight(strings.TrimSpace(st.PublicOrigin), "/"); origin != "" {
		st.PublicBaseURL = origin + "/cl/" + st.AccessToken + "/v1"
	} else {
		st.PublicBaseURL = ""
	}
	if err := saveCursorLinkState(st); err != nil {
		return "", err
	}
	return st.AccessToken, nil
}
