package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// chromeAutostartUsable reports whether chrome has the DB-backed fields the
// everyday auto-start path needs. Empty = fail-closed (do not invent).
func chromeAutostartUsable(c cliChrome) bool {
	return strings.TrimSpace(c.DefaultAutoStartWithIDE) != "" &&
		strings.TrimSpace(c.ProxyHealthURL) != "" &&
		strings.TrimSpace(c.ProxyHealthTimeoutMs) != "" &&
		strings.TrimSpace(c.AutostartPrefPollMs) != "" &&
		strings.TrimSpace(c.AutostartPrefPollMinMs) != "" &&
		strings.TrimSpace(c.AutostartPrefPollMaxMs) != "" &&
		strings.TrimSpace(c.AutostartTimeoutMinMs) != "" &&
		strings.TrimSpace(c.AutostartTimeoutMaxMs) != "" &&
		strings.TrimSpace(c.ProxyFallbackUncompressed) != "" &&
		strings.TrimSpace(c.ProxyActiveFileProtection) != ""
}

func cliChromeCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "trim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "cli-chrome-cache.json"), nil
}

// writeCLIChromeCache stores last-known-good API chrome (DB-sourced via auth-providers).
// Used so daemon/login can honor auto-start when the API is briefly unreachable.
func writeCLIChromeCache(c cliChrome) error {
	path, err := cliChromeCachePath()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func readCLIChromeCache() (cliChrome, bool) {
	path, err := cliChromeCachePath()
	if err != nil {
		return cliChrome{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cliChrome{}, false
	}
	var c cliChrome
	if err := json.Unmarshal(raw, &c); err != nil {
		return cliChrome{}, false
	}
	if !chromeAutostartUsable(c) {
		return cliChrome{}, false
	}
	return c, true
}
