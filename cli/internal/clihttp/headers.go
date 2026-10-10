// Package clihttp attaches Trim cloud auth headers for CLI clients.
package clihttp

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/usetrim/trim/cli/internal/fingerprint"
	"github.com/usetrim/trim/server/pkg/clisign"
)

// AttachAuthHeaders sets Authorization, agent, hardware, timestamp, optional HMAC, and UA.
// version is the CLI semver string (e.g. from main). agentID must be a catalog id (usually "cli").
func AttachAuthHeaders(req *http.Request, token, hmacSecret, version, agentID string, body []byte) error {
	if req == nil {
		return fmt.Errorf("nil request")
	}
	ts := fmt.Sprintf("%d", time.Now().Unix())
	req.Header.Set("Authorization", "Bearer "+token)
	if version != "" {
		req.Header.Set("X-Client-Version", version)
		req.Header.Set("User-Agent", "TrimCLI/"+version)
	}
	if agentID == "" {
		agentID = "cli"
	}
	req.Header.Set("X-Trim-Agent-Id", agentID)
	req.Header.Set("X-Request-Timestamp", ts)
	if hmacSecret != "" {
		req.Header.Set(clisign.HeaderName, clisign.Sign(hmacSecret, ts, req.Method, req.URL.Path, body))
	}
	hw, err := fingerprint.HardwareUUID()
	if err != nil {
		return err
	}
	req.Header.Set("X-Hardware-UUID", hw)
	if ws := strings.TrimSpace(os.Getenv("TRIM_WORKSPACE_ID")); ws != "" {
		req.Header.Set("X-Workspace-Id", ws)
	}
	return nil
}
