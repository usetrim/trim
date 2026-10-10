package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// chromeTimeoutBoundsMs reads AUTOSTART_TIMEOUT_MIN_MS / MAX_MS. Fail-closed.
func chromeTimeoutBoundsMs(chrome cliChrome) (minMs, maxMs int, ok bool) {
	minMs, errMin := strconv.Atoi(strings.TrimSpace(chrome.AutostartTimeoutMinMs))
	maxMs, errMax := strconv.Atoi(strings.TrimSpace(chrome.AutostartTimeoutMaxMs))
	if errMin != nil || errMax != nil || minMs < 1 || maxMs < 1 || minMs > maxMs {
		return 0, 0, false
	}
	return minMs, maxMs, true
}

// chromeTimeoutMs parses a positive site_messages millisecond body within DB bounds.
func chromeTimeoutMs(raw string, chrome cliChrome) time.Duration {
	minMs, maxMs, ok := chromeTimeoutBoundsMs(chrome)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < minMs || n > maxMs {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}

func requireChromeTimeout(raw, code string, chrome cliChrome) (time.Duration, error) {
	d := chromeTimeoutMs(raw, chrome)
	if d > 0 {
		return d, nil
	}
	if chrome.ChromeUnavailable != "" {
		return 0, fmt.Errorf("%s: %s", code, chrome.ChromeUnavailable)
	}
	return 0, fmt.Errorf("%s missing or invalid", code)
}

// autostartPrefPollIntervalMs reads CLI_AUTOSTART_PREF_POLL_MS against
// AUTOSTART_PREF_POLL_MIN_MS / MAX_MS from site_messages. Fail-closed: ok=false
// means do not invent an interval. ms==0 with ok=true means polling disabled.
func autostartPrefPollIntervalMs(chrome cliChrome) (ms int, ok bool) {
	minMs, errMin := strconv.Atoi(strings.TrimSpace(chrome.AutostartPrefPollMinMs))
	maxMs, errMax := strconv.Atoi(strings.TrimSpace(chrome.AutostartPrefPollMaxMs))
	if errMin != nil || errMax != nil || minMs < 1 || maxMs < 1 || minMs > maxMs {
		return 0, false
	}
	raw := strings.TrimSpace(chrome.AutostartPrefPollMs)
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, false
	}
	if n == 0 {
		return 0, true
	}
	if n < minMs || n > maxMs {
		return 0, false
	}
	return n, true
}

// chromeProxyFallbackUncompressed reads CLI_PROXY_FALLBACK_UNCOMPRESSED.
// Fail-closed: missing/invalid => false (do not invent true).
func chromeProxyFallbackUncompressed(chrome cliChrome) bool {
	v := strings.ToLower(strings.TrimSpace(chrome.ProxyFallbackUncompressed))
	return v == "true"
}

// chromeProxyActiveFileProtection reads PROXY_ACTIVE_FILE_PROTECTION.
// Fail-closed: missing/invalid => false (do not invent true).
func chromeProxyActiveFileProtection(chrome cliChrome) bool {
	v := strings.ToLower(strings.TrimSpace(chrome.ProxyActiveFileProtection))
	return v == "true"
}
