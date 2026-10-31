package cli

import (
	"fmt"
	"strings"

	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/server/pkg/proxy"
)

func prefsChromeFrom(chrome cliChrome) deepopt.PrefsChrome {
	return deepopt.PrefsChrome{
		Unavailable:    chrome.ChromeUnavailable,
		EngineInvalid:  chrome.PrefsEngineInvalid,
		RateNonNeg:     chrome.PrefsRateNonNeg,
		TierRequired:   chrome.PrefsTierRequired,
		EngineRequired: chrome.PrefsEngineRequired,
		RateRequired:   chrome.PrefsRateRequired,
	}
}

func deepUXFrom(chrome cliChrome) deepopt.UXChrome {
	return deepopt.UXChrome{
		Unavailable:         chrome.ChromeUnavailable,
		BootstrapReqs:       chrome.DeepBootstrapReqs,
		BootstrapPip:        chrome.DeepBootstrapPip,
		BootstrapOk:         chrome.DeepBootstrapOk,
		BootstrapOkDetail:   chrome.DeepBootstrapOkDetail,
		RequirementsMissing: chrome.DeepRequirementsMissing,
		AutoInstall:         chrome.DeepAutoInstall,
		BinEnvMissingFmt:    chrome.DeepBinEnvMissingFmt,
		BinMissing:          chrome.DeepBinMissing,
		PyEnvMissingFmt:     chrome.DeepPyEnvMissingFmt,
		PyMissing:           chrome.DeepPyMissing,
		LLMMissingFmt:       chrome.DeepLLMMissingFmt,
		PipFailed:           chrome.DeepPipFailed,
		RateRequired:        chrome.DeepRateRequired,
		EngineRequired:      chrome.DeepEngineRequired,
		QuestionRequired:    chrome.DeepQuestionRequired,
		OptimizeFmt:         chrome.DeepOptimizeFmt,
		Timeout:             chrome.DeepTimeout,
		ParseFmt:            chrome.DeepParseFmt,
		// Live Deep auto-bootstrap uses the same colored chrome as trim start / compress --bootstrap.
		NotifyInfo:    cliout.Info,
		NotifySuccess: cliout.Success,
		NotifyMuted:   cliout.Muted,
	}
}

func deepFastOnly(optimized []byte, status string) proxy.DeepResult {
	n := (len(optimized) + 3) / 4
	return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: status}
}

// prepareLiveDeep wires Deep Mode for trim start from synced account preferences + billing dials.
// Fail-closed: deep tier requires engine, compression rate, live-Deep chrome, and synced billing policy (no invent).
func prepareLiveDeep(cfg config.Local, chrome cliChrome) (liveModeLabel string, deepOptimize func([]byte, string) (proxy.DeepResult, error), errDeepFmt string, err error) {
	prefs, err := deepopt.LoadPreferences()
	if err != nil {
		return "", nil, "", err
	}
	if prefs.DefaultTier != deepopt.TierDeep {
		return "", nil, "", nil
	}

	pc := prefsChromeFrom(chrome)
	if err := deepopt.RequireCompressPrefs(prefs, pc); err != nil {
		if chrome.ProxyDeepPrefsRequired != "" {
			return "", nil, "", fmt.Errorf("%s", chrome.ProxyDeepPrefsRequired)
		}
		return "", nil, "", err
	}
	if strings.TrimSpace(chrome.ProxyLiveDeepModeFmt) == "" || strings.TrimSpace(chrome.ProxyDeepCompactStub) == "" {
		if chrome.ProxyDeepChromeRequired != "" {
			return "", nil, "", fmt.Errorf("%s", chrome.ProxyDeepChromeRequired)
		}
		return "", nil, "", fmt.Errorf("%s", chrome.ChromeUnavailable)
	}
	if !deepopt.RequireLiveDeepPolicy(prefs) {
		if chrome.ProxyDeepRuntimeRequired != "" {
			return "", nil, "", fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
		}
		return "", nil, "", fmt.Errorf("%s", chrome.ChromeUnavailable)
	}
	if strings.TrimSpace(cfg.DeepDeviceMap) == "" {
		if chrome.ProxyDeepRuntimeRequired != "" {
			return "", nil, "", fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
		}
		return "", nil, "", fmt.Errorf("%s", chrome.ChromeUnavailable)
	}

	liveModeLabel = fmt.Sprintf(chrome.ProxyLiveDeepModeFmt, prefs.DeepEngine)
	errDeepFmt = chrome.ProxyDeepFailedFmt
	ux := deepUXFrom(chrome)
	chatChrome := deepopt.ChatDeepChrome{
		CompactStub: chrome.ProxyDeepCompactStub,
		Unavailable: chrome.ChromeUnavailable,
	}
	timeoutSec := cfg.DeepTimeoutSec
	deviceMap := cfg.DeepDeviceMap

	deepOptimize = func(optimized []byte, path string) (proxy.DeepResult, error) {
		if deepopt.SkipLiveDeepForRequestPath(path) {
			return deepFastOnly(optimized, proxy.DeepStatusPathSkip), nil
		}
		p, loadErr := deepopt.LoadPreferences()
		if loadErr != nil {
			return proxy.DeepResult{}, loadErr
		}
		if p.DefaultTier != deepopt.TierDeep {
			// Account turned Deep off mid-session - Fast-only for this request (explicit prefs, not invent).
			return deepFastOnly(optimized, proxy.DeepStatusSkippedOff), nil
		}
		if reqErr := deepopt.RequireCompressPrefs(p, pc); reqErr != nil {
			if chrome.ProxyDeepPrefsRequired != "" {
				return proxy.DeepResult{}, fmt.Errorf("%s", chrome.ProxyDeepPrefsRequired)
			}
			return proxy.DeepResult{}, reqErr
		}
		if !deepopt.RequireLiveDeepPolicy(p) {
			if chrome.ProxyDeepRuntimeRequired != "" {
				return proxy.DeepResult{}, fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
			}
			return proxy.DeepResult{}, fmt.Errorf("%s", chrome.ChromeUnavailable)
		}

		est := deepopt.EstimateLiveDeepInputTokens(optimized, path)
		if est == 0 {
			// Nothing compressible (empty/non-chat / IDE chrome-only) - Fast-only.
			return deepFastOnly(optimized, proxy.DeepStatusSkippedEmpty), nil
		}
		if est < deepopt.LiveDeepMinProseTokens {
			// Tiny turns (Cursor "hi") - skip LLMLingua warmup; Fast Mode is enough.
			return deepFastOnly(optimized, proxy.DeepStatusSkippedMin), nil
		}

		rt := deepopt.RuntimeFromPreferences(p, deviceMap)
		pl := strings.ToLower(path)
		var (
			out            []byte
			deepOrigin     int
			deepCompressed int
			deepErr        error
		)
		// Native Anthropic /v1/messages only (not count_tokens - handled above).
		if strings.Contains(pl, "/messages") && !strings.Contains(pl, "count_tokens") {
			out, deepOrigin, deepCompressed, deepErr = deepopt.CompressAnthropicMessagesJSON(optimized, p.DeepEngine, p.CompressionRate, "", ux, timeoutSec, chatChrome, rt)
		} else {
			// OpenAI-shape door: GPT, Gemini openai_compat, DeepSeek, Mistral, Cursor BYOK.
			out, deepOrigin, deepCompressed, deepErr = deepopt.CompressOpenAIChatJSON(optimized, p.DeepEngine, p.CompressionRate, "", ux, timeoutSec, chatChrome, rt)
		}
		if deepErr != nil {
			if p.LiveDeepOOMPolicy == deepopt.OOMSkip && deepopt.IsDeepOOMError(deepErr) {
				if chrome.ProxyDeepOOMSkipped != "" {
					fmt.Println(chrome.ProxyDeepOOMSkipped)
				}
				return deepFastOnly(optimized, proxy.DeepStatusOOMSkip), nil
			}
			// Live IDE resilience: Deep engine failure must not 502 any door
			// (Claude Code / Cursor / GPT / Gemini / DeepSeek). Fail closed to Fast-only.
			// Surface the reason (fail-closed chrome) so 0% Saved is not a silent black hole.
			if chrome.ProxyDeepFailedFmt != "" {
				display := deepErr
				if deepopt.IsDeepRuntimeConfigError(deepErr) && strings.TrimSpace(chrome.ProxyDeepRuntimeRequired) != "" {
					display = fmt.Errorf("%s", chrome.ProxyDeepRuntimeRequired)
				}
				fmt.Printf(chrome.ProxyDeepFailedFmt+"\n", display)
			}
			return deepFastOnly(optimized, proxy.DeepStatusFailClosedError), nil
		}
		// Fail-closed: never forward Deep expansion (agent chrome regression on any door).
		if deepopt.DeepExpanded(optimized, out, deepOrigin, deepCompressed) {
			if chrome.ProxyDeepFailedFmt != "" {
				detail := ""
				if fmtStr := strings.TrimSpace(chrome.ProxyDeepExpansionFailClosedFmt); fmtStr != "" {
					detail = fmt.Sprintf(fmtStr, deepOrigin, deepCompressed)
				}
				if detail != "" {
					fmt.Printf(chrome.ProxyDeepFailedFmt+"\n", detail)
				}
			}
			return proxy.DeepResult{
				Body:       optimized,
				Origin:     deepOrigin,
				Compressed: deepCompressed,
				Status:     proxy.DeepStatusFailClosedExpand,
			}, nil
		}
		return proxy.DeepResult{
			Body:       out,
			Origin:     deepOrigin,
			Compressed: deepCompressed,
			Status:     proxy.DeepStatusApplied,
		}, nil
	}

	// Detached worker: parent prints this after /health so the shell can return cleanly.
	if chrome.ProxyLiveDeepEnabledFmt != "" && !isTrimStartWorker() {
		fmt.Printf(chrome.ProxyLiveDeepEnabledFmt+"\n", prefs.DeepEngine, prefs.DeepV2Backbone, prefs.CompressionRate, prefs.LiveDeepOOMPolicy)
	}

	if prefs.LiveDeepWarmupOnStart {
		go func() {
			rt := deepopt.RuntimeFromPreferences(prefs, deviceMap)
			req := deepopt.Request{
				Text:   "warmup",
				Rate:   prefs.CompressionRate,
				Engine: string(prefs.DeepEngine),
			}
			if err := rt.ApplyEngine(&req, prefs.DeepEngine); err != nil {
				return
			}
			_, _ = deepopt.Compress(req, ux, timeoutSec)
		}()
	}

	return liveModeLabel, deepOptimize, errDeepFmt, nil
}
