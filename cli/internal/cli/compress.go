package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/cli/internal/projectconfig"
	"github.com/usetrim/trim/server/pkg/proxy"
	"github.com/usetrim/trim/server/pkg/trimmer"
)

var errTreesitterRequired = errors.New("treesitter_required")

var (
	compressMode      string
	compressDeepShort bool
	compressEngine    string
	compressQuestion  string
	compressTargetTok int
	compressBootstrap bool
	compressOutFile   string
)

var compressCmd = &cobra.Command{
	Use:   "compress [file]",
	Short: "",
	Long:  "",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadLocal()
		if err != nil {
			return err
		}
		chrome := fetchCLIChrome(cfg)
		ux := deepopt.UXChrome{
			Unavailable:         chrome.ChromeUnavailable,
			BootstrapReqs:       chrome.DeepBootstrapReqs,
			BootstrapPip:        chrome.DeepBootstrapPip,
			RequirementsMissing: chrome.DeepRequirementsMissing,
			AutoInstall:         chrome.DeepAutoInstall,
			BinEnvMissingFmt:    chrome.DeepBinEnvMissingFmt,
			BinMissing:          chrome.DeepBinMissing,
			PyEnvMissingFmt:     chrome.DeepPyEnvMissingFmt,
			PyMissing:           chrome.DeepPyMissing,
			LLMMissingFmt:       chrome.DeepLLMMissingFmt,
			PipFailed:           chrome.DeepPipFailed,
			TargetRequired:      chrome.DeepTargetRequired,
			EngineRequired:      chrome.DeepEngineRequired,
			QuestionRequired:    chrome.DeepQuestionRequired,
			OptimizeFmt:         chrome.DeepOptimizeFmt,
			Timeout:             chrome.DeepTimeout,
			ParseFmt:            chrome.DeepParseFmt,
		}
		prefsChrome := deepopt.PrefsChrome{
			Unavailable:    chrome.ChromeUnavailable,
			EngineInvalid:  chrome.PrefsEngineInvalid,
			TargetNonNeg:   chrome.PrefsTargetNonNeg,
			TierRequired:   chrome.PrefsTierRequired,
			EngineRequired: chrome.PrefsEngineRequired,
			TargetRequired: chrome.PrefsTargetRequired,
		}
		if compressBootstrap {
			return deepopt.Bootstrap(ux)
		}
		if len(args) == 0 {
			if chrome.CompressFileRequired != "" {
				return fmt.Errorf("%s", chrome.CompressFileRequired)
			}
			return chrome.fail("")
		}

		prefs, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}

		tier := deepopt.NormalizeTier(compressMode)
		if compressDeepShort {
			tier = deepopt.TierDeep
		} else if !cmd.Flags().Changed("mode") {
			if prefs.DefaultTier != "" {
				tier = prefs.DefaultTier
			} else if chrome.CompressModeRequired != "" {
				return fmt.Errorf("%s", chrome.CompressModeRequired)
			} else {
				return chrome.fail("")
			}
		} else if tier == "" {
			if chrome.CompressModeInvalid != "" {
				return fmt.Errorf("%s", chrome.CompressModeInvalid)
			}
			return chrome.fail("")
		}
		engine := deepopt.NormalizeEngine(compressEngine)
		if !cmd.Flags().Changed("engine") {
			engine = prefs.DeepEngine
		} else if compressEngine != "" && engine == "" {
			if chrome.PrefsEngineInvalid != "" {
				return fmt.Errorf("%s", chrome.PrefsEngineInvalid)
			}
			return chrome.fail("")
		}
		target := compressTargetTok
		if target <= 0 {
			target = prefs.TargetToken
		}

		raw, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		text := string(raw)

		if tier == deepopt.TierFast {
			out, before, after, err := compressFastAST(args[0], text)
			if errors.Is(err, errTreesitterRequired) {
				if chrome.TreesitterRequired != "" {
					return fmt.Errorf("%s", chrome.TreesitterRequired)
				}
				return chrome.fail("")
			}
			if err != nil {
				return err
			}
			return writeCompressResult(out, before, after, "fast", string(engine), chrome)
		}

		if engine == "" || target <= 0 {
			if chrome.CompressDeepPrefsRequired != "" {
				return fmt.Errorf("%s", chrome.CompressDeepPrefsRequired)
			}
			if err := deepopt.RequireCompressPrefs(prefs, prefsChrome); err != nil {
				return err
			}
			return chrome.fail("")
		}
		if !deepopt.RequireLiveDeepPolicy(prefs) {
			if chrome.ProxyDeepRuntimeRequired != "" {
				return fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
			}
			return chrome.fail("")
		}
		if strings.TrimSpace(cfg.DeepDeviceMap) == "" {
			if chrome.ProxyDeepRuntimeRequired != "" {
				return fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
			}
			return chrome.fail("")
		}

		// Chat/request JSON (Claude Messages / OpenAI-compat GPT/Gemini/DeepSeek/Mistral):
		// never smash the whole body into LLMLingua - last-user-only Deep, same as live proxy.
		if outJSON, origin, compressed, ok, chatErr := compressDeepChatJSONIfNeeded(raw, engine, target, compressQuestion, ux, cfg, prefs, chrome); ok {
			if chatErr != nil {
				return chatErr
			}
			if chrome.CompressDeepResultFmt != "" {
				fmt.Printf(
					chrome.CompressDeepResultFmt+"\n",
					string(engine), origin, compressed, deepopt.FormatSavingPercent(origin, compressed),
				)
			}
			return writeCompressResult(string(outJSON), origin, compressed, "deep", string(engine), chrome)
		}
		// Fail-closed: Responses / chat / tools JSON that we could not structure-preserve
		// must NOT fall through to whole-payload LLMLingua (exact Claude garble root cause).
		if looksLikeChatRequestJSON(raw) {
			n := (len(raw) + 3) / 4
			if chrome.CompressDeepResultFmt != "" {
				fmt.Printf(
					chrome.CompressDeepResultFmt+"\n",
					string(engine), n, n, deepopt.FormatSavingPercent(n, n),
				)
			}
			return writeCompressResult(string(raw), n, n, "deep", string(engine), chrome)
		}

		req := deepopt.Request{
			Text:        text,
			Question:    compressQuestion,
			TargetToken: target,
			Engine:      string(engine),
		}
		if err := deepopt.RuntimeFromPreferences(prefs, cfg.DeepDeviceMap).ApplyEngine(&req, engine); err != nil {
			return err
		}

		res, err := deepopt.Compress(req, ux, cfg.DeepTimeoutSec)
		if err != nil {
			return err
		}
		if chrome.CompressDeepResultFmt != "" {
			fmt.Printf(
				chrome.CompressDeepResultFmt+"\n",
				res.Engine, res.OriginTokens, res.CompressedTokens, res.SavingRate,
			)
		}
		return writeCompressResult(res.CompressedPrompt, res.OriginTokens, res.CompressedTokens, "deep", res.Engine, chrome)
	},
}

// shouldNormalizeChatJSONForDeep is true when the payload is OpenAI-compat / Responses-shaped
// and safe to run Chat tool/content normalizers. False for Anthropic Messages native bodies
// so CLI Deep never smashes Claude Code chrome (path gate mirrors live /v1/messages).
func shouldNormalizeChatJSONForDeep(raw []byte) bool {
	if !gjson.ValidBytes(raw) {
		return false
	}
	if gjson.GetBytes(raw, "input").Exists() {
		return true // Responses → need Chat migrate
	}
	model := strings.ToLower(gjson.GetBytes(raw, "model").String())
	// Anthropic Messages chrome: never Chat-wrap tools (exact smash class on Claude Code).
	// Includes aliases without "claude" in the id, MCP/container, Anthropic extended
	// thinking (budget_tokens), and flat {name,input_schema} client tools.
	// IMPORTANT: DeepSeek Chat Completions also sends top-level thinking:{type:enabled|disabled}
	// WITHOUT budget_tokens (official DeepSeek docs). Do NOT treat bare thinking as Anthropic
	// or we skip Chat normalize for DeepSeek (flat tools / input_text migrate break).
	if strings.Contains(model, "claude") || strings.Contains(model, "anthropic") ||
		gjson.GetBytes(raw, "system").Exists() ||
		gjson.GetBytes(raw, "mcp_servers").Exists() ||
		gjson.GetBytes(raw, "container").Exists() ||
		gjson.GetBytes(raw, "thinking.budget_tokens").Exists() {
		return false
	}
	tools := gjson.GetBytes(raw, "tools")
	if tools.IsArray() {
		for _, t := range tools.Array() {
			typ := strings.ToLower(strings.TrimSpace(t.Get("type").String()))
			if proxyLooksLikeAnthropicDatedToolType(typ) {
				return false
			}
			// Flat Anthropic client tool: name + input_schema, no Chat function wrapper.
			if !t.Get("function").Exists() && t.Get("input_schema").Exists() &&
				strings.TrimSpace(t.Get("name").String()) != "" {
				return false
			}
		}
	}
	return gjson.GetBytes(raw, "messages").Exists()
}

func proxyLooksLikeAnthropicDatedToolType(typ string) bool {
	if typ == "" || typ == "function" || typ == "custom" {
		return false
	}
	for i := 0; i+9 <= len(typ); i++ {
		if typ[i] != '_' {
			continue
		}
		digits := typ[i+1:]
		if len(digits) < 8 {
			continue
		}
		ok := true
		for _, c := range digits[:8] {
			if c < '0' || c > '9' {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// compressDeepChatJSONIfNeeded runs structure-preserving Deep when the file is a chat
// Completions / Messages / Responses body. Plain text / source files fall through (ok=false).
func compressDeepChatJSONIfNeeded(raw []byte, engine deepopt.Engine, target int, question string, ux deepopt.UXChrome, cfg config.Local, prefs deepopt.Preferences, chrome cliChrome) (out []byte, origin, compressed int, ok bool, err error) {
	if !gjson.ValidBytes(raw) {
		return nil, 0, 0, false, nil
	}
	// Responses→Chat + content-part migrate for OpenAI-compat shapes only.
	// Anthropic Messages (system / claude model / input_schema tools): never wrap tools
	// into Chat function shape (exact smash class on Claude Code payloads).
	if shouldNormalizeChatJSONForDeep(raw) {
		normalized := proxy.NormalizeOpenAICompatRequestBody(raw)
		if gjson.GetBytes(normalized, "messages").IsArray() {
			raw = normalized
		}
	}
	if !gjson.GetBytes(raw, "messages").IsArray() {
		return nil, 0, 0, false, nil
	}
	stub := strings.TrimSpace(chrome.ProxyDeepCompactStub)
	if stub == "" {
		if chrome.ProxyDeepChromeRequired != "" {
			return nil, 0, 0, true, fmt.Errorf("%s", chrome.ProxyDeepChromeRequired)
		}
		return nil, 0, 0, true, fmt.Errorf("%s", chrome.ChromeUnavailable)
	}
	chatChrome := deepopt.ChatDeepChrome{
		CompactStub: stub,
		Unavailable: chrome.ChromeUnavailable,
	}
	rt := deepopt.RuntimeFromPreferences(prefs, cfg.DeepDeviceMap)
	timeoutSec := cfg.DeepTimeoutSec
	// Anthropic Messages often carry top-level system; Claude Code may only use messages.
	// Both compressors share last-user-only Deep (identical freeze rules for all doors).
	model := strings.ToLower(gjson.GetBytes(raw, "model").String())
	if gjson.GetBytes(raw, "system").Exists() || strings.Contains(model, "claude") {
		out, origin, compressed, err = deepopt.CompressAnthropicMessagesJSON(raw, engine, target, question, ux, timeoutSec, chatChrome, rt)
	} else {
		out, origin, compressed, err = deepopt.CompressOpenAIChatJSON(raw, engine, target, question, ux, timeoutSec, chatChrome, rt)
	}
	if err != nil {
		return nil, 0, 0, true, err
	}
	if deepopt.DeepExpanded(raw, out, origin, compressed) {
		return raw, origin, origin, true, nil
	}
	return out, origin, compressed, true, nil
}

// looksLikeChatRequestJSON detects agent/chat wire payloads that must never be
// whole-string Deep-compressed (Anthropic Messages, OpenAI Chat, Responses input,
// Gemini native contents[], DeepSeek thinking chrome). Exact smash-class guard.
func looksLikeChatRequestJSON(raw []byte) bool {
	if !gjson.ValidBytes(raw) {
		return false
	}
	if gjson.GetBytes(raw, "messages").Exists() {
		return true
	}
	if gjson.GetBytes(raw, "input").Exists() {
		return true
	}
	// Gemini native GenerateContent shape (not OpenAI-compat) - fail-closed, no smash.
	if gjson.GetBytes(raw, "contents").Exists() {
		return true
	}
	if gjson.GetBytes(raw, "model").Exists() &&
		(gjson.GetBytes(raw, "tools").Exists() || gjson.GetBytes(raw, "system").Exists() ||
			gjson.GetBytes(raw, "instructions").Exists() ||
			gjson.GetBytes(raw, "thinking").Exists() ||
			gjson.GetBytes(raw, "reasoning_effort").Exists() ||
			gjson.GetBytes(raw, "tool_choice").Exists() ||
			gjson.GetBytes(raw, "mcp_servers").Exists() ||
			gjson.GetBytes(raw, "container").Exists() ||
			gjson.GetBytes(raw, "cache_control").Exists()) {
		return true
	}
	return false
}

// compressFastAST runs the same Structural AST / log pipeline as the local proxy.
// Intensity comes from .trimrc when set. Empty mode: structural heuristic only
// (no invent of balanced AST intensity).
func compressFastAST(filePath, text string) (string, int, int, error) {
	startDir := filepath.Dir(filePath)
	proj, err := projectconfig.Load(startDir)
	if err != nil {
		return "", 0, 0, err
	}

	mode := trimmer.NormalizeMode(proj.Mode)
	if mode == "" {
		out := deepopt.FastHeuristic(text)
		before := (len(text) + 3) / 4
		after := (len(out) + 3) / 4
		return out, before, after, nil
	}
	if len(proj.CustomQueries) > 0 && !trimmer.TreeSitterAvailable() {
		return "", 0, 0, errTreesitterRequired
	}
	resolved := projectconfig.Config{
		Mode:                 mode,
		CustomMinLines:       proj.CustomMinLines,
		CustomLogsOnly:       proj.CustomLogsOnly,
		ActiveFileProtection: proj.ActiveFileProtection,
		MaxLogBytes:          proj.MaxLogBytes,
		LogCompactMinBytes:   proj.LogCompactMinBytes,
		LogCompactMaxLines:   proj.LogCompactMaxLines,
		LogNoiseSubstrings:   proj.LogNoiseSubstrings,
		IgnorePaths:          proj.IgnorePaths,
		NeverTrimPaths:       proj.NeverTrimPaths,
		CustomQueries:        proj.CustomQueries,
	}
	if !proj.ActiveFileProtectionSet {
		resolved.ActiveFileProtection = chromeProxyActiveFileProtection(loadCLIChrome())
	}

	lang := trimmer.LangFromFilename(filePath)
	if lang == "" {
		out := deepopt.FastHeuristic(text)
		before := (len(text) + 3) / 4
		after := (len(out) + 3) / 4
		return out, before, after, nil
	}

	header := lang + ":" + filepath.ToSlash(filePath)
	fenced := "```" + header + "\n" + text + "\n```"
	out, before, after := trimmer.ProcessPromptTextOpts(fenced, filepath.Base(filePath), trimmer.ProcessOptions{
		NeverTrim:            proj.MustNeverTrim,
		MaxLogBytes:          proj.MaxLogBytes,
		LogCompactMinBytes:   proj.LogCompactMinBytes,
		LogCompactMaxLines:   proj.LogCompactMaxLines,
		LogNoiseSubstrings:   proj.LogNoiseSubstrings,
		ActiveFileProtection: false,
		Mode:                 resolved.Mode,
		MinLines:             resolved.MinLinesForMode(),
		DisableSkeletonize:   !resolved.SkeletonizeEnabled(),
		AlwaysCompactLogs:    resolved.AlwaysCompactLogs(),
		CustomQueries:        resolved.CustomQueries,
	})
	unwrapped := trimmer.UnwrapSingleCodeFence(out)
	if unwrapped != "" {
		out = unwrapped
	}
	return out, before, after, nil
}

func writeCompressResult(out string, before, after int, tier, engine string, chrome cliChrome) error {
	if compressOutFile != "" {
		if err := os.WriteFile(compressOutFile, []byte(out), 0o600); err != nil {
			return err
		}
		if chrome.CompressWroteFmt != "" {
			fmt.Printf(chrome.CompressWroteFmt+"\n", compressOutFile, tier, engine, before, after)
		}
		return nil
	}
	fmt.Print(out)
	if !strings.HasSuffix(out, "\n") {
		fmt.Println()
	}
	return nil
}

func init() {
	compressCmd.Flags().StringVar(&compressMode, "mode", "", "")
	compressCmd.Flags().BoolVarP(&compressDeepShort, "deep", "d", false, "")
	compressCmd.Flags().StringVar(&compressEngine, "engine", "", "")
	compressCmd.Flags().StringVar(&compressQuestion, "question", "", "")
	compressCmd.Flags().IntVar(&compressTargetTok, "target-token", 0, "")
	compressCmd.Flags().BoolVar(&compressBootstrap, "bootstrap", false, "")
	compressCmd.Flags().StringVarP(&compressOutFile, "out", "o", "", "")
}
