package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IsEnabled follows the Global CLI Telemetry Standard.
// Disabled when DO_NOT_TRACK=1, TRIM_TELEMETRY_DISABLED=1, or local opt-out file.
func IsEnabled() bool {
	if os.Getenv("DO_NOT_TRACK") == "1" {
		return false
	}
	if os.Getenv("TRIM_TELEMETRY_DISABLED") == "1" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	path := home + string(os.PathSeparator) + ".config" + string(os.PathSeparator) + "trim" + string(os.PathSeparator) + "telemetry.off"
	if _, err := os.Stat(path); err == nil {
		return false
	}
	return true
}

// Disable writes a local opt-out marker (~/.config/trim/telemetry.off).
func Disable() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := home + string(os.PathSeparator) + ".config" + string(os.PathSeparator) + "trim"
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := dir + string(os.PathSeparator) + "telemetry.off"
	return os.WriteFile(path, []byte("disabled\n"), 0o600)
}

// Enable removes the local opt-out marker.
func Enable() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := home + string(os.PathSeparator) + ".config" + string(os.PathSeparator) + "trim" + string(os.PathSeparator) + "telemetry.off"
	_ = os.Remove(path)
	return nil
}

var (
	once   sync.Once
	client *http.Client
)

func httpClient() *http.Client {
	once.Do(func() {
		raw := strings.TrimSpace(os.Getenv("TRIM_CLI_HTTP_TIMEOUT_SEC"))
		sec := 0
		if raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 300 {
				sec = n
			}
		}
		if sec < 1 {
			// Fail closed: no invent 3s; skip telemetry HTTP when timeout unset.
			client = nil
			return
		}
		client = &http.Client{Timeout: time.Duration(sec) * time.Second}
	})
	return client
}

// CaptureEvent sends an anonymous event to the cloud API when telemetry is on.
// Never includes tokens, emails, file paths, or prompt contents.
func CaptureEvent(apiBaseURL, eventName string, properties map[string]interface{}) {
	if !IsEnabled() {
		return
	}
	apiBaseURL = strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if apiBaseURL == "" || eventName == "" {
		return
	}

	props := sanitize(properties)
	body, err := json.Marshal(map[string]interface{}{
		"event":      eventName,
		"properties": props,
		"ts":         time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	go func() {
		c := httpClient()
		if c == nil {
			return
		}
		req, err := http.NewRequest(http.MethodPost, apiBaseURL+"/api/v1/telemetry", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TrimTelemetry/1.0")
		res, err := c.Do(req)
		if err != nil {
			return
		}
		_ = res.Body.Close()
	}()
}

func sanitize(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(props))
	blocked := map[string]struct{}{
		"email": {}, "token": {}, "authorization": {}, "api_key": {},
		"home_directory": {}, "path": {}, "prompt": {}, "body": {},
	}
	for k, v := range props {
		lk := strings.ToLower(k)
		if _, bad := blocked[lk]; bad {
			continue
		}
		out[k] = v
	}
	return out
}
