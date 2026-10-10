package telemetry

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/usetrim/trim/server/internal/subscriptions"
)

// Handler accepts anonymous CLI telemetry. No auth required.
// Respects privacy: rejects payloads that look like secrets.
type Handler struct {
	Redis  *redis.Client
	TTLSec int // Redis counter TTL; 0 skips Expire (self-host optional)
}

func NewHandler(rdb *redis.Client, ttlSec int) *Handler {
	return &Handler{Redis: rdb, TTLSec: ttlSec}
}

func writeJSONErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": subscriptions.MessageForCode(code),
	})
}

type eventPayload struct {
	Event      string                 `json:"event"`
	Properties map[string]interface{} `json:"properties"`
	TS         string                 `json:"ts"`
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	var p eventPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&p); err != nil {
		writeJSONErr(w, http.StatusBadRequest, "TELEMETRY_INVALID_PAYLOAD")
		return
	}
	p.Event = strings.TrimSpace(p.Event)
	if p.Event == "" || len(p.Event) > 64 {
		writeJSONErr(w, http.StatusBadRequest, "TELEMETRY_INVALID_EVENT")
		return
	}
	if looksSensitive(p.Properties) {
		writeJSONErr(w, http.StatusBadRequest, "TELEMETRY_REJECTED")
		return
	}
	if p.TS == "" {
		p.TS = time.Now().UTC().Format(time.RFC3339)
	}

	if h.Redis != nil {
		key := "telemetry:counter:" + p.Event
		_ = h.Redis.Incr(r.Context(), key).Err()
		if h.TTLSec > 0 {
			_ = h.Redis.Expire(r.Context(), key, time.Duration(h.TTLSec)*time.Second).Err()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func looksSensitive(props map[string]interface{}) bool {
	if props == nil {
		return false
	}
	blocked := []string{"email", "token", "authorization", "api_key", "password", "prompt", "body"}
	for k := range props {
		lk := strings.ToLower(k)
		for _, b := range blocked {
			if lk == b || strings.Contains(lk, b) {
				return true
			}
		}
	}
	return false
}
