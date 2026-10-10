package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// liveProxyStats is the subset of GET /v1/stats used for clear CLI savings copy.
type liveProxyStats struct {
	LastBeforeTokens    int                  `json:"last_before_tokens"`
	LastAfterTokens     int                  `json:"last_after_tokens"`
	LastDoor            string               `json:"last_door"`
	LastDeepStatus      string               `json:"last_deep_status"`
	LastDeepStageBefore int                  `json:"last_deep_stage_before_tokens"`
	LastDeepStageAfter  int                  `json:"last_deep_stage_after_tokens"`
	LastDeepStageSaved  float64              `json:"last_deep_stage_saved_percent"`
	Chrome              liveProxyStatsChrome `json:"chrome"`
}

type liveProxyStatsChrome struct {
	DeepStatusLabels       map[string]string `json:"deep_status_labels"`
	SavingsDetailChromeTip string            `json:"savings_detail_chrome_tip"`
	LastRequestValueFmt    string            `json:"last_request_value_fmt"`
}

// printLiveSavingsClarity prints Deep status / stage lines so users are not stuck reading raw JSON.
// Fail-closed: empty chrome formats → skip those lines (no invent English).
func printLiveSavingsClarity(chrome cliChrome, port string, body []byte) {
	var st liveProxyStats
	if err := json.Unmarshal(body, &st); err != nil {
		return
	}
	statusCode := strings.TrimSpace(st.LastDeepStatus)
	statusLabel := statusCode
	if st.Chrome.DeepStatusLabels != nil {
		if mapped := strings.TrimSpace(st.Chrome.DeepStatusLabels[statusCode]); mapped != "" {
			statusLabel = mapped
		}
	}
	if chrome.StatsDeepStatusFmt != "" && statusLabel != "" {
		fmt.Printf(chrome.StatsDeepStatusFmt+"\n", statusLabel)
	}
	if chrome.StatsDeepStageFmt != "" && (st.LastDeepStageBefore > 0 || st.LastDeepStageAfter > 0 || statusCode != "") {
		stagePair := formatTokenPair(st.Chrome.LastRequestValueFmt, st.LastDeepStageBefore, st.LastDeepStageAfter)
		fmt.Printf(chrome.StatsDeepStageFmt+"\n", stagePair, st.LastDeepStageSaved)
	}
	if chrome.StatsLastRequestFmt != "" && (st.LastBeforeTokens > 0 || st.LastAfterTokens > 0) {
		wirePair := formatTokenPair(st.Chrome.LastRequestValueFmt, st.LastBeforeTokens, st.LastAfterTokens)
		wirePct := 0.0
		if st.LastBeforeTokens > 0 {
			wirePct = float64(st.LastBeforeTokens-st.LastAfterTokens) / float64(st.LastBeforeTokens) * 100
		}
		fmt.Printf(chrome.StatsLastRequestFmt+"\n", wirePair, wirePct)
	}
	if chrome.StatsDoorFmt != "" && strings.TrimSpace(st.LastDoor) != "" {
		fmt.Printf(chrome.StatsDoorFmt+"\n", strings.TrimSpace(st.LastDoor))
	}
	if tip := strings.TrimSpace(chrome.StatsDashboardTipFmt); tip != "" {
		// Tip is Println + {port} only (not Printf). Normalize accidental %% from older site_messages.
		tip = strings.ReplaceAll(tip, "{port}", strings.TrimSpace(port))
		tip = strings.ReplaceAll(tip, "%%", "%")
		fmt.Println(tip)
	}
}

func formatTokenPair(valueFmt string, before, after int) string {
	valueFmt = strings.TrimSpace(valueFmt)
	if valueFmt != "" && strings.Count(valueFmt, "%d") >= 2 {
		return fmt.Sprintf(valueFmt, before, after)
	}
	return fmt.Sprintf("%d -> %d", before, after)
}
