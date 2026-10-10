package cli

import (
	"net/http"
	"time"

	"github.com/usetrim/trim/cli/internal/config"
)

// newCLIHTTPClient builds an HTTP client with the ops-configured CLI timeout.
// Timeout must come from TRIM_CLI_HTTP_TIMEOUT_SEC (no invent 30s/8s/3s).
func newCLIHTTPClient(cfg config.Local) *http.Client {
	sec := cfg.CLIHTTPTimeoutSec
	if sec < 1 {
		sec = 1
	}
	return &http.Client{Timeout: time.Duration(sec) * time.Second}
}
