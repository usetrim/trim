package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/clierr"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/storage"
	"github.com/usetrim/trim/cli/internal/tlsgen"
)

var (
	setupTLS           bool
	quotaExhaustedOnce sync.Once
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)

		if setupTLS {
			dir, err := tlsgen.ConfigDir()
			if err != nil {
				return err
			}
			org := strings.TrimSpace(os.Getenv("TRIM_TLS_ORG"))
			daysRaw := strings.TrimSpace(os.Getenv("TRIM_TLS_VALIDITY_DAYS"))
			if org == "" || daysRaw == "" {
				msg := strings.TrimSpace(chrome.SetupTLSEnvRequired)
				if msg == "" {
					msg = "CLI_SETUP_TLS_ENV_REQUIRED"
				}
				return fmt.Errorf("%s", msg)
			}
			days, err := strconv.Atoi(daysRaw)
			if err != nil || days < 1 {
				msg := strings.TrimSpace(chrome.SetupTLSDaysInvalid)
				if msg == "" {
					msg = "CLI_SETUP_TLS_DAYS_INVALID"
				}
				return fmt.Errorf("%s", msg)
			}
			certPath, keyPath, err := tlsgen.GenerateLocal(dir, tlsgen.Options{
				Organization: org,
				ValidityDays: days,
			})
			if err != nil {
				return err
			}
			if chrome.SetupTLSGeneratedFmt != "" {
				fmt.Printf(chrome.SetupTLSGeneratedFmt+"\n", certPath)
			} else {
				fmt.Println(certPath)
			}
			if chrome.SetupTLSKeyPathFmt != "" {
				fmt.Printf(chrome.SetupTLSKeyPathFmt+"\n", keyPath)
			}
			fmt.Println()
			if chrome.SetupTLSThen != "" {
				fmt.Println(chrome.SetupTLSThen)
			}
			if chrome.SetupTLSCertFmt != "" {
				fmt.Printf(chrome.SetupTLSCertFmt+"\n", certPath)
			}
			if chrome.SetupTLSKeyFmt != "" {
				fmt.Printf(chrome.SetupTLSKeyFmt+"\n", keyPath)
			}
			if chrome.SetupTLSStart != "" {
				fmt.Println(chrome.SetupTLSStart)
			}
			if chrome.SetupTLSPointIDE != "" {
				fmt.Println(chrome.SetupTLSPointIDE)
			}
			return nil
		}

		scheme := "http"
		if strings.TrimSpace(os.Getenv("TRIM_TLS_CERT")) != "" && strings.TrimSpace(os.Getenv("TRIM_TLS_KEY")) != "" {
			scheme = "https"
		}
		// Cursor / OpenAI SDKs need …/v1. Claude Code needs the listen ORIGIN only (no /v1).
		originURL := fmt.Sprintf("%s://localhost:%s", scheme, cfg.Port)
		baseURL := originURL + "/v1"
		var wrote []string
		wroteFmt := chrome.SetupWroteFmt

		if p, err := ideUserSettingsPath("Cursor"); err == nil {
			if err := mergeOpenAIBaseURL(p, baseURL, cursorKeys()); err == nil {
				wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductCursor, p))
			}
		}
		if p, err := ideUserSettingsPath("Code"); err == nil {
			if err := mergeOpenAIBaseURL(p, baseURL, vscodeContinueKeys(baseURL)); err == nil {
				wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductVSCode, p))
			}
		}
		if p, err := ideUserSettingsPath("Windsurf"); err == nil {
			if err := mergeOpenAIBaseURL(p, baseURL, windsurfKeys()); err == nil {
				wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductWindsurf, p))
			}
		}
		if p, err := continueConfigPath(); err == nil {
			if err := mergeContinueConfig(p, baseURL, chrome.SetupContinueNoModel, chrome.ChromeUnavailable); err == nil {
				wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductContinue, p))
			}
		}
		if p, err := zedSettingsPath(); err == nil {
			if err := mergeZedSettings(p, baseURL); err == nil {
				wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductZed, p))
			}
		}
		if p, err := writeJetBrainsHint(baseURL); err == nil {
			wrote = append(wrote, formatSetupWrote(wroteFmt, chrome.SetupProductJetbrains, p))
		}

		if chrome.SetupEndpointFmt != "" {
			fmt.Printf(chrome.SetupEndpointFmt+"\n\n", baseURL)
		} else {
			fmt.Printf("%s\n\n", baseURL)
		}
		if len(wrote) == 0 {
			if chrome.SetupNoneFound != "" {
				fmt.Println(chrome.SetupNoneFound)
			}
		} else {
			if chrome.SetupUpdatedHeader != "" {
				fmt.Println(chrome.SetupUpdatedHeader)
			}
			for _, line := range wrote {
				fmt.Println("  " + line)
			}
		}
		fmt.Println()
		if chrome.SetupShellHeader != "" {
			fmt.Println(chrome.SetupShellHeader)
		}
		if chrome.SetupShellOpenAIFmt != "" {
			fmt.Printf(chrome.SetupShellOpenAIFmt+"\n", baseURL)
		}
		if chrome.SetupShellAnthropicFmt != "" {
			fmt.Printf(chrome.SetupShellAnthropicFmt+"\n", originURL)
		}
		if chrome.SetupClaudeCodeBlockFmt != "" {
			fmt.Println()
			fmt.Printf(chrome.SetupClaudeCodeBlockFmt+"\n", originURL)
		}
		if chrome.SetupContinueBlockFmt != "" {
			fmt.Println()
			fmt.Printf(chrome.SetupContinueBlockFmt+"\n", baseURL)
		}
		if chrome.SetupVSCodeChatBlockFmt != "" {
			fmt.Println()
			fmt.Printf(chrome.SetupVSCodeChatBlockFmt+"\n", baseURL+"/chat/completions")
		}
		fmt.Println()
		if chrome.SetupJetbrainsHint != "" {
			fmt.Println(chrome.SetupJetbrainsHint)
		}
		if chrome.SetupAPIEndpointFmt != "" {
			fmt.Printf(chrome.SetupAPIEndpointFmt+"\n", baseURL)
		}
		fmt.Println()
		if chrome.SetupNextHeader != "" {
			fmt.Println(chrome.SetupNextHeader)
		}
		if chrome.SetupStepStart != "" {
			fmt.Println(chrome.SetupStepStart)
		}
		if chrome.SetupStepRestart != "" {
			fmt.Println(chrome.SetupStepRestart)
		}
		if chrome.SetupStepLogin != "" {
			fmt.Println(chrome.SetupStepLogin)
		}
		if chrome.SetupStepDashboardFmt != "" {
			fmt.Printf(chrome.SetupStepDashboardFmt+"\n", cfg.AppPublicURL)
		}
		if chrome.SetupOptionalTLS != "" {
			fmt.Println(chrome.SetupOptionalTLS)
		}
		return nil
	},
}

func formatSetupWrote(fmtStr, product, path string) string {
	product = strings.TrimSpace(product)
	path = strings.TrimSpace(path)
	if fmtStr == "" || product == "" {
		// Fail closed: no invent join when chrome SetupWroteFmt is empty.
		return ""
	}
	return fmt.Sprintf(fmtStr, product, path)
}

func cursorKeys() map[string]interface{} {
	return map[string]interface{}{
		"openai.baseUrl":               nil, // filled by merge
		"cursor.openai.baseUrl":        nil,
		"cursor.general.openaiBaseUrl": nil,
	}
}

func windsurfKeys() map[string]interface{} {
	return map[string]interface{}{
		"openai.baseUrl":          nil,
		"windsurf.openai.baseUrl": nil,
		"codeium.openai.baseUrl":  nil,
	}
}

func vscodeContinueKeys(baseURL string) map[string]interface{} {
	return map[string]interface{}{
		"openai.baseUrl": baseURL,
	}
}

func ideUserSettingsPath(product string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var dir string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir = filepath.Join(appData, product, "User")
	case "darwin":
		dir = filepath.Join(home, "Library", "Application Support", product, "User")
	default:
		cfgName := strings.ToLower(product)
		if product == "Code" {
			cfgName = "Code"
		}
		dir = filepath.Join(home, ".config", cfgName, "User")
	}
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func continueConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".continue")
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	// Modern Continue (schema v1) uses config.yaml. Prefer it when present so
	// trim setup does not leave users editing an unused legacy config.json.
	yamlPath := filepath.Join(dir, "config.yaml")
	jsonPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(yamlPath); err == nil {
		return yamlPath, nil
	}
	if _, err := os.Stat(jsonPath); err == nil {
		return jsonPath, nil
	}
	return yamlPath, nil
}

func zedSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var dir string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		dir = filepath.Join(appData, "Zed")
	case "darwin":
		dir = filepath.Join(home, "Library", "Application Support", "Zed")
	default:
		dir = filepath.Join(home, ".config", "zed")
	}
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func mergeZedSettings(path, baseURL string) error {
	raw := map[string]interface{}{}
	data, err := os.ReadFile(path)
	if err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	lms, _ := raw["language_models"].(map[string]interface{})
	if lms == nil {
		lms = map[string]interface{}{}
	}
	openai, _ := lms["openai"].(map[string]interface{})
	if openai == nil {
		openai = map[string]interface{}{}
	}
	openai["api_url"] = baseURL
	lms["openai"] = openai
	raw["language_models"] = lms

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o600)
}

// writeJetBrainsHint stores the OpenAI-compatible endpoint for JetBrains AI Assistant.
// JetBrains does not use a single portable settings.json; operators paste this URL in the IDE.
func writeJetBrainsHint(baseURL string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "trim")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "jetbrains-openai.url")
	body := baseURL + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func mergeOpenAIBaseURL(settingsPath, baseURL string, keys map[string]interface{}) error {
	raw := map[string]interface{}{}
	data, err := os.ReadFile(settingsPath)
	if err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	for k := range keys {
		raw[k] = baseURL
	}
	// Always set the common OpenAI-compatible key.
	raw["openai.baseUrl"] = baseURL

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(settingsPath, out, 0o600)
}

func mergeContinueConfig(path, baseURL, noModelMessage, chromeUnavailable string) error {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return mergeContinueConfigYAML(path, baseURL, noModelMessage, chromeUnavailable)
	}
	return mergeContinueConfigJSON(path, baseURL, noModelMessage, chromeUnavailable)
}

func mergeContinueConfigJSON(path, baseURL, noModelMessage, chromeUnavailable string) error {
	raw := map[string]interface{}{}
	data, err := os.ReadFile(path)
	if err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Continue.dev uses models[].apiBase for OpenAI-compatible providers.
	models, _ := raw["models"].([]interface{})
	updated := false
	for i, m := range models {
		mm, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		provider, _ := mm["provider"].(string)
		if strings.EqualFold(provider, "openai") || strings.EqualFold(provider, "openai-compatible") || provider == "" {
			mm["apiBase"] = baseURL
			models[i] = mm
			updated = true
		}
	}
	if !updated {
		defaultModel := strings.TrimSpace(os.Getenv("TRIM_SETUP_DEFAULT_MODEL"))
		if defaultModel == "" {
			msg := strings.TrimSpace(noModelMessage)
			if msg == "" {
				msg = strings.TrimSpace(chromeUnavailable)
			}
			if msg == "" {
				return clierr.ErrChromeUnavailable
			}
			return fmt.Errorf("%s", msg)
		}
		defaultTitle := strings.TrimSpace(os.Getenv("TRIM_SETUP_DEFAULT_MODEL_TITLE"))
		if defaultTitle == "" {
			msg := strings.TrimSpace(noModelMessage)
			if msg == "" {
				msg = strings.TrimSpace(chromeUnavailable)
			}
			if msg == "" {
				return clierr.ErrChromeUnavailable
			}
			return fmt.Errorf("%s", msg)
		}
		models = append(models, map[string]interface{}{
			"title":    defaultTitle,
			"provider": "openai",
			"model":    defaultModel,
			"apiBase":  baseURL,
		})
	}
	raw["models"] = models

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o600)
}

// mergeContinueConfigYAML updates apiBase in Continue schema v1 YAML, or writes a
// minimal Trim model block when models is empty. Avoids a YAML dependency: only
// rewrites apiBase lines and empty models: [] - fail closed for exotic layouts.
func mergeContinueConfigYAML(path, baseURL, noModelMessage, chromeUnavailable string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(data)
	trimmed := strings.TrimSpace(text)

	defaultModel := strings.TrimSpace(os.Getenv("TRIM_SETUP_DEFAULT_MODEL"))
	if defaultModel == "" {
		defaultModel = "trim-claude-sonnet"
	}
	defaultTitle := strings.TrimSpace(os.Getenv("TRIM_SETUP_DEFAULT_MODEL_TITLE"))
	if defaultTitle == "" {
		defaultTitle = "Claude via Trim"
	}

	writeTemplate := func() error {
		body := fmt.Sprintf(`name: Main Config
version: 1.0.0
schema: v1
models:
  - name: %s
    provider: openai
    model: %s
    apiBase: %s
    apiKey: REPLACE_WITH_PROVIDER_KEY
    roles:
      - chat
      - edit
      - apply
    capabilities:
      - tool_use
    useResponsesApi: false
`, defaultTitle, defaultModel, baseURL)
		return os.WriteFile(path, []byte(body), 0o600)
	}

	if trimmed == "" || strings.Contains(trimmed, "models: []") || !strings.Contains(text, "models:") {
		return writeTemplate()
	}

	lines := strings.Split(text, "\n")
	changed := false
	hasAPIBase := false
	for i, line := range lines {
		trimLeft := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimLeft, "apiBase:") {
			hasAPIBase = true
			indent := line[:len(line)-len(trimLeft)]
			lines[i] = indent + "apiBase: " + baseURL
			changed = true
		}
	}
	if hasAPIBase && changed {
		out := ensureContinueTrimAgentFlags(strings.Join(lines, "\n"))
		if !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		return os.WriteFile(path, []byte(out), 0o600)
	}

	if !strings.Contains(text, "provider:") {
		return writeTemplate()
	}
	msg := strings.TrimSpace(noModelMessage)
	if msg == "" {
		msg = strings.TrimSpace(chromeUnavailable)
	}
	if msg == "" {
		return clierr.ErrChromeUnavailable
	}
	return fmt.Errorf("%s", msg)
}

// ensureContinueTrimAgentFlags adds capabilities tool_use and useResponsesApi: false
// under each openai model that already has apiBase, when those keys are missing.
// Keeps Continue Agent + Trim chat/completions without requiring a YAML library.
func ensureContinueTrimAgentFlags(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimLeft := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimLeft)]
		// Model list item: "  - name:" (or "- name:")
		if strings.HasPrefix(trimLeft, "- ") && strings.Contains(trimLeft, "name:") {
			blockStart := i
			itemIndent := indent
			childIndent := itemIndent + "  "
			j := i + 1
			hasOpenAI := false
			hasAPIBase := false
			hasToolUse := false
			hasUseResponses := false
			insertAt := -1
			for j < len(lines) {
				lj := lines[j]
				tj := strings.TrimLeft(lj, " \t")
				ij := lj[:len(lj)-len(tj)]
				// Next sibling model item at same indent ends the block.
				if tj != "" && len(ij) <= len(itemIndent) && strings.HasPrefix(tj, "- ") {
					break
				}
				if strings.HasPrefix(tj, "provider:") && strings.Contains(tj, "openai") {
					hasOpenAI = true
				}
				if strings.HasPrefix(tj, "apiBase:") {
					hasAPIBase = true
					insertAt = j
				}
				if strings.Contains(tj, "tool_use") {
					hasToolUse = true
				}
				if strings.HasPrefix(tj, "useResponsesApi:") {
					hasUseResponses = true
				}
				j++
			}
			// Copy model block lines.
			for k := blockStart; k < j; k++ {
				out = append(out, lines[k])
				if k == insertAt && hasOpenAI && hasAPIBase {
					if !hasToolUse {
						out = append(out, childIndent+"capabilities:")
						out = append(out, childIndent+"  - tool_use")
					}
					if !hasUseResponses {
						out = append(out, childIndent+"useResponsesApi: false")
					}
				}
			}
			i = j
			continue
		}
		out = append(out, line)
		i++
	}
	return strings.Join(out, "\n")
}

// reportEvent posts a trim metric to the cloud API when the CLI is logged in.
func reportEvent(cfg config.Local, model string, before, after int, latencyMs float64, status, mode string) {
	token, err := storage.GetToken()
	if err != nil || token == "" {
		return
	}
	if strings.TrimSpace(mode) == "" {
		mode = "local_proxy"
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"model":         model,
		"tokens_before": before,
		"tokens_after":  after,
		"latency_ms":    int(latencyMs + 0.5),
		"mode":          mode,
		"status":        status,
	})
	req, err := http.NewRequest(http.MethodPost, stringsTrimRightSlash(cfg.APIBaseURL)+"/api/v1/me/events", nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := newCLIHTTPClient(cfg)
	res, err := doCLIRequest(client, req, token, cfg.CLIHMACSecret, payload)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusPaymentRequired {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
		quotaExhaustedOnce.Do(func() {
			printQuotaExhaustedHint(fetchCLIChrome(cfg), body)
		})
	}
}

func stringsTrimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// revertSetupBaseURLs removes openai/local proxy Base URL keys that still point at Trim.
// Only deletes keys whose value matches the local Trim listen URL (no invent hosts).
func revertSetupBaseURLs(trimBaseURL string) ([]string, error) {
	trimBaseURL = strings.TrimSpace(trimBaseURL)
	if trimBaseURL == "" {
		return nil, nil
	}
	var reverted []string

	trySettings := func(product string, keys map[string]interface{}) {
		p, err := ideUserSettingsPath(product)
		if err != nil {
			return
		}
		if err := clearMatchingOpenAIBaseURL(p, trimBaseURL, keys); err == nil {
			reverted = append(reverted, p)
		}
	}
	trySettings("Cursor", cursorKeys())
	trySettings("Code", vscodeContinueKeys(trimBaseURL))
	trySettings("Windsurf", windsurfKeys())

	if p, err := continueConfigPath(); err == nil {
		if err := clearContinueTrimBase(p, trimBaseURL); err == nil {
			reverted = append(reverted, p)
		}
	}
	if p, err := zedSettingsPath(); err == nil {
		if err := clearZedTrimBase(p, trimBaseURL); err == nil {
			reverted = append(reverted, p)
		}
	}
	return reverted, nil
}

func clearMatchingOpenAIBaseURL(settingsPath, trimBaseURL string, keys map[string]interface{}) error {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return err
	}
	raw := map[string]interface{}{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
	}
	changed := false
	for k := range keys {
		if v, ok := raw[k].(string); ok && strings.TrimSpace(v) == trimBaseURL {
			delete(raw, k)
			changed = true
		}
	}
	if v, ok := raw["openai.baseUrl"].(string); ok && strings.TrimSpace(v) == trimBaseURL {
		delete(raw, "openai.baseUrl")
		changed = true
	}
	if !changed {
		return os.ErrNotExist
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(settingsPath, out, 0o600)
}

func clearContinueTrimBase(path, trimBaseURL string) error {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
		return clearContinueTrimBaseYAML(path, trimBaseURL)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw := map[string]interface{}{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
	}
	models, _ := raw["models"].([]interface{})
	changed := false
	outModels := make([]interface{}, 0, len(models))
	for _, m := range models {
		mm, ok := m.(map[string]interface{})
		if !ok {
			outModels = append(outModels, m)
			continue
		}
		apiBase, _ := mm["apiBase"].(string)
		if strings.TrimSpace(apiBase) == trimBaseURL {
			changed = true
			continue
		}
		outModels = append(outModels, mm)
	}
	if !changed {
		return os.ErrNotExist
	}
	raw["models"] = outModels
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o600)
}

func clearContinueTrimBaseYAML(path, trimBaseURL string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	if !strings.Contains(text, trimBaseURL) {
		return os.ErrNotExist
	}
	lines := strings.Split(text, "\n")
	changed := false
	for i, line := range lines {
		trimLeft := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimLeft, "apiBase:") && strings.Contains(line, trimBaseURL) {
			indent := line[:len(line)-len(trimLeft)]
			lines[i] = indent + "apiBase: "
			changed = true
		}
	}
	if !changed {
		return os.ErrNotExist
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0o600)
}

func clearZedTrimBase(path, trimBaseURL string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	raw := map[string]interface{}{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
	}
	lms, _ := raw["language_models"].(map[string]interface{})
	if lms == nil {
		return os.ErrNotExist
	}
	openai, _ := lms["openai"].(map[string]interface{})
	if openai == nil {
		return os.ErrNotExist
	}
	apiURL, _ := openai["api_url"].(string)
	if strings.TrimSpace(apiURL) != trimBaseURL {
		return os.ErrNotExist
	}
	delete(openai, "api_url")
	if len(openai) == 0 {
		delete(lms, "openai")
	} else {
		lms["openai"] = openai
	}
	raw["language_models"] = lms
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o600)
}

func init() {
	setupCmd.Flags().BoolVar(&setupTLS, "tls", false, "")
}
