package cli

import (
	"fmt"
	"strings"

	"github.com/usetrim/trim/cli/internal/config"
	"github.com/usetrim/trim/cli/internal/deepopt"
	"github.com/usetrim/trim/server/pkg/proxy"
)

func prefsChromeFrom(chrome cliChrome) deepopt.PrefsChrome {
	return deepopt.PrefsChrome{
		Unavailable:    chrome.ChromeUnavailable,
		EngineInvalid:  chrome.PrefsEngineInvalid,
		TargetNonNeg:   chrome.PrefsTargetNonNeg,
		TierRequired:   chrome.PrefsTierRequired,
		EngineRequired: chrome.PrefsEngineRequired,
		TargetRequired: chrome.PrefsTargetRequired,
	}
}

func deepUXFrom(chrome cliChrome) deepopt.UXChrome {
	return deepopt.UXChrome{
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
}

func deepFastOnly(optimized []byte, status string) proxy.DeepResult {
	n := (len(optimized) + 3) / 4
	return proxy.DeepResult{Body: optimized, Origin: n, Compressed: n, Status: status}
}

// prepareLiveDeep wires Deep Mode for trim start from synced account preferences + billing dials.
// Fail-closed: deep tier requires engine, target, live-Deep chrome, and synced billing policy (no invent).
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
			// Nothing compressible (empty/non-chat / chrome-only) - Fast-only, no Deep spam.
			return deepFastOnly(optimized, proxy.DeepStatusSkippedEmpty), nil
		}
		if p.LiveDeepMinInputTokens > 0 && est < p.LiveDeepMinInputTokens {
			if chrome.ProxyDeepSkippedMinFmt != "" {
				fmt.Printf(chrome.ProxyDeepSkippedMinFmt+"\n", est, p.LiveDeepMinInputTokens)
			}
			return deepFastOnly(optimized, proxy.DeepStatusSkippedMin), nil
		}
		if p.LiveDeepSkipOnStream && deepopt.RequestWantsStream(optimized) {
			if chrome.ProxyDeepSkippedStream != "" {
				fmt.Println(chrome.ProxyDeepSkippedStream)
			}
			return deepFastOnly(optimized, proxy.DeepStatusSkippedStream), nil
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
			out, deepOrigin, deepCompressed, deepErr = deepopt.CompressAnthropicMessagesJSON(optimized, p.DeepEngine, p.TargetToken, "", ux, timeoutSec, chatChrome, rt)
		} else {
			// OpenAI-shape door: GPT, Gemini openai_compat, DeepSeek, Mistral, Cursor BYOK.
			out, deepOrigin, deepCompressed, deepErr = deepopt.CompressOpenAIChatJSON(optimized, p.DeepEngine, p.TargetToken, "", ux, timeoutSec, chatChrome, rt)
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
				fmt.Printf(chrome.ProxyDeepFailedFmt+"\n", deepErr)
			}
			return deepFastOnly(optimized, proxy.DeepStatusFailClosedError), nil
		}
		// Fail-closed: never forward Deep expansion (agent chrome regression on any door).
		if deepopt.DeepExpanded(optimized, out, deepOrigin, deepCompressed) {
			if chrome.ProxyDeepFailedFmt != "" {
				fmt.Printf(chrome.ProxyDeepFailedFmt+"\n", fmt.Errorf("expansion fail-closed (%d->%d stage tokens; kept Fast)", deepOrigin, deepCompressed))
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

	if chrome.ProxyLiveDeepEnabledFmt != "" {
		fmt.Printf(chrome.ProxyLiveDeepEnabledFmt+"\n", prefs.DeepEngine, prefs.TargetToken, prefs.LiveDeepMinInputTokens, prefs.LiveDeepOOMPolicy, prefs.LiveDeepSkipOnStream)
	}

	if prefs.LiveDeepWarmupOnStart {
		go func() {
			rt := deepopt.RuntimeFromPreferences(prefs, deviceMap)
			req := deepopt.Request{
				Text:        "warmup",
				TargetToken: prefs.TargetToken,
				Engine:      string(prefs.DeepEngine),
			}
			if err := rt.ApplyEngine(&req, prefs.DeepEngine); err != nil {
				return
			}
			_, _ = deepopt.Compress(req, ux, timeoutSec)
		}()
	}

	return liveModeLabel, deepOptimize, errDeepFmt, nil
}
