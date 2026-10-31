package cli

import (
	"fmt"
	"math"
	"strings"

	"github.com/spf13/cobra"
	"github.com/usetrim/trim/cli/internal/cliout"
	"github.com/usetrim/trim/cli/internal/deepopt"
)

var configRatesCmd = &cobra.Command{
	Use:   "rates",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		chrome := loadCLIChrome()
		p, err := deepopt.LoadPreferences()
		if err != nil {
			return err
		}
		return printCompressionRatesFlow(p, chrome)
	},
}

// printCompressionRatesFlow matches trim start coloring:
// ● Success header → Table (cyan labels, accent/ok values) → Muted tip.
func printCompressionRatesFlow(p deepopt.Preferences, chrome cliChrome) error {
	title := strings.TrimSpace(chrome.ConfigRatesTitle)
	if title == "" {
		return chrome.fail("")
	}
	if !deepopt.RequireLiveDeepPolicy(p) || p.CompressionRateMin <= 0 || p.CompressionRateMax < p.CompressionRateMin {
		unsynced := strings.TrimSpace(chrome.ConfigRatesUnsynced)
		if unsynced != "" {
			cliout.Warn(unsynced)
			return nil
		}
		return chrome.fail("")
	}

	cliout.Success(title)

	defaultLab := strings.TrimSpace(chrome.ConfigRatesDefaultLabel)
	selectedLab := strings.TrimSpace(chrome.ConfigRatesSelectedLabel)
	selectedMark := strings.TrimSpace(chrome.ConfigRatesSelectedMark)
	minLab := strings.TrimSpace(chrome.ConfigRatesMinLabel)
	maxLab := strings.TrimSpace(chrome.ConfigRatesMaxLabel)
	stepLab := strings.TrimSpace(chrome.ConfigRatesStepLabel)
	if selectedMark == "" || minLab == "" || maxLab == "" || stepLab == "" {
		return chrome.fail("")
	}

	rows := make([]cliout.Row, 0, 4)
	if defaultLab != "" && p.DefaultCompressionRate > 0 {
		rows = append(rows, cliout.Row{
			Label: defaultLab,
			Value: formatRate(p.DefaultCompressionRate),
			Kind:  "accent",
		})
	}
	if selectedLab != "" && p.CompressionRate > 0 {
		rows = append(rows, cliout.Row{
			Label: selectedLab,
			Value: formatRate(p.CompressionRate),
			Kind:  "ok",
		})
	}
	rows = append(rows,
		cliout.Row{Label: minLab, Value: formatRate(p.CompressionRateMin)},
		cliout.Row{Label: maxLab, Value: formatRate(p.CompressionRateMax)},
		cliout.Row{Label: stepLab, Value: formatRate(p.CompressionRateStep)},
	)
	cliout.Table(rows)

	rates := deepopt.ListCompressionRates(p.CompressionRateMin, p.CompressionRateMax, p.CompressionRateStep)
	if len(rates) == 0 {
		return nil
	}
	listRows := make([]cliout.Row, 0, len(rates))
	for _, r := range rates {
		label := formatRate(r)
		value := ""
		kind := ""
		isSelected := nearlyEqualRate(r, p.CompressionRate)
		isDefault := nearlyEqualRate(r, p.DefaultCompressionRate)
		switch {
		case isSelected:
			value = selectedMark
			kind = "ok"
		case isDefault:
			value = defaultLab
			kind = "accent"
		}
		listRows = append(listRows, cliout.Row{Label: label, Value: value, Kind: kind})
	}
	cliout.Table(listRows)

	if tip := strings.TrimSpace(chrome.ConfigRatesSyncHint); tip != "" {
		cliout.Muted(tip)
	}
	return nil
}

func formatRate(v float64) string {
	return fmt.Sprintf("%.2g", v)
}

func nearlyEqualRate(a, b float64) bool {
	return math.Abs(a-b) < 0.0005
}

func init() {
	configCmd.AddCommand(configRatesCmd)
}
