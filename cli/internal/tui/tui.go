package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/usetrim/trim/cli/internal/clihttp"
	"github.com/usetrim/trim/cli/internal/metrics"
	"github.com/usetrim/trim/server/pkg/localchrome"
)

type statsPayload struct {
	Requests            int64              `json:"requests"`
	TokensBefore        int64              `json:"tokens_before"`
	TokensAfter         int64              `json:"tokens_after"`
	SavedUSD            float64            `json:"saved_usd_est"`
	LastLatency         float64            `json:"last_latency_ms"`
	LastDoor            string             `json:"last_door"`
	LastDeepStatus      string             `json:"last_deep_status"`
	LastDeepStageBefore int                `json:"last_deep_stage_before_tokens"`
	LastDeepStageAfter  int                `json:"last_deep_stage_after_tokens"`
	LastDeepStageSaved  float64            `json:"last_deep_stage_saved_percent"`
	Chrome              localchrome.Chrome `json:"chrome"`
}

type cloudQuota struct {
	PlanTier              string `json:"plan_tier"`
	MonthlyCreditLimit    int    `json:"monthly_credit_limit"`
	MonthlyCreditUsed     int    `json:"monthly_credit_used"`
	PurchasedTopupCredits int    `json:"purchased_topup_credits"`
	Remaining             int    `json:"remaining"`
	Unlimited             bool   `json:"unlimited"`
	UnlimitedLabel        string `json:"unlimited_label"`
	Code                  string `json:"code"`
	ExhaustedTitle        string `json:"exhausted_title"`
	ExhaustedBody         string `json:"exhausted_body"`
	TierUpgrade           string `json:"tier_upgrade"`
}

type model struct {
	port              string
	apiBase           string
	token             string
	hmacSecret        string
	cliVersion        string
	savingsUsdPerMTok float64
	httpTimeoutSec    int
	refreshSec        int
	seriesDays        int
	errorMaxChars     int
	stats             statsPayload
	cloud             *cloudQuota
	cloudErr          string
	chrome            localchrome.Chrome
	series            []metrics.DayPoint
	errMsg            string
	width             int
	height            int
	lastFetch         time.Time
}

type tickMsg time.Time
type statsMsg struct {
	stats    statsPayload
	series   []metrics.DayPoint
	cloud    *cloudQuota
	cloudErr string
	err      error
}

func New(port, apiBase, token, hmacSecret, cliVersion string, savingsUsdPerMTok float64, httpTimeoutSec, refreshSec, seriesDays, errorMaxChars int, chrome localchrome.Chrome) (model, error) {
	if httpTimeoutSec < 1 {
		return model{}, fmt.Errorf("CONFIG_ENV_TRIM_CLI_HTTP_TIMEOUT_SEC_INVALID")
	}
	if refreshSec < 1 {
		return model{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_REFRESH_SEC_INVALID")
	}
	if seriesDays < 1 {
		return model{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_SERIES_DAYS_INVALID")
	}
	if errorMaxChars < 1 {
		return model{}, fmt.Errorf("CONFIG_ENV_TRIM_TUI_ERROR_MAX_CHARS_INVALID")
	}
	return model{
		port:              port,
		apiBase:           strings.TrimRight(strings.TrimSpace(apiBase), "/"),
		token:             strings.TrimSpace(token),
		hmacSecret:        strings.TrimSpace(hmacSecret),
		cliVersion:        strings.TrimSpace(cliVersion),
		savingsUsdPerMTok: savingsUsdPerMTok,
		httpTimeoutSec:    httpTimeoutSec,
		refreshSec:        refreshSec,
		seriesDays:        seriesDays,
		errorMaxChars:     errorMaxChars,
		chrome:            chrome,
	}, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchAll(m.port, m.apiBase, m.token, m.hmacSecret, m.cliVersion, m.savingsUsdPerMTok, m.httpTimeoutSec, m.seriesDays, m.chrome), m.tick())
}

func (m model) tick() tea.Cmd {
	return tea.Tick(time.Duration(m.refreshSec)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchCloudQuota(apiBase, token, hmacSecret, cliVersion string, httpTimeoutSec int) (*cloudQuota, string) {
	if apiBase == "" || token == "" {
		return nil, ""
	}
	url := apiBase + "/api/v1/me/quota"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err.Error()
	}
	if err := clihttp.AttachAuthHeaders(req, token, hmacSecret, cliVersion, "cli", nil); err != nil {
		return nil, err.Error()
	}
	req.Header.Set("Accept", "application/json")
	if httpTimeoutSec < 1 {
		return nil, "CONFIG_ENV_TRIM_CLI_HTTP_TIMEOUT_SEC_INVALID"
	}
	client := &http.Client{Timeout: time.Duration(httpTimeoutSec) * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 65536))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Sprintf("HTTP %d", res.StatusCode)
	}
	var q cloudQuota
	if err := json.Unmarshal(body, &q); err != nil {
		return nil, err.Error()
	}
	return &q, ""
}

func fetchAll(port, apiBase, token, hmacSecret, cliVersion string, savingsUsdPerMTok float64, httpTimeoutSec, seriesDays int, chrome localchrome.Chrome) tea.Cmd {
	return func() tea.Msg {
		cloud, cloudErr := fetchCloudQuota(apiBase, token, hmacSecret, cliVersion, httpTimeoutSec)

		var series []metrics.DayPoint
		if store, storeErr := metrics.OpenDefault(); storeErr == nil && store != nil {
			if pts, serr := store.Series(seriesDays, savingsUsdPerMTok); serr == nil {
				series = pts
			}
		}

		urlFmt := strings.TrimSpace(chrome.StatsURLFmt)
		if urlFmt == "" || !strings.Contains(urlFmt, "{port}") {
			return statsMsg{
				err:      fmt.Errorf("%s", "LOCAL_STATS_URL_FMT"),
				series:   series,
				cloud:    cloud,
				cloudErr: cloudErr,
			}
		}
		url := strings.ReplaceAll(urlFmt, "{port}", port)
		if httpTimeoutSec < 1 {
			msg := strings.TrimSpace(chrome.TUIHttpTimeoutUnset)
			if msg == "" {
				msg = "LOCAL_TUI_HTTP_TIMEOUT_UNSET"
			}
			return statsMsg{err: fmt.Errorf("%s", msg), series: series, cloud: cloud, cloudErr: cloudErr}
		}
		client := &http.Client{Timeout: time.Duration(httpTimeoutSec) * time.Second}
		res, err := client.Get(url)
		if err == nil {
			defer res.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(res.Body, 65536))
			var s statsPayload
			if err := json.Unmarshal(body, &s); err == nil {
				return statsMsg{stats: s, series: series, cloud: cloud, cloudErr: cloudErr}
			}
		}
		store, storeErr := metrics.OpenDefault()
		if storeErr != nil || store == nil {
			return statsMsg{err: err, cloud: cloud, cloudErr: cloudErr, series: series}
		}
		sum, sumErr := store.TodaySummary(savingsUsdPerMTok)
		if sumErr != nil {
			return statsMsg{err: sumErr, cloud: cloud, cloudErr: cloudErr, series: series}
		}
		return statsMsg{
			stats: statsPayload{
				Requests:     sum.Requests,
				TokensBefore: sum.TokensBefore,
				TokensAfter:  sum.TokensAfter,
				SavedUSD:     sum.SavedUSD,
				LastLatency:  sum.LastLatency,
				Chrome:       localchrome.Chrome{},
			},
			series:   series,
			cloud:    cloud,
			cloudErr: cloudErr,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, fetchAll(m.port, m.apiBase, m.token, m.hmacSecret, m.cliVersion, m.savingsUsdPerMTok, m.httpTimeoutSec, m.seriesDays, m.chrome)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.lastFetch = time.Time(msg)
		return m, tea.Batch(fetchAll(m.port, m.apiBase, m.token, m.hmacSecret, m.cliVersion, m.savingsUsdPerMTok, m.httpTimeoutSec, m.seriesDays, m.chrome), m.tick())
	case statsMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.errMsg = ""
			m.stats = msg.stats
			if msg.stats.Chrome.TUITitle != "" {
				m.chrome = msg.stats.Chrome
			}
		}
		m.cloud = msg.cloud
		m.cloudErr = msg.cloudErr
		if msg.series != nil {
			m.series = msg.series
		}
	}
	return m, nil
}

func (m model) View() string {
	ch := m.chrome
	daysStr := fmt.Sprintf("%d", m.seriesDays)
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(ch.TUITitle)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(28)

	savedPct := 0.0
	if m.stats.TokensBefore > 0 {
		savedPct = (1.0 - float64(m.stats.TokensAfter)/float64(m.stats.TokensBefore)) * 100
	}

	card1 := box.Render(fmt.Sprintf("%s\n\n%d", muted(ch.TUILabelRequests), m.stats.Requests))
	card2 := box.Render(fmt.Sprintf("%s\n\n%d -> %d", muted(ch.TUILabelTokens), m.stats.TokensBefore, m.stats.TokensAfter))
	card3 := box.Render(fmt.Sprintf("%s\n\n%.1f%%", muted(ch.TUILabelReduction), savedPct))
	estFmt := ch.TUIEstSavedFmt
	latFmt := ch.TUILatencyFmt
	var card4, card5 string
	if estFmt != "" {
		card4 = box.Render(fmt.Sprintf("%s\n\n"+estFmt, muted(ch.TUILabelEstSaved), m.stats.SavedUSD))
	} else {
		card4 = box.Render(fmt.Sprintf("%s\n\n", muted(ch.TUILabelEstSaved)))
	}
	if latFmt != "" {
		card5 = box.Render(fmt.Sprintf("%s\n\n"+latFmt, muted(ch.TUILabelLastLatency), m.stats.LastLatency))
	} else {
		card5 = box.Render(fmt.Sprintf("%s\n\n", muted(ch.TUILabelLastLatency)))
	}
	doorLabel := strings.TrimSpace(ch.TUILabelLastDoor)
	doorVal := strings.TrimSpace(m.stats.LastDoor)
	var card6 string
	if doorLabel != "" {
		card6 = box.Render(fmt.Sprintf("%s\n\n%s", muted(doorLabel), doorVal))
	}
	deepStatusLabel := strings.TrimSpace(ch.LabelDeepStatus)
	deepStatusVal := strings.TrimSpace(m.stats.LastDeepStatus)
	if ch.DeepStatusLabels != nil {
		if mapped := strings.TrimSpace(ch.DeepStatusLabels[deepStatusVal]); mapped != "" {
			deepStatusVal = mapped
		}
	}
	var card7 string
	if deepStatusLabel != "" {
		card7 = box.Render(fmt.Sprintf("%s\n\n%s", muted(deepStatusLabel), deepStatusVal))
	}
	deepStageLabel := strings.TrimSpace(ch.LabelDeepStage)
	var card8 string
	if deepStageLabel != "" {
		pair := fmt.Sprintf("%d -> %d", m.stats.LastDeepStageBefore, m.stats.LastDeepStageAfter)
		if strings.TrimSpace(ch.LastRequestValueFmt) != "" && strings.Count(ch.LastRequestValueFmt, "%d") >= 2 {
			pair = fmt.Sprintf(ch.LastRequestValueFmt, m.stats.LastDeepStageBefore, m.stats.LastDeepStageAfter)
		}
		savedLabel := strings.TrimSpace(ch.LabelDeepStageSaved)
		if savedLabel == "" {
			card8 = box.Render(fmt.Sprintf("%s\n\n%s", muted(deepStageLabel), pair))
		} else {
			card8 = box.Render(fmt.Sprintf("%s\n\n%s\n%.1f%%", muted(deepStageLabel), pair, m.stats.LastDeepStageSaved))
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, card4, "  ", card5)
	if card6 != "" {
		row2 = lipgloss.JoinHorizontal(lipgloss.Top, row2, "  ", card6)
	}
	row3 := ""
	if card7 != "" || card8 != "" {
		parts := []string{}
		if card7 != "" {
			parts = append(parts, card7)
		}
		if card8 != "" {
			parts = append(parts, card8)
		}
		row3 = lipgloss.JoinHorizontal(lipgloss.Top, parts[0])
		for i := 1; i < len(parts); i++ {
			row3 = lipgloss.JoinHorizontal(lipgloss.Top, row3, "  ", parts[i])
		}
	}

	seriesEmpty := localchrome.ReplaceDays(ch.TUISeriesEmpty, daysStr)
	seriesTitle := localchrome.ReplaceDays(ch.TUISeriesTitle, daysStr)
	seriesBlock := muted(seriesEmpty)
	if len(m.series) > 0 {
		var b strings.Builder
		b.WriteString(muted(seriesTitle))
		b.WriteString("\n")
		start := 0
		if m.seriesDays > 0 && len(m.series) > m.seriesDays {
			start = len(m.series) - m.seriesDays
		}
		rowFmt := ch.TUISeriesRowFmt
		if rowFmt != "" {
			for _, p := range m.series[start:] {
				b.WriteString(fmt.Sprintf(rowFmt,
					p.Day, p.Requests, p.TokensBefore, p.TokensAfter, p.SavedUSD))
			}
		}
		seriesBlock = b.String()
	}

	cloudTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render(ch.TUICloudTitle)
	cloudBlock := muted(ch.TUICloudSignedOut)
	if m.apiBase != "" && m.token != "" {
		if m.cloudErr != "" {
			cloudBlock = lipgloss.NewStyle().Foreground(lipgloss.Color("167")).Render(
				strings.ReplaceAll(ch.TUICloudOfflineFmt, "{error}", m.truncateErr(m.cloudErr)),
			)
		} else if m.cloud != nil {
			c1 := box.Render(fmt.Sprintf("%s\n\n%s", muted(ch.TUICloudLabelTier), m.cloud.PlanTier))
			usedLabel := fmt.Sprintf("%d / %d", m.cloud.MonthlyCreditUsed, m.cloud.MonthlyCreditLimit)
			remainingLabel := fmt.Sprintf("%d", m.cloud.Remaining)
			if m.cloud.Unlimited {
				if lbl := strings.TrimSpace(m.cloud.UnlimitedLabel); lbl != "" {
					usedLabel = lbl
					remainingLabel = lbl
				} else {
					// Fail closed: no invent numeric remaining while unlimited chrome is missing.
					usedLabel = ""
					remainingLabel = ""
				}
			}
			c2 := box.Render(fmt.Sprintf("%s\n\n%s", muted(ch.TUICloudLabelUsed), usedLabel))
			c3 := box.Render(fmt.Sprintf("%s\n\n%s", muted(ch.TUICloudLabelRemaining), remainingLabel))
			c4 := box.Render(fmt.Sprintf("%s\n\n%d", muted(ch.TUICloudLabelTopup), m.cloud.PurchasedTopupCredits))
			cloudBlock = lipgloss.JoinHorizontal(lipgloss.Top, c1, "  ", c2, "  ", c3, "  ", c4)
			exhausted := !m.cloud.Unlimited && (m.cloud.Remaining <= 0 || m.cloud.Code == "QUOTA_EXHAUSTED" || m.cloud.Code == "WORKSPACE_QUOTA_EXHAUSTED")
			if exhausted {
				title := strings.TrimSpace(m.cloud.ExhaustedTitle)
				body := strings.TrimSpace(m.cloud.ExhaustedBody)
				url := strings.TrimSpace(m.cloud.TierUpgrade)
				var lines []string
				if title != "" {
					lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("167")).Bold(true).Render(title))
				}
				if body != "" {
					lines = append(lines, muted(body))
				}
				if url != "" {
					lines = append(lines, url)
				}
				if len(lines) > 0 {
					cloudBlock = cloudBlock + "\n\n" + strings.Join(lines, "\n")
				}
			}
		}
	}

	status := muted(strings.ReplaceAll(ch.TUIProxyURLFmt, "{port}", m.port))
	if m.errMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("167")).Render(
			strings.ReplaceAll(ch.TUIOfflineFmt, "{error}", m.truncateErr(m.errMsg)),
		)
	} else if !m.lastFetch.IsZero() {
		status = muted(ch.TUIUpdatedPrefix + m.lastFetch.Format(time.Kitchen))
	}

	footer := muted(ch.TUIFooter)
	hint := muted(ch.TUIHint)

	if row3 != "" {
		return fmt.Sprintf("\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n%s\n%s\n",
			title, row, row2, row3, seriesBlock, cloudTitle, cloudBlock, status, footer, hint)
	}
	return fmt.Sprintf("\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n%s\n%s\n",
		title, row, row2, seriesBlock, cloudTitle, cloudBlock, status, footer, hint)
}

func muted(s string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(s)
}

func (m model) truncateErr(s string) string {
	s = strings.TrimSpace(s)
	n := m.errorMaxChars
	if n < 1 || len(s) <= n {
		return s
	}
	suffix := strings.TrimSpace(m.chrome.TUITruncSuffix)
	if suffix == "" {
		return s[:n]
	}
	if len(suffix) >= n {
		return suffix[:n]
	}
	return s[:n-len(suffix)] + suffix
}

// Run starts the Bubble Tea dashboard. Blocks until quit.
// httpTimeoutSec comes from TRIM_CLI_HTTP_TIMEOUT_SEC (no invent 2s/3s).
// refreshSec comes from TRIM_TUI_REFRESH_SEC (no invent 2s).
// seriesDays comes from TRIM_TUI_SERIES_DAYS (no invent 7).
// errorMaxChars comes from TRIM_TUI_ERROR_MAX_CHARS (no invent 80).
func Run(port, apiBase, token, hmacSecret, cliVersion string, savingsUsdPerMTok float64, httpTimeoutSec, refreshSec, seriesDays, errorMaxChars int, chrome localchrome.Chrome) error {
	m, err := New(port, apiBase, token, hmacSecret, cliVersion, savingsUsdPerMTok, httpTimeoutSec, refreshSec, seriesDays, errorMaxChars, chrome)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
