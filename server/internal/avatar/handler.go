package avatar

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/usetrim/trim/server/internal/middleware"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// Handler optionally syncs profile avatars to Cloudinary when credentials are configured.
type Handler struct {
	DB           *pgxpool.Pool
	CloudName    string
	APIKey       string
	APISecret    string
	UploadPreset string
	HTTP         *http.Client
}

func NewHandler(db *pgxpool.Pool, httpTimeoutSec int) *Handler {
	h := &Handler{
		DB:           db,
		CloudName:    strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME")),
		APIKey:       strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY")),
		APISecret:    strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET")),
		UploadPreset: strings.TrimSpace(os.Getenv("CLOUDINARY_UPLOAD_PRESET")),
	}
	if h.configured() {
		if httpTimeoutSec < 1 || httpTimeoutSec > 300 {
			// Fail closed: caller must pass CLOUDINARY_HTTP_TIMEOUT_SEC via config.
			h.HTTP = nil
		} else {
			h.HTTP = &http.Client{Timeout: time.Duration(httpTimeoutSec) * time.Second}
		}
	}
	return h
}
func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

func (h *Handler) configured() bool {
	return h != nil && h.CloudName != "" && (h.UploadPreset != "" || (h.APIKey != "" && h.APISecret != ""))
}

type syncResponse struct {
	Status    string `json:"status"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Message   string `json:"message,omitempty"`
}

// Sync fetches the user's current avatar_url and re-hosts it on Cloudinary when enabled.
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		writeJSONErr(w, http.StatusUnauthorized, "WS_UNAUTHORIZED")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if !h.configured() {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(syncResponse{
			Status:  "skipped",
			Message: subscriptions.MessageForCode("AVATAR_CLOUDINARY_UNCONFIGURED"),
		})
		return
	}
	if h.HTTP == nil {
		writeJSONErr(w, http.StatusServiceUnavailable, "AVATAR_HTTP_TIMEOUT_MISSING")
		return
	}

	var current string
	err := h.DB.QueryRow(r.Context(), `
		select coalesce(avatar_url, '') from public.profiles where id = $1
	`, userID).Scan(&current)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": subscriptions.MessageForCode("PROFILE_NOT_FOUND"),
		})
		return
	}
	current = strings.TrimSpace(current)
	if current == "" {
		_ = json.NewEncoder(w).Encode(syncResponse{Status: "skipped", Message: subscriptions.MessageForCode("AVATAR_NO_URL")})
		return
	}
	if strings.Contains(current, "res.cloudinary.com") {
		_ = json.NewEncoder(w).Encode(syncResponse{Status: "ok", AvatarURL: current, Message: subscriptions.MessageForCode("AVATAR_ALREADY_CDN")})
		return
	}

	cdnURL, err := h.uploadByURL(r.Context(), current, userID)
	if err != nil {
		writeJSONErr(w, http.StatusBadGateway, "AVATAR_UPLOAD_FAILED")
		return
	}

	_, err = h.DB.Exec(r.Context(), `
		update public.profiles set avatar_url = $2, updated_at = now() where id = $1
	`, userID, cdnURL)
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "AVATAR_PROFILE_UPDATE_FAILED")
		return
	}

	_ = json.NewEncoder(w).Encode(syncResponse{
		Status:    "ok",
		AvatarURL: cdnURL,
		Message:   subscriptions.MessageForCode("AVATAR_SYNC_DONE"),
	})
}

func (h *Handler) uploadByURL(ctx context.Context, sourceURL, userID string) (string, error) {
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", url.PathEscape(h.CloudName))
	form := url.Values{}
	form.Set("file", sourceURL)
	form.Set("folder", "trim/avatars")
	form.Set("public_id", "user_"+userID)
	form.Set("overwrite", "true")

	if h.UploadPreset != "" {
		form.Set("upload_preset", h.UploadPreset)
	} else {
		ts := fmt.Sprintf("%d", time.Now().Unix())
		form.Set("timestamp", ts)
		form.Set("api_key", h.APIKey)
		// Signature: sorted params excluding file/api_key/resource_type + api_secret
		toSign := fmt.Sprintf("folder=trim/avatars&overwrite=true&public_id=user_%s&timestamp=%s%s", userID, ts, h.APISecret)
		sum := sha1.Sum([]byte(toSign))
		form.Set("signature", hex.EncodeToString(sum[:]))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("cloudinary upload failed (%d)", resp.StatusCode)
	}
	var parsed struct {
		SecureURL string `json:"secure_url"`
		URL       string `json:"url"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	out := strings.TrimSpace(parsed.SecureURL)
	if out == "" {
		out = strings.TrimSpace(parsed.URL)
	}
	if out == "" {
		return "", fmt.Errorf("cloudinary returned empty url")
	}
	return out, nil
}
